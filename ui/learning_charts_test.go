package ui

import (
	"encoding/json"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestBuildLearningChartJSON(t *testing.T) {
	raw := buildLearningChartJSON(
		store.LearningHealthSummary{
			EventsTotal: 10, AvgFalsePositiveRate: 0.25, ScannerFailureRate: 0.1,
			PendingRecommendations: 2, ActiveRepoRules: 3, GroupedFindings: 4,
		},
		map[string]int{"scanner_failed": 7, "user_marked_false_positive": 3},
		[]store.CalibrationRuleStat{
			{Source: "graph", RuleID: "GRAPH-ORPHAN", TotalFindings: 100, FalsePositiveRate: 0.9},
			{Source: "static", RuleID: "QUAL-DEBUG", TotalFindings: 50, FalsePositiveRate: 0.2},
		},
	)
	var payload learningChartPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.EventLabels) != 2 || payload.EventValues[0] != 7 {
		t.Fatalf("events=%v %v", payload.EventLabels, payload.EventValues)
	}
	if !strings.Contains(payload.EventLabels[0], "Scanner") {
		t.Fatalf("label=%q", payload.EventLabels[0])
	}
	if len(payload.NoisyRuleLabels) != 2 || payload.NoisyRuleFPRates[0] < 89 {
		t.Fatalf("noisy=%v rates=%v", payload.NoisyRuleLabels, payload.NoisyRuleFPRates)
	}
	if payload.FPRatePct < 24 || payload.FPRatePct > 26 {
		t.Fatalf("fp pct=%v", payload.FPRatePct)
	}
}

func TestRateMeterHelpers(t *testing.T) {
	if rateMeterClass(0.5) != "danger" || rateMeterClass(0.25) != "warn" || rateMeterClass(0.05) != "success" {
		t.Fatal("class mapping")
	}
	if rateMeterWidth(0.333) != "33%" {
		t.Fatalf("width=%s", rateMeterWidth(0.333))
	}
}
