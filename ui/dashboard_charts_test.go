package ui

import (
	"encoding/json"
	"testing"

	"git.commsnet.org/commstech/bugbot/store"
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
