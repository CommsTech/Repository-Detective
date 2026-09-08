package findinglearn

import "testing"

func TestApplyRepoRoutingForForgeHighSeverity(t *testing.T) {
	rules := []RepoCalibrationRule{{
		Source: "gosec",
		RuleID: "G703",
		Action: "report_only",
		Reason: "workspace-bound write",
		Active: true,
	}}
	action, note := ApplyRepoRoutingForForge("auto_issue", "gosec", "G703", "patcher/rules_hadolint.go", rules)
	if action != "report_only" || note == "" {
		t.Fatalf("expected report_only for calibrated high finding, got action=%s note=%q", action, note)
	}
}

func TestApplyRepoRoutingPathPattern(t *testing.T) {
	rules := []RepoCalibrationRule{{
		Source:      "gosec",
		RuleID:      "G201",
		PathPattern: "store/*",
		Action:      "report_only",
		Active:      true,
	}}
	action, _ := ApplyRepoRoutingForForge("auto_issue", "gosec", "G201", "store/learning_sqlite.go", rules)
	if action != "report_only" {
		t.Fatalf("store path should match, got %s", action)
	}
	action, note := ApplyRepoRoutingForForge("auto_issue", "gosec", "G201", "api/handler.go", rules)
	if action != "auto_issue" || note != "" {
		t.Fatalf("non-store path must not match, got action=%s note=%q", action, note)
	}
}
