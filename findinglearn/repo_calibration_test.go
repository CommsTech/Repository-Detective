package findinglearn

import (
	"testing"
	"time"
)

func TestApplyRepoRuleDowngradesLowSeverity(t *testing.T) {
	exp := time.Now().UTC().Add(24 * time.Hour)
	sev, conf, note := ApplyRepoRule("low", 0.8, "maintainability", "HEALTH-MANY-PARAMS", "store/foo.go", RepoCalibrationRule{
		Source: "maintainability", RuleID: "HEALTH-MANY-PARAMS", PathPattern: "store/",
		Action: "informational", Reason: "Store layer — many params expected", Active: true, ExpiresAt: &exp,
	}, time.Now().UTC())
	if sev != "info" || conf > 0.55 || note == "" {
		t.Fatalf("got sev=%s conf=%v note=%q", sev, conf, note)
	}
}

func TestApplyRepoRuleProtectsHighSeverity(t *testing.T) {
	exp := time.Now().UTC().Add(24 * time.Hour)
	sev, _, note := ApplyRepoRule("high", 0.9, "security", "SEC-EVAL", "a.go", RepoCalibrationRule{
		RuleID: "SEC-EVAL", PathPattern: "a.go", Action: "informational", Active: true, ExpiresAt: &exp,
	}, time.Now().UTC())
	if sev != "high" || note != "" {
		t.Fatalf("high must not be downgraded: sev=%s note=%q", sev, note)
	}
}
