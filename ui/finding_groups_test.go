package ui

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestBuildScanFindingsBreakdownGroupsGraphNoise(t *testing.T) {
	findings := make(map[int64]store.Finding)
	for i := 0; i < 5; i++ {
		findings[int64(i+1)] = store.Finding{
			ID:       int64(i + 1),
			RuleID:   "GRAPH-ORPHAN-FILE",
			Source:   "graph",
			Severity: "info",
			Title:    "Orphan file",
			FilePath: "file.go",
		}
	}
	findings[99] = store.Finding{
		ID: 99, RuleID: "SEC-EVAL", Source: "static", Severity: "critical", Title: "eval",
	}
	out := BuildScanFindingsBreakdown(findings)
	if out.GroupedCount != 5 {
		t.Fatalf("expected 5 grouped, got %d", out.GroupedCount)
	}
	if out.ActionableCount != 1 {
		t.Fatalf("expected 1 actionable, got %d", out.ActionableCount)
	}
	if len(out.Grouped) != 1 || out.Grouped[0].Count != 5 {
		t.Fatalf("expected one group of 5, got %+v", out.Grouped)
	}
}

func TestBuildScanFindingsBreakdownSmallGroupsStayActionable(t *testing.T) {
	findings := map[int64]store.Finding{
		1: {ID: 1, RuleID: "GRAPH-ORPHAN-FILE", Source: "graph", Severity: "info"},
		2: {ID: 2, RuleID: "GRAPH-ORPHAN-FILE", Source: "graph", Severity: "info"},
	}
	out := BuildScanFindingsBreakdown(findings)
	if out.GroupedCount != 0 || out.ActionableCount != 2 {
		t.Fatalf("expected 2 actionable ungrouped, got %+v", out)
	}
}
