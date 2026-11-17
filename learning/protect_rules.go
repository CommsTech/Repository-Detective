package learning

import "strings"

// AIConfirmedFalsePositive is intentionally conservative: an AI prediction,
// including a "false_positive" label, is not independent evidence for
// automatically suppressing a repository finding. Human-reviewed, scoped
// calibration follows the separate acceptance workflow.
func AIConfirmedFalsePositive(classification string) bool {
	_ = classification
	return false
}

// IsProtectedRuleID reports rule IDs / sources that must never become unattended
// report_only / suppress calibrations. Category is often empty on generated
// recommendations, so rule-ID protection is the authoritative gate.
func IsProtectedRuleID(source, ruleID string) bool {
	rule := strings.ToUpper(strings.TrimSpace(ruleID))
	rule = strings.Trim(rule, "`\"' ")
	src := strings.ToLower(strings.TrimSpace(source))

	for _, protectedSource := range []string{"gitleaks", "trivy", "grype", "govulncheck", "semgrep", "custom_rule"} {
		if strings.Contains(src, protectedSource) {
			return true
		}
	}
	// gosec: only allow well-known informational tooling rules through.
	if strings.Contains(src, "gosec") {
		switch rule {
		case "G104", "G304", "G204":
			return false
		default:
			return true
		}
	}
	// checkov secret rules and static security family.
	if strings.Contains(src, "checkov") && (strings.Contains(rule, "SECRET") || strings.HasPrefix(rule, "CKV_SECRET")) {
		return true
	}
	// staticcheck unused-code (U1000) is actionable tech debt — never auto-downgrade.
	if strings.Contains(src, "staticcheck") || rule == "U1000" || strings.HasPrefix(rule, "U1") {
		if rule == "U1000" || strings.HasPrefix(rule, "U1") {
			return true
		}
	}
	for _, prefix := range []string{"CUSTOM_", "CVE-", "GHSA-", "TRIVY-", "GRYPE-", "GITLEAKS-", "SEC-", "CKV_SECRET", "G201", "G202", "G703"} {
		if strings.HasPrefix(rule, prefix) {
			return true
		}
	}
	for _, marker := range []string{"SECRET", "CREDENTIAL", "PASSWORD", "TOKEN", "PRIVATE-KEY", "PRIVATE_KEY"} {
		if strings.Contains(rule, marker) {
			return true
		}
	}
	return false
}
