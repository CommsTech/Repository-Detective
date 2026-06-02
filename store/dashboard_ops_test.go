package store_test

import (
	"context"
	"testing"
	"time"

	"git.commsnet.org/commstech/bugbot/store"
)

func TestClassifyScanFailure(t *testing.T) {
	tests := map[string]string{
		"clone failed: authentication required": "clone_auth",
		"context deadline exceeded":             "timeout",
		"prepare workspace: content not found":  "prepare",
		"scanner trivy binary_missing":          "scanner",
		"invalid config profile":                "config",
		"something else":                        "other",
	}
	for msg, want := range tests {
		if got := store.ClassifyScanFailure(msg); got != want {
			t.Fatalf("ClassifyScanFailure(%q) = %q, want %q", msg, got, want)
		}
	}
}

func TestBuildRemediationInsightPlannerDisabled(t *testing.T) {
	insight := store.BuildRemediationInsight(10, 0, false, false, "off")
	if insight.Summary == "" || len(insight.Reasons) == 0 {
		t.Fatal("expected explanation when planner disabled")
	}
}

func TestDashboardScannerCountsAreUnique(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	scan1, _ := s.CreateScan(ctx, store.Scan{ID: "scancount01", RepositoryID: repo.ID, TriggerType: store.TriggerManual, Status: store.ScanStatusCompleted})
	scan2, _ := s.CreateScan(ctx, store.Scan{ID: "scancount02", RepositoryID: repo.ID, TriggerType: store.TriggerManual, Status: store.ScanStatusCompleted})
	_ = s.AddScannerResults(ctx, []store.ScannerResultRecord{
		{ScanID: scan1.ID, ScannerName: "trivy", Status: "binary_missing"},
		{ScanID: scan2.ID, ScannerName: "trivy", Status: "binary_missing"},
		{ScanID: scan1.ID, ScannerName: "hadolint", Status: "binary_missing"},
	})

	summary, err := s.DashboardSummary(ctx, 5)
	if err != nil {
		t.Fatal(err)
	}
	if summary.ScannerToolsMissingCount != 2 {
		t.Fatalf("ScannerToolsMissingCount = %d, want 2 distinct scanners (not 3 events)", summary.ScannerToolsMissingCount)
	}
	if summary.Platform.RawMissingEvents != 3 {
		t.Fatalf("RawMissingEvents = %d, want 3 raw events preserved", summary.Platform.RawMissingEvents)
	}
}

func TestDashboardBacklogAggregation(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "b", FullName: "o/b"})
	now := time.Now().UTC()
	old := now.Add(-14 * 24 * time.Hour)
	_, _ = s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "new7d", Severity: "high", Status: store.FindingStatusOpen,
		FirstSeenAt: now.Add(-2 * 24 * time.Hour), LastSeenAt: now,
	})
	_, _ = s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "regress", Severity: "medium", Status: store.FindingStatusOpen,
		FirstSeenAt: old, LastSeenAt: now,
	})

	summary, err := s.DashboardSummary(ctx, 5)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Backlog.NewLast7Days < 1 {
		t.Fatalf("NewLast7Days = %d", summary.Backlog.NewLast7Days)
	}
	if summary.Backlog.RegressionsLast7Days < 1 {
		t.Fatalf("RegressionsLast7Days = %d", summary.Backlog.RegressionsLast7Days)
	}
}

func TestDashboardFailedScanBuckets(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "f", FullName: "o/f"})
	_, _ = s.CreateScan(ctx, store.Scan{
		ID: "failscan01", RepositoryID: repo.ID, TriggerType: store.TriggerManual,
		Status: store.ScanStatusFailed, Error: "context deadline exceeded",
	})

	summary, err := s.DashboardSummary(ctx, 5)
	if err != nil {
		t.Fatal(err)
	}
	if summary.FailedScansCount < 1 {
		t.Fatal("expected failed scan count")
	}
	found := false
	for _, b := range summary.ScanHealth.FailureBuckets {
		if b.Bucket == "timeout" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected timeout bucket in %+v", summary.ScanHealth.FailureBuckets)
	}
}

func TestMissingToolsNotCountedAsFindings(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	scan, _ := s.CreateScan(ctx, store.Scan{ID: "missing001", RepositoryID: repo.ID, Status: store.ScanStatusCompleted})
	for i := 0; i < 5; i++ {
		_ = s.AddScannerResults(ctx, []store.ScannerResultRecord{
			{ScanID: scan.ID, ScannerName: "trivy", Status: "binary_missing"},
		})
	}
	summary, err := s.DashboardSummary(ctx, 5)
	if err != nil {
		t.Fatal(err)
	}
	if summary.OpenFindingsCount != 0 {
		t.Fatalf("open findings should not include scanner missing events, got %d", summary.OpenFindingsCount)
	}
	if summary.ScannerToolsMissingCount != 1 {
		t.Fatalf("expected 1 unique missing scanner, got %d", summary.ScannerToolsMissingCount)
	}
}
