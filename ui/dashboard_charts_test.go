package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestBuildDashboardChartJSONValid(t *testing.T) {
	summary := store.DashboardSummary{
		OpenFindingsBySeverity: map[string]int{
			"critical": 1,
			"high":     2,
		},
		OpenFindingsByCategory: map[string]int{
			"security": 3,
		},
		Backlog: store.FindingBacklogSummary{OpenUnique: 3},
	}
	repos := []store.RepositorySummary{{
		Repository:        store.Repository{FullName: "org/app"},
		OpenFindingsCount: 3,
		LastScanStatus:    "completed",
	}}

	raw := buildDashboardChartJSON(summary, repos)
	var payload dashboardChartPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(payload.SeverityLabels) != 2 {
		t.Fatalf("expected 2 severity labels, got %v", payload.SeverityLabels)
	}
	if payload.CategoryValues[0] != 3 {
		t.Fatalf("expected category count 3, got %v", payload.CategoryValues)
	}
}

func TestScanTrendFromDayCountsSpansWindow(t *testing.T) {
	now := time.Now().UTC()
	byDay := map[string]int{
		now.AddDate(0, 0, -10).Format("2006-01-02"): 3,
		now.AddDate(0, 0, -3).Format("2006-01-02"):  7,
		now.Format("2006-01-02"):                    2,
	}
	trend := scanTrendFromDayCounts(byDay, 14)
	if len(trend) != 14 {
		t.Fatalf("len=%d", len(trend))
	}
	var sum int
	for _, p := range trend {
		sum += p.value
	}
	if sum != 12 {
		t.Fatalf("sum=%d want 12", sum)
	}
	if trend[len(trend)-1].value != 2 {
		t.Fatalf("today value=%d", trend[len(trend)-1].value)
	}
	if trend[3].value != 3 {
		t.Fatalf("day -10 value=%d label=%s", trend[3].value, trend[3].label)
	}
}

func TestBuildDashboardChartJSONUsesStoreActivity(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "chart.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	repo, err := s.UpsertRepository(ctx, store.Repository{
		Owner: "o", Name: "r", FullName: "o/r", ForgeType: store.ForgeTypeGitea, ConnectedRepo: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	insert := func(id string, dayOffset int) {
		t.Helper()
		_, err := s.CreateScan(ctx, store.Scan{
			ID:           id,
			RepositoryID: repo.ID,
			TriggerType:  store.TriggerManual,
			Ref:          "main",
			Status:       store.ScanStatusCompleted,
			StartedAt:    now.AddDate(0, 0, dayOffset),
		})
		if err != nil {
			t.Fatalf("create scan %s: %v", id, err)
		}
	}
	insert("scan-day-minus-5", -5)
	insert("scan-day-minus-2a", -2)
	insert("scan-day-minus-2b", -2)
	insert("scan-today", 0)

	raw := buildDashboardChartJSONWithStore(ctx, s, store.DashboardSummary{}, nil)
	var payload dashboardChartPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.ScanTrendValues) != 14 {
		t.Fatalf("trend len=%d", len(payload.ScanTrendValues))
	}
	var sum int
	for _, n := range payload.ScanTrendValues {
		sum += n
	}
	if sum != 4 {
		t.Fatalf("sum=%d values=%v", sum, payload.ScanTrendValues)
	}
	if payload.ScanTrendValues[len(payload.ScanTrendValues)-1] != 1 {
		t.Fatalf("today=%d", payload.ScanTrendValues[len(payload.ScanTrendValues)-1])
	}
	if payload.ScanTrendValues[len(payload.ScanTrendValues)-3] != 2 {
		t.Fatalf("day-2=%d", payload.ScanTrendValues[len(payload.ScanTrendValues)-3])
	}
	if len(payload.RemediationTrendValues) != 14 || len(payload.PlanTrendValues) != 14 {
		t.Fatalf("expected remediation/plan series length 14, got rem=%d plans=%d",
			len(payload.RemediationTrendValues), len(payload.PlanTrendValues))
	}
}

func TestBuildDashboardChartJSONUsesFullWindowNotRecentFindingTotals(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "chart-regression.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	repo, err := s.UpsertRepository(ctx, store.Repository{
		Owner: "o", Name: "r", FullName: "o/r", ForgeType: store.ForgeTypeGitea, ConnectedRepo: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	// Historical completed scans spread across days (what the chart should show).
	for _, off := range []int{-9, -6, -3} {
		_, err := s.CreateScan(ctx, store.Scan{
			ID:           fmt.Sprintf("hist-%d", -off),
			RepositoryID: repo.ID,
			TriggerType:  store.TriggerManual,
			Ref:          "main",
			Status:       store.ScanStatusCompleted,
			StartedAt:    now.AddDate(0, 0, off),
			SummaryJSON:  []byte(`{"issues_found":50}`),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	// DB also has two completed scans today (chart must count scans, not issues_found).
	for _, id := range []string{"recent-a", "recent-b"} {
		_, err := s.CreateScan(ctx, store.Scan{
			ID:           id,
			RepositoryID: repo.ID,
			TriggerType:  store.TriggerManual,
			Ref:          "main",
			Status:       store.ScanStatusCompleted,
			StartedAt:    now,
			SummaryJSON:  []byte(`{"issues_found":9999}`),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	// RecentScans mirror (old bug summed issues_found from this short list only).
	recent := []store.ScanWithRepo{
		{
			Scan: store.Scan{
				ID: "recent-a", RepositoryID: repo.ID, Status: store.ScanStatusCompleted,
				StartedAt: now, SummaryJSON: []byte(`{"issues_found":9999}`),
			},
		},
		{
			Scan: store.Scan{
				ID: "recent-b", RepositoryID: repo.ID, Status: store.ScanStatusCompleted,
				StartedAt: now, SummaryJSON: []byte(`{"issues_found":8888}`),
			},
		},
	}

	raw := buildDashboardChartJSONWithStore(ctx, s, store.DashboardSummary{RecentScans: recent}, nil)
	var payload dashboardChartPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatal(err)
	}
	var sum int
	for _, n := range payload.ScanTrendValues {
		sum += n
	}
	// 3 historical + 2 recent today = 5 completed scans in window, not 18887 findings.
	if sum != 5 {
		t.Fatalf("expected 5 completed scans across window, got sum=%d values=%v", sum, payload.ScanTrendValues)
	}
	if payload.ScanTrendValues[len(payload.ScanTrendValues)-1] != 2 {
		t.Fatalf("today should be 2 scans not findings spike, got %d", payload.ScanTrendValues[len(payload.ScanTrendValues)-1])
	}
}
