package e2e_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/closure"
	"git.commsnet.org/commstech/repository-detective/gitea"
	"git.commsnet.org/commstech/repository-detective/issues"
	"git.commsnet.org/commstech/repository-detective/remediation"
	"git.commsnet.org/commstech/repository-detective/store"
)

// End-to-end lifecycle tests using SQLite + fakes (no live Gitea).

type workflowStore struct {
	*store.SQLiteStore
}

func (w workflowStore) ListPatchAttemptsByRepositoryAndStatus(ctx context.Context, repositoryID int64, status string) ([]closure.PatchAttemptRow, error) {
	recs, err := w.SQLiteStore.ListPatchAttemptsByRepositoryAndStatus(ctx, repositoryID, status)
	if err != nil {
		return nil, err
	}
	out := make([]closure.PatchAttemptRow, 0, len(recs))
	for _, rec := range recs {
		out = append(out, patchToRow(ctx, w.SQLiteStore, rec))
	}
	return out, nil
}

func (w workflowStore) GetPatchAttemptForClosure(ctx context.Context, attemptID string) (closure.PatchAttemptRow, error) {
	rec, finding, err := w.SQLiteStore.GetPatchAttemptForClosure(ctx, attemptID)
	if err != nil {
		return closure.PatchAttemptRow{}, err
	}
	row := patchToRow(ctx, w.SQLiteStore, rec)
	if finding.Source != "" {
		row.OriginalSource = finding.Source
	}
	if finding.Fingerprint != "" {
		row.Fingerprint = finding.Fingerprint
	}
	return row, nil
}

func patchToRow(ctx context.Context, s *store.SQLiteStore, rec store.PatchAttemptRecord) closure.PatchAttemptRow {
	row := closure.PatchAttemptRow{
		AttemptID: rec.AttemptID, PlanID: rec.PlanID, RepositoryID: rec.RepositoryID,
		Status: rec.Status, MergeCommitSHA: rec.MergeCommitSHA, MergedAt: rec.MergedAt,
	}
	if rec.FindingID != nil {
		row.FindingID = *rec.FindingID
		if f, err := s.GetFindingDetail(ctx, *rec.FindingID); err == nil {
			row.Fingerprint = f.Fingerprint
			row.OriginalSource = f.Source
		}
	}
	if rec.PullRequestNumber != nil {
		row.PullRequestNumber = *rec.PullRequestNumber
	}
	return row
}

func (w workflowStore) UpdatePatchAttemptMerged(ctx context.Context, attemptID, mergeSHA string, mergedAt time.Time) error {
	return w.SQLiteStore.UpdatePatchAttemptMerged(ctx, attemptID, mergeSHA, mergedAt)
}

func (w workflowStore) GetLatestClosureEvidenceByFindingID(ctx context.Context, findingID int64) (closure.EvidenceRow, error) {
	rec, err := w.SQLiteStore.GetLatestClosureEvidenceByFindingID(ctx, findingID)
	if err != nil {
		return closure.EvidenceRow{}, err
	}
	return evidenceToRow(rec), nil
}

func (w workflowStore) SaveClosureEvidence(ctx context.Context, row closure.EvidenceRow) (closure.EvidenceRow, error) {
	rec, err := w.SQLiteStore.SaveClosureEvidence(ctx, evidenceFromRow(row))
	if err != nil {
		return closure.EvidenceRow{}, err
	}
	return evidenceToRow(rec), nil
}

func (w workflowStore) UpdateClosureEvidence(ctx context.Context, row closure.EvidenceRow) error {
	return w.SQLiteStore.UpdateClosureEvidence(ctx, evidenceFromRow(row))
}

func (w workflowStore) ListClosureEvidenceByRepositoryAndStatus(ctx context.Context, repositoryID int64, status string) ([]closure.EvidenceRow, error) {
	recs, err := w.SQLiteStore.ListClosureEvidenceByRepositoryAndStatus(ctx, repositoryID, status)
	if err != nil {
		return nil, err
	}
	out := make([]closure.EvidenceRow, 0, len(recs))
	for _, rec := range recs {
		out = append(out, evidenceToRow(rec))
	}
	return out, nil
}

func (w workflowStore) GetFindingByID(ctx context.Context, findingID int64) (closure.FindingRow, error) {
	d, err := w.SQLiteStore.GetFindingDetail(ctx, findingID)
	if err != nil {
		return closure.FindingRow{}, err
	}
	return closure.FindingRow{ID: d.ID, RepositoryID: d.RepositoryID, Fingerprint: d.Fingerprint, Source: d.Source, Status: d.Status}, nil
}

func (w workflowStore) UpdateFindingStatus(ctx context.Context, findingID int64, status string) error {
	return w.SQLiteStore.UpdateFindingStatus(ctx, findingID, status)
}

func (w workflowStore) AddLifecycleEvent(ctx context.Context, findingID int64, scanID, eventType, message string) error {
	fid := findingID
	return w.SQLiteStore.AddLifecycleEvent(ctx, store.LifecycleEvent{
		FindingID: &fid, ScanID: scanID, EventType: eventType, Message: message,
	})
}

func (w workflowStore) ListExternalIssuesByFinding(ctx context.Context, findingID int64) ([]closure.ExternalIssueRow, error) {
	recs, err := w.SQLiteStore.ListExternalIssuesByFinding(ctx, findingID)
	if err != nil {
		return nil, err
	}
	out := make([]closure.ExternalIssueRow, 0, len(recs))
	for _, rec := range recs {
		out = append(out, closure.ExternalIssueRow{IssueNumber: rec.IssueNumber})
	}
	return out, nil
}

func (w workflowStore) GetRepository(ctx context.Context, repositoryID int64) (closure.RepositoryRow, error) {
	repo, err := w.SQLiteStore.GetRepository(ctx, repositoryID)
	if err != nil {
		return closure.RepositoryRow{}, err
	}
	return closure.RepositoryRow{ID: repo.ID, Owner: repo.Owner, Name: repo.Name, FullName: repo.FullName}, nil
}

func evidenceFromRow(row closure.EvidenceRow) store.ClosureEvidenceRecord {
	return store.ClosureEvidenceRecord{
		ID: row.ID, FindingID: row.FindingID, PatchAttemptID: row.PatchAttemptID,
		RepositoryID: row.RepositoryID, Fingerprint: row.Fingerprint,
		MergeCommitSHA: row.MergeCommitSHA, VerificationScanID: row.VerificationScanID,
		OriginalSource: row.OriginalSource, ScannerStatus: row.ScannerStatus,
		FingerprintPresent: row.FingerprintPresent, Status: row.Status, Reason: row.Reason,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func evidenceToRow(rec store.ClosureEvidenceRecord) closure.EvidenceRow {
	return closure.EvidenceRow{
		ID: rec.ID, FindingID: rec.FindingID, PatchAttemptID: rec.PatchAttemptID,
		RepositoryID: rec.RepositoryID, Fingerprint: rec.Fingerprint,
		MergeCommitSHA: rec.MergeCommitSHA, VerificationScanID: rec.VerificationScanID,
		OriginalSource: rec.OriginalSource, ScannerStatus: rec.ScannerStatus,
		FingerprintPresent: rec.FingerprintPresent, Status: rec.Status, Reason: rec.Reason,
		CreatedAt: rec.CreatedAt, UpdatedAt: rec.UpdatedAt,
	}
}

type fakePRClient struct {
	merged map[int]bool
}

func (f *fakePRClient) GetPullRequest(_ context.Context, _, _ string, prNumber int) (*gitea.PullRequest, error) {
	pr := &gitea.PullRequest{Number: prNumber, Merged: f.merged[prNumber]}
	if f.merged[prNumber] {
		pr.Head.SHA = "merge-sha-abc"
		pr.MergedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return pr, nil
}

type fakeIssues struct {
	comments []string
	labels   []string
}

func (f *fakeIssues) CommentIssue(_ context.Context, _, _ string, _ int, body string) error {
	f.comments = append(f.comments, body)
	return nil
}

func (f *fakeIssues) AddLifecycleLabels(_ context.Context, _, _ string, _ int, lifecycleLabel string) error {
	f.labels = append(f.labels, lifecycleLabel)
	return nil
}

func (f *fakeIssues) CloseIssue(_ context.Context, _, _ string, _ int) error { return nil }

type fixture struct {
	ctx     context.Context
	store   workflowStore
	engine  *closure.Engine
	issues  *fakeIssues
	pr      *fakePRClient
	repoID  int64
	finding int64
	attempt string
	planID  string
	fp      string
}

func openFixture(t *testing.T) *fixture {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Driver: "sqlite", Path: filepath.Join(dir, "e2e.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	sqlite, ok := s.(*store.SQLiteStore)
	if !ok {
		t.Fatal("expected sqlite store")
	}
	ctx := context.Background()
	repo, err := sqlite.UpsertRepository(ctx, store.Repository{
		Owner: "acme", Name: "demo", FullName: "acme/demo", ConnectedRepo: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	fp := "fp-staticcheck-s1039-demo"
	finding, err := sqlite.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: fp, Title: "S1039", Severity: "low",
		Source: "staticcheck", Status: store.FindingStatusOpen,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlite.UpsertExternalIssue(ctx, store.ExternalIssue{
		FindingID: finding.ID, ForgeType: "gitea", IssueNumber: 42,
		IssueURL: "https://gitea.example/acme/demo/issues/42",
	})
	if err != nil {
		t.Fatal(err)
	}
	planID := "rp-e2e-1"
	_, err = sqlite.SaveRemediationPlan(ctx, store.RemediationPlanRecord{
		PlanID: planID, FindingID: ptrInt64(finding.ID), RepositoryID: ptrInt64(repo.ID),
		Fingerprint: fp, Source: "staticcheck", Title: "S1039", Summary: "remove unnecessary Sprintf",
		FixStrategy: "replace fmt.Sprintf with literal", RegressionRisk: remediation.RiskLow,
		FixComplexity: remediation.ComplexitySmall, SafeForAutoPR: true, Status: store.RemediationStatusApproved,
	})
	if err != nil {
		t.Fatal(err)
	}
	prNum := 7
	attemptID := "pa-e2e-1"
	_, err = sqlite.SavePatchAttempt(ctx, store.PatchAttemptRecord{
		AttemptID: attemptID, PlanID: planID, RepositoryID: repo.ID, FindingID: ptrInt64(finding.ID),
		BranchName: "repository-detective/fix/s1039", BaseRef: "main", Status: store.PatchAttemptStatusPROpened,
		PullRequestNumber: &prNum, PullRequestURL: "https://gitea.example/acme/demo/pulls/7",
	})
	if err != nil {
		t.Fatal(err)
	}

	prFake := &fakePRClient{merged: map[int]bool{}}
	issueFake := &fakeIssues{}
	engine := &closure.Engine{
		Config: closure.Config{Enabled: true, Comment: true, RequireScannerSuccess: true},
		Store:  workflowStore{SQLiteStore: sqlite},
		PR:     prFake,
		Issues: issueFake,
	}

	return &fixture{
		ctx: ctx, store: workflowStore{SQLiteStore: sqlite}, engine: engine,
		issues: issueFake, pr: prFake, repoID: repo.ID, finding: finding.ID,
		attempt: attemptID, planID: planID, fp: fp,
	}
}

func ptrInt64(v int64) *int64 { return &v }

func scanCtx(repoID int64, scanID, fp string, includeFP bool, scannerStatus string) closure.ScanContext {
	seen := map[string]struct{}{}
	if includeFP {
		seen[fp] = struct{}{}
	}
	return closure.ScanContext{
		ScanID: scanID, RepositoryID: repoID, Owner: "acme", Repo: "demo",
		FingerprintsSeen: seen,
		ScannerResults:   map[string]string{"staticcheck": scannerStatus},
	}
}

func TestConnectedRepoFlowVerifiedClosure(t *testing.T) {
	f := openFixture(t)

	// PR opened but not merged — no closure yet
	if err := f.engine.OnScanFinish(f.ctx, scanCtx(f.repoID, "scan-pre", f.fp, false, "clean")); err != nil {
		t.Fatal(err)
	}
	ev, err := f.engine.GetEvidence(f.ctx, f.finding)
	if err == nil && ev.Status == closure.StatusVerified {
		t.Fatal("should not verify before merge")
	}

	// Mark PR merged
	f.pr.merged[7] = true
	if _, err := f.engine.CheckPatchAttemptMerge(f.ctx, f.attempt); err != nil {
		t.Fatal(err)
	}
	ev, err = f.engine.GetEvidence(f.ctx, f.finding)
	if err != nil || ev.Status != closure.StatusPendingRescan {
		t.Fatalf("expected pending_rescan after merge, got %+v err=%v", ev, err)
	}
	if !containsLabel(f.issues.labels, issues.LifecyclePendingRescan) {
		t.Fatalf("expected pending-rescan label, got %v", f.issues.labels)
	}

	// Rescan without fingerprint → verified
	if err := f.engine.OnScanFinish(f.ctx, scanCtx(f.repoID, "scan-verify", f.fp, false, "clean")); err != nil {
		t.Fatal(err)
	}
	ev, err = f.engine.GetEvidence(f.ctx, f.finding)
	if err != nil || ev.Status != closure.StatusVerified {
		t.Fatalf("expected verified, got %+v err=%v", ev, err)
	}
	finding, _ := f.store.GetFindingByID(f.ctx, f.finding)
	if finding.Status != "resolved_verified" {
		t.Fatalf("finding status %q", finding.Status)
	}
	if !containsLabel(f.issues.labels, issues.LifecycleResolvedVerified) {
		t.Fatalf("expected resolved-verified label, got %v", f.issues.labels)
	}
}

func TestPROpenedNotMergedNoClosure(t *testing.T) {
	f := openFixture(t)
	if err := f.engine.OnScanFinish(f.ctx, scanCtx(f.repoID, "scan-1", f.fp, false, "clean")); err != nil {
		t.Fatal(err)
	}
	_, err := f.store.GetLatestClosureEvidenceByFindingID(f.ctx, f.finding)
	if err == nil {
		t.Fatal("expected no closure evidence when PR not merged")
	}
	patch, _ := f.store.GetPatchAttemptByAttemptID(f.ctx, f.attempt)
	if patch.Status != store.PatchAttemptStatusPROpened {
		t.Fatalf("patch should remain pr_opened, got %q", patch.Status)
	}
}

func TestPRMergedNoRescanPending(t *testing.T) {
	f := openFixture(t)
	f.pr.merged[7] = true
	if _, err := f.engine.CheckPatchAttemptMerge(f.ctx, f.attempt); err != nil {
		t.Fatal(err)
	}
	ev, err := f.engine.GetEvidence(f.ctx, f.finding)
	if err != nil || ev.Status != closure.StatusPendingRescan {
		t.Fatalf("expected pending_rescan, got %+v", ev)
	}
	finding, _ := f.store.GetFindingByID(f.ctx, f.finding)
	if finding.Status != "pending_rescan" {
		t.Fatalf("finding status %q", finding.Status)
	}
}

func TestPRMergedRescanFingerprintStillPresent(t *testing.T) {
	f := openFixture(t)
	f.pr.merged[7] = true
	_, _ = f.engine.CheckPatchAttemptMerge(f.ctx, f.attempt)
	if err := f.engine.OnScanFinish(f.ctx, scanCtx(f.repoID, "scan-still", f.fp, true, "clean")); err != nil {
		t.Fatal(err)
	}
	ev, _ := f.engine.GetEvidence(f.ctx, f.finding)
	if ev.Status != closure.StatusStillPresent {
		t.Fatalf("expected still_present, got %q", ev.Status)
	}
}

func TestPRMergedRescanScannerFailedClosureBlocked(t *testing.T) {
	f := openFixture(t)
	f.pr.merged[7] = true
	_, _ = f.engine.CheckPatchAttemptMerge(f.ctx, f.attempt)
	if err := f.engine.OnScanFinish(f.ctx, scanCtx(f.repoID, "scan-fail", f.fp, false, "failed")); err != nil {
		t.Fatal(err)
	}
	ev, _ := f.engine.GetEvidence(f.ctx, f.finding)
	if ev.Status != closure.StatusBlocked {
		t.Fatalf("expected blocked, got %q", ev.Status)
	}
	if !containsLabel(f.issues.labels, issues.LifecycleClosureBlocked) {
		t.Fatalf("expected closure-blocked label, got %v", f.issues.labels)
	}
}

func containsLabel(labels []string, want string) bool {
	for _, l := range labels {
		if l == want {
			return true
		}
	}
	return false
}
