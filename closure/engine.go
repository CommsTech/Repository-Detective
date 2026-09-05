package closure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/issues"
)

// IssueActions applies Gitea issue updates for closure workflow.
type IssueActions interface {
	CommentIssue(ctx context.Context, owner, repo string, issueNumber int, body string) error
	AddLifecycleLabels(ctx context.Context, owner, repo string, issueNumber int, lifecycleLabel string) error
	CloseIssue(ctx context.Context, owner, repo string, issueNumber int) error
}

// Store abstracts persistence for closure workflow.
type Store interface {
	GetPatchAttemptForClosure(ctx context.Context, attemptID string) (PatchAttemptRow, error)
	ListPatchAttemptsByRepositoryAndStatus(ctx context.Context, repositoryID int64, status string) ([]PatchAttemptRow, error)
	UpdatePatchAttemptMerged(ctx context.Context, attemptID string, mergeSHA string, mergedAt time.Time) error
	GetLatestClosureEvidenceByFindingID(ctx context.Context, findingID int64) (EvidenceRow, error)
	SaveClosureEvidence(ctx context.Context, row EvidenceRow) (EvidenceRow, error)
	UpdateClosureEvidence(ctx context.Context, row EvidenceRow) error
	ListClosureEvidenceByRepositoryAndStatus(ctx context.Context, repositoryID int64, status string) ([]EvidenceRow, error)
	GetFindingByID(ctx context.Context, findingID int64) (FindingRow, error)
	UpdateFindingStatus(ctx context.Context, findingID int64, status string) error
	AddLifecycleEvent(ctx context.Context, findingID int64, scanID, eventType, message string) error
	ListExternalIssuesByFinding(ctx context.Context, findingID int64) ([]ExternalIssueRow, error)
	GetRepository(ctx context.Context, repositoryID int64) (RepositoryRow, error)
}

// Notifier emits closure notification events.
type Notifier interface {
	Emit(ctx context.Context, repositoryID int64, eventType, severity, repoFull, scanID, title, summary string)
}

// PatchAttemptRow is a minimal patch attempt record for closure.
type PatchAttemptRow struct {
	AttemptID         string
	PlanID            string
	FindingID         int64
	RepositoryID      int64
	Fingerprint       string
	OriginalSource    string
	Status            string
	PullRequestNumber int
	MergeCommitSHA    string
	MergedAt          *time.Time
}

// EvidenceRow is a persistence row for closure evidence.
type EvidenceRow struct {
	ID                 int64
	FindingID          int64
	PatchAttemptID     string
	RepositoryID       int64
	Fingerprint        string
	MergeCommitSHA     string
	VerificationScanID string
	OriginalSource     string
	ScannerStatus      string
	FingerprintPresent bool
	Status             string
	Reason             string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// FindingRow is a minimal finding record.
type FindingRow struct {
	ID           int64
	RepositoryID int64
	Fingerprint  string
	Source       string
	Status       string
}

// RepositoryRow is minimal repo metadata.
type RepositoryRow struct {
	ID       int64
	Owner    string
	Name     string
	FullName string
}

// ExternalIssueRow links a finding to a forge issue.
type ExternalIssueRow struct {
	IssueNumber int
}

// Engine orchestrates evidence-based closure.
type Engine struct {
	Config Config
	Store  Store
	PR     PRClient
	Issues IssueActions
	Notify Notifier
}

// OnScanFinish checks PR merges and verifies pending closure evidence after a scan.
func (e *Engine) OnScanFinish(ctx context.Context, scan ScanContext) error {
	if e == nil || !e.Config.Enabled || e.Store == nil {
		return nil
	}
	if err := e.checkOpenPRs(ctx, scan); err != nil {
		return err
	}
	if err := e.verifyPending(ctx, scan); err != nil {
		return err
	}
	return nil
}

// CheckPatchAttemptMerge queries Gitea and updates state for one patch attempt.
func (e *Engine) CheckPatchAttemptMerge(ctx context.Context, attemptID string) (Evidence, error) {
	if e == nil || !e.Config.Enabled || e.Store == nil {
		return Evidence{}, fmt.Errorf("evidence closure disabled")
	}
	attempt, err := e.Store.GetPatchAttemptForClosure(ctx, attemptID)
	if err != nil {
		return Evidence{}, err
	}
	return e.detectAndRecordMerge(ctx, attempt)
}

// DirectRemediationInput records closure evidence for a fix merged outside the PR workflow.
type DirectRemediationInput struct {
	FindingID      int64
	RepositoryID   int64
	Fingerprint    string
	OriginalSource string
	MergeCommitSHA string
	Reason         string
}

// RecordDirectRemediation persists closure evidence for a direct-to-main remediation.
// MergeCommitSHA must be set so Verify treats the remediation as merged (PRMerged=true).
func (e *Engine) RecordDirectRemediation(ctx context.Context, in DirectRemediationInput) (Evidence, error) {
	if e == nil || !e.Config.Enabled || e.Store == nil {
		return Evidence{}, fmt.Errorf("evidence closure disabled")
	}
	if in.FindingID <= 0 || in.RepositoryID <= 0 {
		return Evidence{}, fmt.Errorf("finding_id and repository_id are required")
	}
	mergeSHA := strings.TrimSpace(in.MergeCommitSHA)
	if mergeSHA == "" {
		return Evidence{}, fmt.Errorf("merge_commit_sha is required")
	}
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		reason = "Remediation merged directly to default branch"
	}
	row := EvidenceRow{
		FindingID:      in.FindingID,
		RepositoryID:   in.RepositoryID,
		Fingerprint:    strings.TrimSpace(in.Fingerprint),
		MergeCommitSHA: mergeSHA,
		OriginalSource: strings.TrimSpace(in.OriginalSource),
		Status:         StatusPendingRescan,
		Reason:         reason,
	}
	saved, err := e.Store.SaveClosureEvidence(ctx, row)
	if err != nil {
		return Evidence{}, err
	}
	return rowToEvidence(saved), nil
}

// VerifyFindingClosure verifies closure for a finding using the latest scan context.
// When no prior closure evidence exists, a direct-scan verification record is created.
func (e *Engine) VerifyFindingClosure(ctx context.Context, findingID int64, scan ScanContext) (Evidence, error) {
	if e == nil || !e.Config.Enabled || e.Store == nil {
		return Evidence{}, fmt.Errorf("evidence closure disabled")
	}
	row, err := e.ensureClosureEvidenceRow(ctx, findingID, scan.RepositoryID)
	if err != nil {
		return Evidence{}, err
	}
	return e.applyVerification(ctx, row, scan)
}

func (e *Engine) ensureClosureEvidenceRow(ctx context.Context, findingID, repositoryID int64) (EvidenceRow, error) {
	row, err := e.Store.GetLatestClosureEvidenceByFindingID(ctx, findingID)
	if err == nil {
		return row, nil
	}
	if !IsEvidenceNotFound(err) {
		return EvidenceRow{}, err
	}
	finding, err := e.Store.GetFindingByID(ctx, findingID)
	if err != nil {
		return EvidenceRow{}, err
	}
	repoID := repositoryID
	if repoID <= 0 {
		repoID = finding.RepositoryID
	}
	row = EvidenceRow{
		FindingID:      findingID,
		RepositoryID:   repoID,
		Fingerprint:    finding.Fingerprint,
		OriginalSource: finding.Source,
		MergeCommitSHA: directScanMergeMarker,
		Status:         StatusPendingRescan,
		Reason:         "Direct verification against completed scan",
	}
	return e.Store.SaveClosureEvidence(ctx, row)
}

// GetEvidence returns latest closure evidence for a finding.
func (e *Engine) GetEvidence(ctx context.Context, findingID int64) (Evidence, error) {
	row, err := e.Store.GetLatestClosureEvidenceByFindingID(ctx, findingID)
	if err != nil {
		return Evidence{}, err
	}
	return rowToEvidence(row), nil
}

func (e *Engine) checkOpenPRs(ctx context.Context, scan ScanContext) error {
	attempts, err := e.Store.ListPatchAttemptsByRepositoryAndStatus(ctx, scan.RepositoryID, "pr_opened")
	if err != nil {
		return err
	}
	for _, attempt := range attempts {
		if _, err := e.detectAndRecordMerge(ctx, attempt); err != nil {
			// non-fatal per attempt
			continue
		}
	}
	return nil
}

func (e *Engine) detectAndRecordMerge(ctx context.Context, attempt PatchAttemptRow) (Evidence, error) {
	if attempt.Status == "pr_merged" {
		return Evidence{}, nil
	}
	if attempt.PullRequestNumber <= 0 {
		return Evidence{}, nil
	}
	repo, err := e.Store.GetRepository(ctx, attempt.RepositoryID)
	if err != nil {
		return Evidence{}, err
	}
	info, err := DetectMerge(ctx, e.PR, repo.Owner, repo.Name, attempt.PullRequestNumber)
	if err != nil || !info.Merged {
		return Evidence{}, err
	}
	if err := e.Store.UpdatePatchAttemptMerged(ctx, attempt.AttemptID, info.MergeCommitSHA, info.MergedAt); err != nil {
		return Evidence{}, err
	}
	row := EvidenceRow{
		FindingID:      attempt.FindingID,
		PatchAttemptID: attempt.AttemptID,
		RepositoryID:   attempt.RepositoryID,
		Fingerprint:    attempt.Fingerprint,
		MergeCommitSHA: info.MergeCommitSHA,
		OriginalSource: attempt.OriginalSource,
		Status:         StatusPendingRescan,
		Reason:         "PR merged; waiting for verification scan",
	}
	saved, err := e.Store.SaveClosureEvidence(ctx, row)
	if err != nil {
		return Evidence{}, err
	}
	e.onPRMerged(ctx, attempt, repo)
	return rowToEvidence(saved), nil
}

func (e *Engine) verifyPending(ctx context.Context, scan ScanContext) error {
	rows, err := e.Store.ListClosureEvidenceByRepositoryAndStatus(ctx, scan.RepositoryID, StatusPendingRescan)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := e.applyVerification(ctx, row, scan); err != nil {
			continue
		}
	}
	return nil
}

func (e *Engine) applyVerification(ctx context.Context, row EvidenceRow, scan ScanContext) (Evidence, error) {
	prMerged := row.MergeCommitSHA != "" || row.PatchAttemptID != ""
	result := Verify(VerifyInput{
		Evidence:       rowToEvidence(row),
		Scan:           scan,
		RequireScanner: e.Config.RequireScannerSuccess,
		PRMerged:       prMerged,
	})
	row.VerificationScanID = scan.ScanID
	row.ScannerStatus = result.ScannerStatus
	row.FingerprintPresent = result.FingerprintPresent
	row.Status = result.Status
	row.Reason = result.Reason
	row.UpdatedAt = time.Now().UTC()
	if err := e.Store.UpdateClosureEvidence(ctx, row); err != nil {
		return Evidence{}, err
	}

	repo, _ := e.Store.GetRepository(ctx, row.RepositoryID)
	e.applyOutcome(ctx, row, result, repo, scan.ScanID)
	return rowToEvidence(row), nil
}

func (e *Engine) applyOutcome(ctx context.Context, row EvidenceRow, result VerifyResult, repo RepositoryRow, scanID string) {
	findingStatus := storeFindingStatus(result.Status)
	if findingStatus != "" {
		_ = e.Store.UpdateFindingStatus(ctx, row.FindingID, findingStatus)
	}
	_ = e.Store.AddLifecycleEvent(ctx, row.FindingID, scanID, "closure_"+result.Status, result.Reason)

	ext, _ := e.Store.ListExternalIssuesByFinding(ctx, row.FindingID)
	issueNum := 0
	if len(ext) > 0 {
		issueNum = ext[0].IssueNumber
	}

	if e.Issues != nil && issueNum > 0 && e.Config.Comment {
		e.commentAndLabel(ctx, repo, issueNum, row, result, scanID)
	}

	if e.Notify != nil {
		e.emitNotification(ctx, row, result, repo, scanID)
	}
}

func (e *Engine) commentAndLabel(ctx context.Context, repo RepositoryRow, issueNum int, row EvidenceRow, result VerifyResult, scanID string) {
	scanner := ScannerForSource(row.OriginalSource)
	var comment string
	var label string
	switch result.Status {
	case StatusVerified:
		comment = VerifiedComment(scanID, row.Fingerprint, scanner)
		label = issues.LifecycleResolvedVerified
		if e.Config.CloseIssues {
			if err := e.Issues.CloseIssue(ctx, repo.Owner, repo.Name, issueNum); err != nil {
				_ = e.Store.AddLifecycleEvent(ctx, row.FindingID, scanID, EventClosureIssueCloseFailed,
					fmt.Sprintf("close issue #%d: %v", issueNum, err))
			}
		}
	case StatusBlocked:
		comment = BlockedComment(scanner, scanID)
		label = issues.LifecycleClosureBlocked
	case StatusStillPresent:
		comment = StillPresentComment(row.Fingerprint, scanID)
		label = issues.LifecycleStillPresent
	default:
		return
	}
	_ = e.Issues.CommentIssue(ctx, repo.Owner, repo.Name, issueNum, comment)
	if label != "" {
		if err := e.Issues.AddLifecycleLabels(ctx, repo.Owner, repo.Name, issueNum, label); err != nil {
			_ = e.Store.AddLifecycleEvent(ctx, row.FindingID, scanID, EventClosureIssueCloseFailed,
				fmt.Sprintf("add lifecycle label on #%d: %v", issueNum, err))
		}
	}
}

func (e *Engine) onPRMerged(ctx context.Context, attempt PatchAttemptRow, repo RepositoryRow) {
	_ = e.Store.UpdateFindingStatus(ctx, attempt.FindingID, "pending_rescan")
	_ = e.Store.AddLifecycleEvent(ctx, attempt.FindingID, "", EventFixPRMerged, "Remediation PR merged; pending rescan")
	ext, _ := e.Store.ListExternalIssuesByFinding(ctx, attempt.FindingID)
	if e.Issues != nil && len(ext) > 0 && e.Config.Comment {
		_ = e.Issues.CommentIssue(ctx, repo.Owner, repo.Name, ext[0].IssueNumber, PRMergedComment())
		if err := e.Issues.AddLifecycleLabels(ctx, repo.Owner, repo.Name, ext[0].IssueNumber, issues.LifecycleFixPRMerged); err != nil {
			_ = e.Store.AddLifecycleEvent(ctx, attempt.FindingID, "", EventClosureIssueCloseFailed,
				fmt.Sprintf("add fix-pr-merged label on #%d: %v", ext[0].IssueNumber, err))
		}
		if err := e.Issues.AddLifecycleLabels(ctx, repo.Owner, repo.Name, ext[0].IssueNumber, issues.LifecyclePendingRescan); err != nil {
			_ = e.Store.AddLifecycleEvent(ctx, attempt.FindingID, "", EventClosureIssueCloseFailed,
				fmt.Sprintf("add pending-rescan label on #%d: %v", ext[0].IssueNumber, err))
		}
	}
	if e.Notify != nil {
		e.Notify.Emit(ctx, attempt.RepositoryID, EventFixPRMerged, "info", repo.FullName, "", "Remediation PR merged", "Waiting for verification scan")
	}
}

func (e *Engine) emitNotification(ctx context.Context, row EvidenceRow, result VerifyResult, repo RepositoryRow, scanID string) {
	switch result.Status {
	case StatusVerified:
		e.Notify.Emit(ctx, row.RepositoryID, EventClosureVerified, "info", repo.FullName, scanID, "Closure verified", result.Reason)
	case StatusBlocked:
		e.Notify.Emit(ctx, row.RepositoryID, EventClosureBlocked, "medium", repo.FullName, scanID, "Closure blocked", result.Reason)
	case StatusStillPresent:
		e.Notify.Emit(ctx, row.RepositoryID, EventRemediationStillPresent, "medium", repo.FullName, scanID, "Finding still present", result.Reason)
	}
}

func storeFindingStatus(closureStatus string) string {
	switch closureStatus {
	case StatusVerified:
		return "resolved_verified"
	case StatusStillPresent:
		return "still_present"
	case StatusBlocked:
		return "closure_blocked"
	case StatusPendingRescan:
		return "pending_rescan"
	default:
		return ""
	}
}

func rowToEvidence(row EvidenceRow) Evidence {
	return Evidence{
		ID:                 row.ID,
		FindingID:          row.FindingID,
		PatchAttemptID:     row.PatchAttemptID,
		RepositoryID:       row.RepositoryID,
		Fingerprint:        row.Fingerprint,
		MergeCommitSHA:     row.MergeCommitSHA,
		VerificationScanID: row.VerificationScanID,
		OriginalSource:     row.OriginalSource,
		ScannerStatus:      row.ScannerStatus,
		FingerprintPresent: row.FingerprintPresent,
		Status:             row.Status,
		Reason:             row.Reason,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}
