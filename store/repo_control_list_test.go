package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestListRepositoryControlRows(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "control-list.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "amber", Name: "app", FullName: "amber/app"})
	if err != nil {
		t.Fatal(err)
	}
	scanID := "control-scan-1"
	summary, _ := store.MergeSummaryPipelineFields(nil, map[string]any{
		"issues_found": 2, "persistence_status": store.PersistenceStatusComplete,
		"issue_sync_status": store.IssueSyncStatusSkipped, "dry_run_report_only": true,
	})
	_, err = s.CreateScan(ctx, store.Scan{
		ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerManual,
		Status: store.ScanStatusCompleted, SummaryJSON: summary,
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := store.NewRecorder(s, nil)
	_, err = rec.RecordFindings(ctx, repo.ID, scanID, []ai.CodeIssue{
		{Fingerprint: "fp1", Title: "a", Severity: "low", Source: "graph"},
		{Fingerprint: "fp2", Title: "b", Severity: "low", Source: "graph"},
	})
	if err != nil {
		t.Fatal(err)
	}

	rows, err := s.ListRepositoryControlRows(ctx, store.ListOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d want 1", len(rows))
	}
	if rows[0].ScanFindingsTotal < 1 {
		t.Fatalf("expected scan findings, got %d", rows[0].ScanFindingsTotal)
	}
	if rows[0].IssueSyncStatus != store.IssueSyncStatusSkipped {
		t.Fatalf("issue sync=%q", rows[0].IssueSyncStatus)
	}
}

func TestDisableRepoScanningExcludesScheduler(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "disable-scan.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	repo, _ := s.UpsertRepository(ctx, store.Repository{
		Owner: "o", Name: "r", FullName: "o/r", ConnectedRepo: true, DefaultBranch: "main",
	})
	cron := "0 2 * * *"
	on := true
	off := false
	if err := s.SaveRepoSettings(ctx, store.RepoSettings{
		RepositoryID: repo.ID, Enabled: &on, ScheduleEnabled: &on, ScheduleCron: &cron,
	}); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListScheduledRepositories(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("scheduled before disable: %v len=%d", err, len(list))
	}
	if err := s.SaveRepoSettings(ctx, store.RepoSettings{RepositoryID: repo.ID, Enabled: &off}); err != nil {
		t.Fatal(err)
	}
	list, err = s.ListScheduledRepositories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected disabled repo excluded from scheduler, got %d", len(list))
	}
}
