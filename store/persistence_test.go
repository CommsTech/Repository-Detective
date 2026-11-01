package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/sirupsen/logrus"
)

func TestPipelineStateIsReconcilable(t *testing.T) {
	t.Parallel()
	complete := store.ScanPipelineState{
		PersistenceStatus:        store.PersistenceStatusComplete,
		PersistenceExpectedCount: 10,
		IssuesFound:              10,
	}
	if !complete.IsReconcilable(10) {
		t.Fatal("expected reconcilable when complete and counts match")
	}
	pending := store.ScanPipelineState{PersistenceStatus: store.PersistenceStatusPending, IssuesFound: 10}
	if pending.IsReconcilable(10) {
		t.Fatal("pending scan must not be reconcilable")
	}
	legacy := store.ScanPipelineState{IssuesFound: 5}
	if !legacy.IsReconcilable(5) {
		t.Fatal("legacy scan with matching instances should be reconcilable")
	}
	if legacy.IsReconcilable(2) {
		t.Fatal("legacy scan with partial instances must not be reconcilable")
	}
}

func TestRecordFindingsBeforeIssueSync(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "persist.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	scanID := "persistscan01"
	_, _ = s.CreateScan(ctx, store.Scan{ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerManual, Status: store.ScanStatusAnalysisComplete})

	rec := store.NewRecorder(s, logrus.New())
	issues := []ai.CodeIssue{
		{Fingerprint: "fp-a", Title: "A", Severity: "medium", Source: "health", Category: "reliability"},
		{Fingerprint: "fp-b", Title: "B", Severity: "low", Source: "health", Category: "reliability"},
	}
	ids, err := rec.RecordFindings(ctx, repo.ID, scanID, issues)
	if err != nil {
		t.Fatalf("record findings: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 finding ids, got %d", len(ids))
	}
	if !rec.IsPersistenceComplete(ctx, scanID) {
		t.Fatal("persistence should be complete")
	}
	count, err := s.CountFindingInstancesForScan(ctx, scanID)
	if err != nil || count != 2 {
		t.Fatalf("instances=%d err=%v", count, err)
	}
	scan, _ := s.GetScan(ctx, scanID)
	if scan.Status != store.ScanStatusCompleted {
		t.Fatalf("expected completed status, got %q", scan.Status)
	}
	pipeline := store.PipelineStateFromSummary(scan.SummaryJSON)
	if pipeline.PersistenceStatus != store.PersistenceStatusComplete {
		t.Fatalf("pipeline status %q", pipeline.PersistenceStatus)
	}
	if pipeline.IssueSyncStatus != store.IssueSyncStatusPending && pipeline.IssueSyncStatus != "" {
		t.Fatalf("issue sync should remain pending until filing phase, got %q", pipeline.IssueSyncStatus)
	}
}

func TestPersistFindingsWithRepoCalibrationRules(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "calib.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	rid := repo.ID
	_, err = s.CreateRepoCalibrationRule(ctx, store.RepoCalibrationRule{
		RepositoryID: &rid, Scope: "repo", Source: "health", RuleID: "HEALTH-MANY-PARAMS",
		PathPattern: "store/", Action: "informational", Reason: "test calibration", Active: true, EvidenceCount: 5,
	})
	if err != nil {
		t.Fatalf("create calibration rule: %v", err)
	}

	scanID := "calibscan01"
	_, _ = s.CreateScan(ctx, store.Scan{ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerManual, Status: store.ScanStatusAnalysisComplete})

	done := make(chan error, 1)
	go func() {
		issues := []ai.CodeIssue{
			{
				Fingerprint: "fp-calib", Title: "many params", Severity: "low", Confidence: 0.8,
				Source: "health", RuleID: "HEALTH-MANY-PARAMS", File: "store/example.go", Category: "maintainability",
			},
		}
		_, _, err := s.(*store.SQLiteStore).PersistScanFindingsBatch(ctx, repo.ID, scanID, issues, time.Now().UTC())
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("persist with calibration rules: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("PersistScanFindingsBatch hung — calibration rules must load before transaction")
	}

	finding, err := s.GetFindingByFingerprint(ctx, repo.ID, "fp-calib")
	if err != nil {
		t.Fatalf("get finding: %v", err)
	}
	if finding.Severity != "info" {
		t.Fatalf("expected calibrated severity info, got %q", finding.Severity)
	}
}

func TestMarkIssueSyncCompleteAfterFilingPhase(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "issuesync.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	scanID := "issuesyncscan01"
	summary := []byte(`{"issues_found":0,"persistence_status":"complete","issue_sync_status":"pending"}`)
	_, _ = s.CreateScan(ctx, store.Scan{
		ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerManual,
		Status: store.ScanStatusCompleted, SummaryJSON: summary,
	})

	rec := store.NewRecorder(s, logrus.New())
	rec.MarkIssueSyncComplete(ctx, scanID)

	scan, _ := s.GetScan(ctx, scanID)
	pipeline := store.PipelineStateFromSummary(scan.SummaryJSON)
	if pipeline.IssueSyncStatus != store.IssueSyncStatusComplete {
		t.Fatalf("expected issue_sync complete, got %q", pipeline.IssueSyncStatus)
	}
}

func TestPersistScanFindingsBatchIsTransactional(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "batch.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	scanID := "batchscan01"
	_, _ = s.CreateScan(ctx, store.Scan{ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerManual, Status: store.ScanStatusStarted})

	now := time.Now().UTC()
	n := 50
	issues := make([]ai.CodeIssue, n)
	for i := 0; i < n; i++ {
		issues[i] = ai.CodeIssue{
			Fingerprint: "fp-" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
			Title:       "issue",
			Severity:    "low",
			Source:      "health",
		}
	}
	persisted, _, err := s.(*store.SQLiteStore).PersistScanFindingsBatch(ctx, repo.ID, scanID, issues, now)
	if err != nil {
		t.Fatalf("batch persist: %v", err)
	}
	if persisted != n {
		t.Fatalf("persisted %d want %d", persisted, n)
	}
	count, _ := s.CountFindingInstancesForScan(ctx, scanID)
	if count != n {
		t.Fatalf("instance count %d want %d", count, n)
	}
}

func TestGetLatestReconcilableScanSkipsPartial(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "recon.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	partialSummary := []byte(`{"issues_found":3,"persistence_status":"pending","persistence_expected_count":3}`)
	_, _ = s.CreateScan(ctx, store.Scan{
		ID: scanID("partial1"), RepositoryID: repo.ID, TriggerType: store.TriggerManual,
		Status: store.ScanStatusAnalysisComplete, StartedAt: time.Now().UTC(), SummaryJSON: partialSummary,
	})
	goodSummary := []byte(`{"issues_found":1,"persistence_status":"complete","persistence_expected_count":1,"persistence_persisted_count":1}`)
	goodID := scanID("good1")
	_, _ = s.CreateScan(ctx, store.Scan{
		ID: goodID, RepositoryID: repo.ID, TriggerType: store.TriggerManual,
		Status: store.ScanStatusCompleted, StartedAt: time.Now().UTC().Add(time.Second), SummaryJSON: goodSummary,
	})
	finding, _ := s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "fp-good", Title: "x", Severity: "low",
		FirstSeenAt: time.Now().UTC(), LastSeenAt: time.Now().UTC(), LastSeenScanID: goodID,
	})
	_ = s.AddFindingInstance(ctx, store.FindingInstance{FindingID: finding.ID, ScanID: goodID, CreatedAt: time.Now().UTC()})

	latest, err := s.GetLatestReconcilableScanForRepository(ctx, repo.ID)
	if err != nil {
		t.Fatalf("latest reconcilable: %v", err)
	}
	if latest.ID != goodID {
		t.Fatalf("expected good scan, got %s", latest.ID)
	}
}

func scanID(s string) string {
	if len(s) >= 16 {
		return s
	}
	return s + "000000000000000"
}
