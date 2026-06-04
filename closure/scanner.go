package closure

import "strings"

// ScannerForSource maps a finding source to the scanner name used in scan results.
func ScannerForSource(source string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	switch source {
	case "golangci-lint":
		return "staticcheck"
	case "reliability", "tech_debt", "maintainability", "test_gap", "performance", "ai_generated_risk":
		return "health"
	case "semgrep", "trivy", "grype", "gitleaks", "govulncheck", "gosec",
		"staticcheck", "hadolint", "checkov", "linters", "graph", "health", "static":
		return source
	default:
		return source
	}
}

// ScannerSucceeded reports whether a scanner status counts as successful evidence.
func ScannerSucceeded(status string, requireSuccess bool) bool {
	if !requireSuccess {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "clean", "found":
		return true
	case "disabled":
		return false
	default:
		return false
	}
}

// ScannerMissing reports whether scanner evidence is absent from the scan.
func ScannerMissing(status string) bool {
	return strings.TrimSpace(status) == ""
}
