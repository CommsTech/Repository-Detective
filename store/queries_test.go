package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestCountCompletedScansByDay(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	now := time.Now().UTC()
	_, _ = s.CreateScan(ctx, store.Scan{
		ID: "day-a", RepositoryID: repo.ID, TriggerType: store.TriggerManual,
		Status: store.ScanStatusCompleted, StartedAt: now.AddDate(0, 0, -4),
	})
	_, _ = s.CreateScan(ctx, store.Scan{
		ID: "day-b1", RepositoryID: repo.ID, TriggerType: store.TriggerManual,
		Status: store.ScanStatusCompleted, StartedAt: now.AddDate(0, 0, -1),
	})
	_, _ = s.CreateScan(ctx, store.Scan{
		ID: "day-b2", RepositoryID: repo.ID, TriggerType: store.TriggerManual,
		Status: store.ScanStatusCompleted, StartedAt: now.AddDate(0, 0, -1),
	})
	_, _ = s.CreateScan(ctx, store.Scan{
		ID: "day-fail", RepositoryID: repo.ID, TriggerType: store.TriggerManual,
		Status: store.ScanStatusFailed, StartedAt: now.AddDate(0, 0, -1),
	})

	byDay, err := s.CountCompletedScansByDay(ctx, now.AddDate(0, 0, -13))
	if err != nil {
		t.Fatal(err)
	}
	if byDay[now.AddDate(0, 0, -4).Format("2006-01-02")] != 1 {
		t.Fatalf("day-4=%v", byDay)
	}
	if byDay[now.AddDate(0, 0, -1).Format("2006-01-02")] != 2 {
		t.Fatalf("day-1=%v", byDay)
	}
}

func TestDashboardSummaryQuery(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	_, _ = s.CreateScan(ctx, store.Scan{ID: "dashscan01", RepositoryID: repo.ID, TriggerType: store.TriggerManual, Status: store.ScanStatusFailed})
	now := time.Now().UTC()
	_, _ = s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "rd-dash", Severity: "high", Status: store.FindingStatusOpen, FirstSeenAt: now, LastSeenAt: now})

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

func TestCountFindingsAndSeverityByRepo(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "count.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	now := time.Now().UTC()
	_, _ = s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "f1", Severity: "high", Status: "open", Category: "security", Source: "semgrep", FirstSeenAt: now, LastSeenAt: now})
	_, _ = s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "f2", Severity: "low", Status: "resolved", Category: "quality", Source: "ruff", FirstSeenAt: now, LastSeenAt: now})

	total, err := s.CountFindings(ctx, store.FindingFilter{RepositoryID: repo.ID})
	if err != nil || total != 2 {
		t.Fatalf("count all: got %d err=%v", total, err)
	}
	open, err := s.CountFindings(ctx, store.FindingFilter{RepositoryID: repo.ID, Status: "open"})
	if err != nil || open != 1 {
		t.Fatalf("count open: got %d err=%v", open, err)
	}
	bySev, err := s.OpenFindingsBySeverityForRepository(ctx, repo.ID)
	if err != nil || bySev["high"] != 1 {
		t.Fatalf("severity map: %#v err=%v", bySev, err)
	}
}

func TestListFindingsIgnoresSQLInjectionPayloads(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "sqli.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	now := time.Now().UTC()
	_, _ = s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "safe", Severity: "high", Category: "security", Source: "semgrep", FirstSeenAt: now, LastSeenAt: now})

	got, err := s.ListFindings(ctx, store.FindingFilter{
		Severity: `high' OR 1=1 --`,
		Category: `security'; DROP TABLE findings; --`,
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("list findings: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected injection payloads to match nothing, got %d rows", len(got))
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

func TestCountAutoRemediatedAndPlansByDay(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	repo, err := s.UpsertRepository(ctx, store.Repository{
		Owner: "o", Name: "r", FullName: "o/r", ForgeType: store.ForgeTypeGitea, ConnectedRepo: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	finding, err := s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "fp-auto-rem", Source: "hadolint", RuleID: "DL3018",
		Status: store.FindingStatusOpen, LastSeenScanID: "s1",
	})
	if err != nil {
		t.Fatal(err)
	}
	fid := finding.ID
	planDay := now.AddDate(0, 0, -2)
	_, err = s.SaveRemediationPlan(ctx, store.RemediationPlanRecord{
		PlanID: "plan-chart-1", FindingID: &fid, RepositoryID: &repo.ID, Fingerprint: "fp-auto-rem",
		Status: store.RemediationStatusApproved, CreatedAt: planDay, UpdatedAt: planDay,
	})
	if err != nil {
		t.Fatal(err)
	}
	mergedAt := now.AddDate(0, 0, -1)
	_, err = s.SavePatchAttempt(ctx, store.PatchAttemptRecord{
		AttemptID: "attempt-chart-1", PlanID: "plan-chart-1", RepositoryID: repo.ID, FindingID: &fid,
		Status: store.PatchAttemptStatusPRMerged, CreatedAt: mergedAt, UpdatedAt: mergedAt, MergedAt: &mergedAt,
	})
	if err != nil {
		t.Fatal(err)
	}

	since := now.AddDate(0, 0, -13)
	plans, err := s.CountRemediationPlansByDay(ctx, since)
	if err != nil {
		t.Fatal(err)
	}
	if plans[planDay.Format("2006-01-02")] != 1 {
		t.Fatalf("plans by day=%v", plans)
	}
	remediated, err := s.CountAutoRemediatedFindingsByDay(ctx, since)
	if err != nil {
		t.Fatal(err)
	}
	if remediated[mergedAt.Format("2006-01-02")] != 1 {
		t.Fatalf("auto-remediated by day=%v", remediated)
	}
}
