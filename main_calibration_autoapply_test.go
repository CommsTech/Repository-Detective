package main

import "testing"

// Generated recommendations often carry an empty category, so the category-based
// guard in learning.IsProtectedFromAutoDowngrade lets everything through. These
// cases must therefore be caught by rule ID before an unattended downgrade.
func TestRuleIDProtectedFromAutoApply(t *testing.T) {
	protected := []struct{ source, ruleID string }{
		{"static", "SEC-EVAL"},
		{"static", "SEC-XSS-INNERHTML"},
		{"static", "SEC-HARDCODED-SECRET"},
		{"checkov", "CKV_SECRET_6"},
		{"gitleaks", "generic-api-key"},
		{"trivy", "CVE-2026-1234"},
		{"grype", "GHSA-abcd-efgh-ijkl"},
		{"govulncheck", "GO-2026-1111"},
		{"semgrep", "javascript.lang.security.audit"},
		{"static", "REL-AWS-TOKEN-REUSE"},
		{"graph (AI auditor)", "`SEC-EVAL`"},
	}
	for _, tc := range protected {
		if !ruleIDProtectedFromAutoApply(tc.source, tc.ruleID) {
			t.Errorf("expected %s/%s to be protected from auto-apply", tc.source, tc.ruleID)
		}
	}

	allowed := []struct{ source, ruleID string }{
		{"graph", "GRAPH-ORPHAN-FILE"},
		{"graph", "GRAPH-SUSPICIOUS-ISLAND"},
		{"maintainability", "HEALTH-LARGE-FILE"},
		{"tech_debt", "HEALTH-TECH-PHRASE"},
		{"static", "OPT-NESTED-LOOP"},
		{"static", "QUAL-DEBUG"},
		{"golangci-lint", "LINT-GO-typecheck"},
		{"ruff", "LINT-RUFF-E902-1"},
		{"gosec", "G104"},
	}
	for _, tc := range allowed {
		if ruleIDProtectedFromAutoApply(tc.source, tc.ruleID) {
			t.Errorf("expected %s/%s to be eligible for auto-apply", tc.source, tc.ruleID)
		}
	}
}
