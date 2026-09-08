package calibration

import (
	"path/filepath"
	"strings"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/profile"
)

// ApplyKnownSafeRouting downgrades forge routing for scanner findings that match
// trusted patterns (placeholders, tooling subprocesses, workspace-bound writes).
// Findings remain visible; only ReportingAction / confidence are adjusted.
func ApplyKnownSafeRouting(issues []ai.CodeIssue) {
	for i := range issues {
		applyKnownSafeOne(&issues[i])
	}
}

func applyKnownSafeOne(issue *ai.CodeIssue) {
	if issue == nil {
		return
	}
	src := strings.ToLower(strings.TrimSpace(issue.Source))
	rule := strings.TrimSpace(issue.RuleID)
	file := filepath.ToSlash(issue.File)
	code := issue.CodeSnippet
	if code == "" {
		code = issue.Evidence
	}

	reason := ""
	switch {
	case isTrustedSQLPlaceholder(src, rule, file, code):
		reason = "known-safe: SQL uses trusted placeholders / store-layer parameterization"
	case isToolingSubprocess(src, rule, file):
		reason = "known-safe: fixed tooling subprocess in scanner/patcher/preinstall paths"
	case isWorkspaceBoundWrite(src, rule, file):
		reason = "known-safe: write confined to remediation workspace helpers"
	case isShellcheckStyleNoise(src, rule):
		reason = "known-safe: ShellCheck style/unused-variable noise (dashboard only)"
	case isGosecUncheckedAssignNoise(src, rule):
		reason = "known-safe: gosec G104 unchecked-assign is informational for this codebase"
	case isIntentionalWithoutCancel(src, rule, file, code):
		reason = "known-safe: context.WithoutCancel used to detach background audit from request cancel"
	}
	if reason == "" {
		return
	}
	issue.ReportingAction = profile.ActionReportOnly
	if issue.SuppressionReason == "" {
		issue.SuppressionReason = reason
	}
	if issue.Confidence > 0.55 {
		issue.Confidence = 0.55
	}
	// Medium/low verified-safe patterns become info; keep high/critical severity visible
	// but forge-quiet via report_only (operator still sees them on the dashboard).
	sev := strings.ToLower(strings.TrimSpace(issue.Severity))
	if sev == "medium" || sev == "low" || sev == "warning" || sev == "warn" {
		issue.Severity = "info"
	}
}

func isTrustedSQLPlaceholder(source, rule, file, code string) bool {
	if !strings.EqualFold(source, "gosec") {
		return false
	}
	rule = strings.ToUpper(rule)
	if rule != "G201" && rule != "G202" {
		return false
	}
	lowerFile := strings.ToLower(file)
	trimmed := strings.TrimSpace(code)
	if strings.Contains(lowerFile, "store/") || strings.HasPrefix(lowerFile, "store/") {
		if strings.Contains(trimmed, "fmt.Sprintf") ||
			strings.Contains(trimmed, "placeholders") ||
			strings.Contains(trimmed, "strings.Join") {
			return true
		}
	}
	if strings.Contains(trimmed, "strings.Join(placeholders") && strings.Contains(trimmed, "IN (") {
		return true
	}
	// Explicit placeholder-only construction (common RD store pattern).
	if strings.Contains(trimmed, `"?"`) || strings.Contains(trimmed, ` "?" `) {
		if strings.Contains(trimmed, "fmt.Sprintf") || strings.Contains(trimmed, "Join(") {
			return true
		}
	}
	return false
}

func isToolingSubprocess(source, rule, file string) bool {
	if !strings.EqualFold(source, "gosec") || !strings.EqualFold(rule, "G204") {
		return false
	}
	lower := strings.ToLower(file)
	for _, prefix := range []string{
		"scanners/", "patcher/", "preinstall/", "sbom/", "analyzers/", "cmd/", "scripts/",
	} {
		if strings.HasPrefix(lower, prefix) || strings.Contains(lower, "/"+prefix) {
			return true
		}
	}
	return false
}

func isWorkspaceBoundWrite(source, rule, file string) bool {
	if !strings.EqualFold(source, "gosec") || !strings.EqualFold(rule, "G703") {
		return false
	}
	lower := strings.ToLower(file)
	return strings.HasPrefix(lower, "patcher/") || strings.Contains(lower, "/patcher/")
}

func isShellcheckStyleNoise(source, rule string) bool {
	src := strings.ToLower(source)
	if src != "shellcheck" && src != "linters" {
		return false
	}
	r := strings.ToUpper(rule)
	// SC2034 unused var; common style noise that rarely warrants forge issues.
	return strings.Contains(r, "2034") || strings.HasSuffix(r, "SC2034") || r == "LINT-SHELL-2034"
}

func isGosecUncheckedAssignNoise(source, rule string) bool {
	return strings.EqualFold(source, "gosec") && strings.EqualFold(rule, "G104")
}

func isIntentionalWithoutCancel(source, rule, file, code string) bool {
	if !strings.EqualFold(source, "gosec") || !strings.EqualFold(rule, "G118") {
		return false
	}
	lower := strings.ToLower(file)
	if !(strings.HasPrefix(lower, "preinstall/") || strings.Contains(lower, "/preinstall/")) {
		return false
	}
	return strings.Contains(code, "WithoutCancel") || strings.Contains(code, "context.WithoutCancel")
}
