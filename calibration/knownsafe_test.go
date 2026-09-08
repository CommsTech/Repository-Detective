package calibration

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/profile"
)

func TestApplyKnownSafeRoutingG201Store(t *testing.T) {
	issues := []ai.CodeIssue{{
		Source:      "gosec",
		RuleID:      "G201",
		Severity:    "medium",
		Category:    "security",
		File:        "store/learning_sqlite.go",
		CodeSnippet: `query := fmt.Sprintf("DELETE FROM x WHERE id IN (%s)", strings.Join(placeholders, ","))`,
		Confidence:  0.9,
		ReportingAction: profile.ActionAutoIssue,
	}}
	ApplyKnownSafeRouting(issues)
	if issues[0].ReportingAction != profile.ActionReportOnly {
		t.Fatalf("expected report_only, got %s", issues[0].ReportingAction)
	}
	if issues[0].Severity != "info" {
		t.Fatalf("expected info severity for medium FP, got %s", issues[0].Severity)
	}
}

func TestApplyKnownSafeRoutingG703Patcher(t *testing.T) {
	issues := []ai.CodeIssue{{
		Source:          "gosec",
		RuleID:          "G703",
		Severity:        "high",
		File:            "patcher/rules_hadolint.go",
		Confidence:      0.95,
		ReportingAction: profile.ActionAutoIssue,
	}}
	ApplyKnownSafeRouting(issues)
	if issues[0].ReportingAction != profile.ActionReportOnly {
		t.Fatalf("expected report_only for G703 patcher, got %s", issues[0].ReportingAction)
	}
	if issues[0].Severity != "high" {
		t.Fatalf("high severity must remain visible, got %s", issues[0].Severity)
	}
}

func TestApplyKnownSafeRoutingLeavesRealSQL(t *testing.T) {
	issues := []ai.CodeIssue{{
		Source:          "gosec",
		RuleID:          "G201",
		Severity:        "high",
		File:            "api/handler.go",
		CodeSnippet:     `query := fmt.Sprintf("SELECT * FROM users WHERE name = '%s'", userInput)`,
		ReportingAction: profile.ActionAutoIssue,
		Confidence:      0.9,
	}}
	ApplyKnownSafeRouting(issues)
	if issues[0].ReportingAction != profile.ActionAutoIssue {
		t.Fatalf("must not quiet untrusted SQL formatting, got %s", issues[0].ReportingAction)
	}
}
