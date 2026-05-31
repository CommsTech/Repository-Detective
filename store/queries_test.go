package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/bugbot/store"
)

func TestDashboardSummaryQuery(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	_, _ = s.CreateScan(ctx, store.Scan{ID: "dashscan01", RepositoryID: repo.ID, TriggerType: store.TriggerManual, Status: store.ScanStatusFailed})
	now := time.Now().UTC()
	_, _ = s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "bugbot-dash", Severity: "high", Status: store.FindingStatusOpen, FirstSeenAt: now, LastSeenAt: now})

	summary, err := s.DashboardSummary(ctx, 5)
	if err != nil {
		t.Fatalf("dashboard summary: %v", err)
	}
	if summary.TotalRepositories != 1 {
		t.Fatalf("repos %d", summary.TotalRepositories)
	}
	if summary.FailedScansCount < 1 {
		t.Fatal("expected failed scan count")
	}
}

func TestListRepositoriesWithSummary(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "a", Name: "b", FullName: "a/b"})
	_, _ = s.CreateScan(ctx, store.Scan{ID: "sumscan001", RepositoryID: repo.ID, TriggerType: store.TriggerPush, Status: store.ScanStatusCompleted})

	list, err := s.ListRepositoriesWithSummary(ctx, store.ListOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].LastScanStatus == "" {
		t.Fatalf("unexpected summary: %+v", list)
	}
}

func TestListFindingsFilters(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "filter.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	now := time.Now().UTC()
	_, _ = s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "f1", Severity: "high", Category: "security", Source: "semgrep", FirstSeenAt: now, LastSeenAt: now})
	_, _ = s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "f2", Severity: "low", Category: "quality", Source: "ruff", FirstSeenAt: now, LastSeenAt: now})

	high, err := s.ListFindings(ctx, store.FindingFilter{Severity: "high", Limit: 10})
	if err != nil || len(high) != 1 {
		t.Fatalf("high filter: len=%d err=%v", len(high), err)
	}
}

func TestListScansByRepository(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	for _, id := range []string{"scan001a", "scan001b", "scan001c"} {
		_, _ = s.CreateScan(ctx, store.Scan{ID: id, RepositoryID: repo.ID, TriggerType: store.TriggerManual})
	}
	scans, err := s.ListScansByRepository(ctx, repo.ID, store.ListOptions{Limit: 2})
	if err != nil || len(scans) != 2 {
		t.Fatalf("list scans: len=%d err=%v", len(scans), err)
	}
}

func TestValidateSettingsUpdate(t *testing.T) {
	policy := "gate_pr"
	if err := store.ValidateSettingsUpdate(store.SettingsUpdate{PolicyLevel: &policy}); err != nil {
		t.Fatal(err)
	}
	bad := "not_a_policy"
	if err := store.ValidateSettingsUpdate(store.SettingsUpdate{PolicyLevel: &bad}); err == nil {
		t.Fatal("expected validation error")
	}
}
