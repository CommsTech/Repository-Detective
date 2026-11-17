package calibration

import (
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/profile"
)

func TestApplyKnownSafeRoutingG201Store(t *testing.T) {
	issues := []ai.CodeIssue{{
		Source:          "gosec",
		RuleID:          "G201",
		Severity:        "medium",
		Category:        "security",
		File:            "store/learning_sqlite.go",
		CodeSnippet:     `query := fmt.Sprintf("DELETE FROM x WHERE id IN (%s)", strings.Join(placeholders, ","))`,
		Confidence:      0.9,
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

func TestApplyKnownSafeRoutingG203UIHelpers(t *testing.T) {
	issues := []ai.CodeIssue{{
		Source:          "gosec",
		RuleID:          "G203",
		Severity:        "medium",
		File:            "ui/ui_helpers.go",
		CodeSnippet:     `return template.JS(raw) // after json.Valid`,
		ReportingAction: profile.ActionAutoIssue,
		Confidence:      0.9,
	}}
	ApplyKnownSafeRouting(issues)
	if issues[0].ReportingAction != profile.ActionReportOnly {
		t.Fatalf("expected report_only for validated JSON script content, got %s", issues[0].ReportingAction)
	}
}

func TestApplyKnownSafeRoutingDocsdataOpenAPI(t *testing.T) {
	issues := []ai.CodeIssue{{
		Source:          "checkov",
		RuleID:          "CKV_OPENAPI_5",
		Severity:        "medium",
		File:            "docsdata/openapi.yaml",
		ReportingAction: profile.ActionAutoIssue,
		Confidence:      0.9,
	}}
	ApplyKnownSafeRouting(issues)
	if issues[0].ReportingAction != profile.ActionReportOnly {
		t.Fatalf("expected report_only for docsdata OpenAPI, got %s", issues[0].ReportingAction)
	}
}

func TestApplyKnownSafeRoutingVulnerableDemo(t *testing.T) {
	issues := []ai.CodeIssue{{
		Source:          "trivy",
		RuleID:          "TRIVY-CVE-2024-26130",
		Severity:        "high",
		File:            "examples/vulnerable-demo/requirements.txt",
		ReportingAction: profile.ActionAutoIssue,
		Confidence:      0.95,
	}}
	ApplyKnownSafeRouting(issues)
	if issues[0].ReportingAction != profile.ActionReportOnly {
		t.Fatalf("expected report_only for vulnerable-demo, got %s", issues[0].ReportingAction)
	}
}

func TestApplyKnownSafeRoutingHadolintDockerfileSC1091(t *testing.T) {
	issues := []ai.CodeIssue{{
		Source:          "hadolint",
		RuleID:          "SC1091",
		File:            "Dockerfile",
		Severity:        "low",
		ReportingAction: profile.ActionAutoIssue,
	}}
	ApplyKnownSafeRouting(issues)
	if issues[0].ReportingAction != profile.ActionReportOnly {
		t.Fatalf("expected report_only for Dockerfile SC1091, got %s", issues[0].ReportingAction)
	}
	if !strings.Contains(issues[0].SuppressionReason, "SC1091") {
		t.Fatalf("expected SC1091 reason, got %q", issues[0].SuppressionReason)
	}
}

func TestApplyKnownSafeRoutingSandboxWalk(t *testing.T) {
	issues := []ai.CodeIssue{{
		Source:          "gosec",
		RuleID:          "G122",
		Severity:        "high",
		File:            "preinstall/clone.go",
		ReportingAction: profile.ActionAutoIssue,
		Confidence:      0.9,
	}}
	ApplyKnownSafeRouting(issues)
	if issues[0].ReportingAction != profile.ActionReportOnly {
		t.Fatalf("expected report_only for sandbox WalkDir, got %s", issues[0].ReportingAction)
	}
}

func TestApplyKnownSafeRoutingNestedLoopDocs(t *testing.T) {
	issues := []ai.CodeIssue{{
		Source:          "static",
		RuleID:          "OPT-NESTED-LOOP",
		Severity:        "info",
		File:            "LICENSE",
		ReportingAction: profile.ActionAutoIssue,
	}}
	ApplyKnownSafeRouting(issues)
	if issues[0].ReportingAction != profile.ActionReportOnly {
		t.Fatalf("expected report_only for LICENSE nested-loop noise, got %s", issues[0].ReportingAction)
	}
}

func TestApplyKnownSafeRoutingGolangciExportData(t *testing.T) {
	issues := []ai.CodeIssue{{
		Source:          "golangci-lint",
		RuleID:          "LINT-GO-typecheck",
		Severity:        "info",
		Title:           `could not import sync/atomic (-: could not load export data: unsupported version: 2)`,
		ReportingAction: profile.ActionAutoIssue,
	}}
	ApplyKnownSafeRouting(issues)
	if issues[0].ReportingAction != profile.ActionReportOnly {
		t.Fatalf("expected report_only for golangci export-data noise, got %s", issues[0].ReportingAction)
	}
}

func TestApplyKnownSafeRoutingG304ToolingPath(t *testing.T) {
	issues := []ai.CodeIssue{{
		Source:          "gosec",
		RuleID:          "G304",
		Severity:        "medium",
		File:            "sbom/persist.go",
		ReportingAction: profile.ActionAutoIssue,
		Confidence:      0.9,
	}}
	ApplyKnownSafeRouting(issues)
	if issues[0].ReportingAction != profile.ActionReportOnly {
		t.Fatalf("expected report_only for G304 tooling path, got %s", issues[0].ReportingAction)
	}
}
