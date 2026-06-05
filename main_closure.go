package main

import (
	"context"
	"fmt"
	"time"

	"git.commsnet.org/commstech/bugbot/analyzers"
	"git.commsnet.org/commstech/bugbot/closure"
	"git.commsnet.org/commstech/bugbot/issues"
	"git.commsnet.org/commstech/bugbot/notify"
	"git.commsnet.org/commstech/bugbot/store"
	"github.com/gin-gonic/gin"
)

var closureEngine *closure.Engine

func initClosureEngine() {
	cfg := closure.Config{
		Enabled:               config.EvidenceClosureEnabled,
		CloseIssues:           config.EvidenceClosureCloseIssues,
		Comment:               config.EvidenceClosureComment,
		RequireScannerSuccess: config.EvidenceClosureRequireScannerSuccess,
	}
	closureEngine = &closure.Engine{
		Config: cfg,
		Store:  closureStoreAdapter{},
		PR:     giteaClient,
		Issues: closureIssueBridge{},
		Notify: closureNotifyBridge{},
	}
}

type closureStoreAdapter struct{}

func (closureStoreAdapter) ListPatchAttemptsByRepositoryAndStatus(ctx context.Context, repositoryID int64, status string) ([]closure.PatchAttemptRow, error) {
	recs, err := bugbotStore.ListPatchAttemptsByRepositoryAndStatus(ctx, repositoryID, status)
	if err != nil {
		return nil, err
	}
	out := make([]closure.PatchAttemptRow, 0, len(recs))
	for _, rec := range recs {
		out = append(out, patchAttemptToClosureRow(ctx, rec))
	}
	return out, nil
}

func (closureStoreAdapter) GetPatchAttemptForClosure(ctx context.Context, attemptID string) (closure.PatchAttemptRow, error) {
	rec, finding, err := bugbotStore.GetPatchAttemptForClosure(ctx, attemptID)
	if err != nil {
		return closure.PatchAttemptRow{}, err
	}
	row := patchAttemptToClosureRow(ctx, rec)
	if finding.Source != "" {
		row.OriginalSource = finding.Source
	}
	if finding.Fingerprint != "" {
		row.Fingerprint = finding.Fingerprint
	}
	return row, nil
}

func patchAttemptToClosureRow(ctx context.Context, rec store.PatchAttemptRecord) closure.PatchAttemptRow {
	row := closure.PatchAttemptRow{
		AttemptID:      rec.AttemptID,
		PlanID:         rec.PlanID,
		RepositoryID:   rec.RepositoryID,
		Status:         rec.Status,
		MergeCommitSHA: rec.MergeCommitSHA,
		MergedAt:       rec.MergedAt,
	}
	if rec.FindingID != nil {
		row.FindingID = *rec.FindingID
		if finding, err := bugbotStore.GetFindingDetail(ctx, *rec.FindingID); err == nil {
			row.Fingerprint = finding.Fingerprint
			row.OriginalSource = finding.Source
		}
	}
	if rec.PullRequestNumber != nil {
		row.PullRequestNumber = *rec.PullRequestNumber
	}
	return row
}

func (closureStoreAdapter) UpdatePatchAttemptMerged(ctx context.Context, attemptID, mergeSHA string, mergedAt time.Time) error {
	return bugbotStore.UpdatePatchAttemptMerged(ctx, attemptID, mergeSHA, mergedAt)
}

func (closureStoreAdapter) GetLatestClosureEvidenceByFindingID(ctx context.Context, findingID int64) (closure.EvidenceRow, error) {
	rec, err := bugbotStore.GetLatestClosureEvidenceByFindingID(ctx, findingID)
	if err != nil {
		return closure.EvidenceRow{}, err
	}
	return closureEvidenceToRow(rec), nil
}

func (closureStoreAdapter) SaveClosureEvidence(ctx context.Context, row closure.EvidenceRow) (closure.EvidenceRow, error) {
	rec, err := bugbotStore.SaveClosureEvidence(ctx, closureEvidenceFromRow(row))
	if err != nil {
		return closure.EvidenceRow{}, err
	}
	return closureEvidenceToRow(rec), nil
}

func (closureStoreAdapter) UpdateClosureEvidence(ctx context.Context, row closure.EvidenceRow) error {
	return bugbotStore.UpdateClosureEvidence(ctx, closureEvidenceFromRow(row))
}

func (closureStoreAdapter) ListClosureEvidenceByRepositoryAndStatus(ctx context.Context, repositoryID int64, status string) ([]closure.EvidenceRow, error) {
	recs, err := bugbotStore.ListClosureEvidenceByRepositoryAndStatus(ctx, repositoryID, status)
	if err != nil {
		return nil, err
	}
	out := make([]closure.EvidenceRow, 0, len(recs))
	for _, rec := range recs {
		out = append(out, closureEvidenceToRow(rec))
	}
	return out, nil
}

func (closureStoreAdapter) GetFindingByID(ctx context.Context, findingID int64) (closure.FindingRow, error) {
	detail, err := bugbotStore.GetFindingDetail(ctx, findingID)
	if err != nil {
		return closure.FindingRow{}, err
	}
	return closure.FindingRow{ID: detail.ID, RepositoryID: detail.RepositoryID, Fingerprint: detail.Fingerprint, Source: detail.Source, Status: detail.Status}, nil
}

func (closureStoreAdapter) UpdateFindingStatus(ctx context.Context, findingID int64, status string) error {
	return bugbotStore.UpdateFindingStatus(ctx, findingID, status)
}

func (closureStoreAdapter) AddLifecycleEvent(ctx context.Context, findingID int64, scanID, eventType, message string) error {
	fid := findingID
	return bugbotStore.AddLifecycleEvent(ctx, store.LifecycleEvent{
		FindingID: &fid, ScanID: scanID, EventType: eventType, Message: message,
	})
}

func (closureStoreAdapter) ListExternalIssuesByFinding(ctx context.Context, findingID int64) ([]closure.ExternalIssueRow, error) {
	recs, err := bugbotStore.ListExternalIssuesByFinding(ctx, findingID)
	if err != nil {
		return nil, err
	}
	out := make([]closure.ExternalIssueRow, 0, len(recs))
	for _, rec := range recs {
		out = append(out, closure.ExternalIssueRow{IssueNumber: rec.IssueNumber})
	}
	return out, nil
}

func (closureStoreAdapter) GetRepository(ctx context.Context, repositoryID int64) (closure.RepositoryRow, error) {
	repo, err := bugbotStore.GetRepository(ctx, repositoryID)
	if err != nil {
		return closure.RepositoryRow{}, err
	}
	return closure.RepositoryRow{ID: repo.ID, Owner: repo.Owner, Name: repo.Name, FullName: repo.FullName}, nil
}

func closureEvidenceFromRow(row closure.EvidenceRow) store.ClosureEvidenceRecord {
	return store.ClosureEvidenceRecord{
		ID: row.ID, FindingID: row.FindingID, PatchAttemptID: row.PatchAttemptID,
		RepositoryID: row.RepositoryID, Fingerprint: row.Fingerprint,
		MergeCommitSHA: row.MergeCommitSHA, VerificationScanID: row.VerificationScanID,
		OriginalSource: row.OriginalSource, ScannerStatus: row.ScannerStatus,
		FingerprintPresent: row.FingerprintPresent, Status: row.Status, Reason: row.Reason,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func closureEvidenceToRow(rec store.ClosureEvidenceRecord) closure.EvidenceRow {
	return closure.EvidenceRow{
		ID: rec.ID, FindingID: rec.FindingID, PatchAttemptID: rec.PatchAttemptID,
		RepositoryID: rec.RepositoryID, Fingerprint: rec.Fingerprint,
		MergeCommitSHA: rec.MergeCommitSHA, VerificationScanID: rec.VerificationScanID,
		OriginalSource: rec.OriginalSource, ScannerStatus: rec.ScannerStatus,
		FingerprintPresent: rec.FingerprintPresent, Status: rec.Status, Reason: rec.Reason,
		CreatedAt: rec.CreatedAt, UpdatedAt: rec.UpdatedAt,
	}
}

type closureIssueBridge struct{}

func (closureIssueBridge) CommentIssue(ctx context.Context, owner, repo string, issueNumber int, body string) error {
	if giteaClient == nil {
		return fmt.Errorf("gitea unavailable")
	}
	return giteaClient.CreateIssueComment(ctx, owner, repo, issueNumber, body)
}

func (closureIssueBridge) AddLifecycleLabels(ctx context.Context, owner, repo string, issueNumber int, lifecycleLabel string) error {
	if giteaClient == nil {
		return fmt.Errorf("gitea unavailable")
	}
	_, err := giteaClient.AddIssueLabels(ctx, owner, repo, issueNumber, issues.ExpandLifecycleLabel(lifecycleLabel))
	return err
}

func (closureIssueBridge) CloseIssue(ctx context.Context, owner, repo string, issueNumber int) error {
	if giteaClient == nil {
		return fmt.Errorf("gitea unavailable")
	}
	return giteaClient.CloseIssue(ctx, owner, repo, issueNumber)
}

type closureNotifyBridge struct{}

func (closureNotifyBridge) Emit(ctx context.Context, repositoryID int64, eventType, severity, repoFull, scanID, title, summary string) {
	if notifyManager == nil || !config.NotificationsEnabled {
		return
	}
	notifyManager.Emit(ctx, repositoryID, notify.Event{
		Type: eventType, Severity: severity, Repository: repoFull,
		ScanID: scanID, Title: title, Summary: summary,
	})
}

func maybeProcessEvidenceClosure(ctx context.Context, owner, repo string, repositoryID int64, result *analyzers.AnalysisResult) {
	if closureEngine == nil || !config.EvidenceClosureEnabled || result == nil || result.ScanID == "" {
		return
	}
	scanCtx := buildClosureScanContext(ctx, owner, repo, repositoryID, result)
	_ = closureEngine.OnScanFinish(ctx, scanCtx)
}

func buildClosureScanContext(ctx context.Context, owner, repo string, repositoryID int64, result *analyzers.AnalysisResult) closure.ScanContext {
	fps := make(map[string]struct{}, len(result.Issues))
	for _, issue := range result.Issues {
		if issue.Fingerprint != "" {
			fps[issue.Fingerprint] = struct{}{}
		}
	}
	scannerMap := map[string]string{}
	if bugbotStore != nil && result.ScanID != "" {
		if recs, err := bugbotStore.ListScannerResultsByScan(ctx, result.ScanID); err == nil {
			for _, sr := range recs {
				scannerMap[sr.ScannerName] = sr.Status
			}
		}
	}
	if len(scannerMap) == 0 {
		for _, sr := range result.ScannerResults {
			scannerMap[sr.Scanner] = string(sr.Status)
		}
	}
	return closure.ScanContext{
		ScanID:           result.ScanID,
		RepositoryID:     repositoryID,
		Owner:            owner,
		Repo:             repo,
		FingerprintsSeen: fps,
		ScannerResults:   scannerMap,
	}
}

func verifyFindingClosure(ctx context.Context, findingID int64) (closure.Evidence, error) {
	if closureEngine == nil || bugbotStore == nil {
		return closure.Evidence{}, fmt.Errorf("evidence closure disabled")
	}
	detail, err := bugbotStore.GetFindingDetail(ctx, findingID)
	if err != nil {
		return closure.Evidence{}, err
	}
	repo, err := bugbotStore.GetRepository(ctx, detail.RepositoryID)
	if err != nil {
		return closure.Evidence{}, err
	}
	scanID, err := latestCompletedScanID(ctx, detail.RepositoryID)
	if err != nil {
		return closure.Evidence{}, err
	}
	scanCtx, err := buildClosureScanContextFromStore(ctx, repo.Owner, repo.Name, detail.RepositoryID, scanID)
	if err != nil {
		return closure.Evidence{}, err
	}
	return closureEngine.VerifyFindingClosure(ctx, findingID, scanCtx)
}

func latestCompletedScanID(ctx context.Context, repositoryID int64) (string, error) {
	scans, err := bugbotStore.ListScansByRepository(ctx, repositoryID, store.ListOptions{Limit: 10})
	if err != nil {
		return "", err
	}
	for _, sc := range scans {
		if sc.Status == store.ScanStatusCompleted {
			return sc.ID, nil
		}
	}
	return "", fmt.Errorf("no completed scan available for verification")
}

func buildClosureScanContextFromStore(ctx context.Context, owner, repo string, repositoryID int64, scanID string) (closure.ScanContext, error) {
	if scanID == "" {
		return closure.ScanContext{}, fmt.Errorf("scan id required")
	}
	seen := map[string]struct{}{}
	if fps, err := bugbotStore.ListFingerprintsInScan(ctx, scanID, repositoryID); err == nil {
		for fp, present := range fps {
			if present {
				seen[fp] = struct{}{}
			}
		}
	}
	scannerMap := map[string]string{}
	if recs, err := bugbotStore.ListScannerResultsByScan(ctx, scanID); err == nil {
		for _, sr := range recs {
			scannerMap[sr.ScannerName] = sr.Status
		}
	}
	return closure.ScanContext{
		ScanID:           scanID,
		RepositoryID:     repositoryID,
		Owner:            owner,
		Repo:             repo,
		FingerprintsSeen: seen,
		ScannerResults:   scannerMap,
	}, nil
}

func recordDirectRemediation(ctx context.Context, findingID int64, mergeCommitSHA, reason string) (closure.Evidence, error) {
	if closureEngine == nil || bugbotStore == nil {
		return closure.Evidence{}, fmt.Errorf("evidence closure disabled")
	}
	detail, err := bugbotStore.GetFindingDetail(ctx, findingID)
	if err != nil {
		return closure.Evidence{}, err
	}
	return closureEngine.RecordDirectRemediation(ctx, closure.DirectRemediationInput{
		FindingID:      findingID,
		RepositoryID:   detail.RepositoryID,
		Fingerprint:    detail.Fingerprint,
		OriginalSource: detail.Source,
		MergeCommitSHA: mergeCommitSHA,
		Reason:         reason,
	})
}

type closureBridge struct{}

func (closureBridge) GetClosureEvidence(c *gin.Context, findingID int64) (closure.Evidence, error) {
	if closureEngine == nil {
		return closure.Evidence{}, fmt.Errorf("evidence closure disabled")
	}
	ev, err := closureEngine.GetEvidence(c.Request.Context(), findingID)
	if err != nil {
		return closure.Evidence{}, fmt.Errorf("no closure evidence found")
	}
	if ev.Status == "" && ev.FindingID == 0 {
		return closure.Evidence{FindingID: findingID, Status: closure.StatusPendingRescan, Reason: "not verified yet"}, nil
	}
	return ev, nil
}

func (closureBridge) VerifyClosure(c *gin.Context, findingID int64) (closure.Evidence, error) {
	return verifyFindingClosure(c.Request.Context(), findingID)
}

func (closureBridge) RecordDirectRemediation(c *gin.Context, findingID int64, mergeCommitSHA, reason string) (closure.Evidence, error) {
	return recordDirectRemediation(c.Request.Context(), findingID, mergeCommitSHA, reason)
}

func (closureBridge) CheckPatchAttemptMerge(c *gin.Context, attemptID string) (closure.Evidence, error) {
	if closureEngine == nil {
		return closure.Evidence{}, fmt.Errorf("evidence closure disabled")
	}
	return closureEngine.CheckPatchAttemptMerge(c.Request.Context(), attemptID)
}

type closureUIBridge struct{}

func (closureUIBridge) GetClosureEvidence(ctx context.Context, findingID int64) (closure.Evidence, error) {
	if closureEngine == nil {
		return closure.Evidence{}, fmt.Errorf("evidence closure disabled")
	}
	return closureEngine.GetEvidence(ctx, findingID)
}

func (closureUIBridge) VerifyClosure(ctx context.Context, findingID int64) (closure.Evidence, error) {
	return verifyFindingClosure(ctx, findingID)
}

func (closureUIBridge) CheckPatchAttemptMerge(ctx context.Context, attemptID string) (closure.Evidence, error) {
	if closureEngine == nil {
		return closure.Evidence{}, fmt.Errorf("evidence closure disabled")
	}
	return closureEngine.CheckPatchAttemptMerge(ctx, attemptID)
}

func markFixPROpened(ctx context.Context, owner, repo string, issueNumber int) {
	if !config.EvidenceClosureComment || giteaClient == nil || issueNumber <= 0 {
		return
	}
	_, _ = giteaClient.AddIssueLabels(ctx, owner, repo, issueNumber, issues.ExpandLifecycleLabel(issues.LifecycleFixPROpened))
}
