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
	title := issue.Title
	if issue.Description != "" {
		title = title + " " + issue.Description
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
	case isDoctorCLIFatalExit(src, rule, file):
		reason = "known-safe: doctor CLI process exits are intentional, not library fatal paths"
	case isPrivacyLocalhostClassification(src, rule, file, code):
		reason = "known-safe: localhost host check is privacy-mode classification, not infra leakage"
	case isConfigTemplateSecretEntropy(src, rule, file):
		reason = "known-safe: config.env.template placeholders are not live secrets"
	case isValidatedJSONScriptContent(src, rule, file, code):
		reason = "known-safe: template.JS wraps JSON.Valid-checked payloads for application/json script tags"
	case isDocsdataOpenAPICheckov(src, rule, file):
		reason = "known-safe: bundled docsdata OpenAPI is documentation surface, not a live API gateway"
	case isWorkflowEnvShellInjectionNoise(src, rule, file, code, title):
		reason = "known-safe: workflow inputs reach shell only via env: intermediates (not inline ${{ }} in run:)"
	case isSandboxWalkChmod(src, rule, file):
		reason = "known-safe: WalkDir+Chmod hardens operator-owned preinstall sandbox trees"
	case isVulnerableDemoFixture(src, rule, file):
		reason = "known-safe: examples/vulnerable-demo intentionally contains vulnerable pins for product demos"
		// Demo fixtures are never actionable on the product repo — demote so they
		// do not keep the self-scan score/open queue red.
		issue.Severity = "info"
	case isHadolintDockerfileSourceFollow(src, rule, file):
		reason = "known-safe: Dockerfile sources scripts COPY'd into the image; hadolint SC1091 cannot follow them"
	case isNestedLoopNonSourceNoise(src, rule, file):
		reason = "known-safe: nested-loop heuristic on docs/license/markdown is not actionable code"
	case isGolangciExportDataNoise(src, rule, title):
		reason = "known-safe: golangci-lint typecheck export-data / module-resolution noise from scanner toolchain skew"
	case isToolingPathFileRead(src, rule, file):
		reason = "known-safe: file read confined to scanner/patcher/sbom/preinstall/health/graph tooling paths"
	case isCalibrationSelfReference(src, rule, file):
		reason = "known-safe: calibration/knownsafe documents localhost patterns; not live infra leakage"
	case isEnvExampleCommentBlock(src, rule, file):
		reason = "known-safe: .env.example commented templates are documentation, not dead production code"
	case isDebugLiteralInDocs(src, rule, file):
		reason = "known-safe: QUAL-DEBUG / debug literals in markdown and docs are documentation noise"
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

func isDoctorCLIFatalExit(source, rule, file string) bool {
	if !strings.EqualFold(rule, "HEALTH-FATAL-EXIT") {
		return false
	}
	lower := strings.ToLower(file)
	return strings.Contains(lower, "main_doctor.go") || strings.HasSuffix(lower, "doctor/main.go")
}

func isPrivacyLocalhostClassification(source, rule, file, code string) bool {
	if !strings.EqualFold(rule, "REL-INTERNAL-INFRA-REF") {
		return false
	}
	lower := strings.ToLower(file)
	if !strings.Contains(lower, "internal/privacy/") {
		return false
	}
	return strings.Contains(code, "localhost") || strings.Contains(strings.ToLower(code), "host ==")
}

func isConfigTemplateSecretEntropy(source, rule, file string) bool {
	if !strings.Contains(strings.ToUpper(rule), "SECRET") && !strings.EqualFold(rule, "CKV_SECRET_6") {
		return false
	}
	lower := strings.ToLower(file)
	return strings.Contains(lower, ".template") || strings.Contains(lower, ".example") ||
		strings.HasSuffix(lower, "config.env.template")
}

func isValidatedJSONScriptContent(source, rule, file, code string) bool {
	if !strings.EqualFold(source, "gosec") || !strings.EqualFold(rule, "G203") {
		return false
	}
	lower := strings.ToLower(file)
	if !(strings.Contains(lower, "ui/ui_helpers.go") || strings.HasSuffix(lower, "ui_helpers.go")) {
		return false
	}
	return strings.Contains(code, "template.JS") || strings.Contains(code, "json.Valid") ||
		strings.Contains(code, "jsonScriptContent")
}

func isDocsdataOpenAPICheckov(source, rule, file string) bool {
	if !strings.EqualFold(source, "checkov") {
		return false
	}
	if !strings.Contains(strings.ToUpper(rule), "OPENAPI") {
		return false
	}
	lower := strings.ToLower(filepath.ToSlash(file))
	return strings.Contains(lower, "docsdata/") || strings.HasSuffix(lower, "openapi.yaml") ||
		strings.HasSuffix(lower, "openapi.yml")
}

func isWorkflowEnvShellInjectionNoise(source, rule, file, code, title string) bool {
	if !strings.EqualFold(source, "semgrep") {
		return false
	}
	blob := strings.ToLower(rule + " " + code + " " + title)
	if !strings.Contains(blob, "shell-injection") && !strings.Contains(blob, "run-shell-injection") {
		return false
	}
	lower := strings.ToLower(filepath.ToSlash(file))
	if !(strings.Contains(lower, ".github/workflows/") || strings.Contains(lower, ".gitea/workflows/")) {
		return false
	}
	// Remains noisy when inputs are already staged via env:; treat as known-safe after that pattern.
	return strings.Contains(code, "INPUT_") || strings.Contains(code, "env:") ||
		strings.Contains(lower, "docker-publish.yml")
}

func isSandboxWalkChmod(source, rule, file string) bool {
	if !strings.EqualFold(source, "gosec") || !strings.EqualFold(rule, "G122") {
		return false
	}
	lower := strings.ToLower(filepath.ToSlash(file))
	return strings.HasPrefix(lower, "preinstall/") || strings.Contains(lower, "/preinstall/")
}

func isVulnerableDemoFixture(source, rule, file string) bool {
	src := strings.ToLower(source)
	if src != "trivy" && src != "grype" && src != "govulncheck" && src != "osv" && src != "nancy" {
		return false
	}
	lower := strings.ToLower(filepath.ToSlash(file))
	return strings.Contains(lower, "examples/vulnerable-demo/") ||
		strings.Contains(lower, "vulnerable-demo/")
}

func isHadolintDockerfileSourceFollow(source, rule, file string) bool {
	if !strings.EqualFold(source, "hadolint") {
		return false
	}
	rule = strings.ToUpper(strings.TrimSpace(rule))
	if rule != "SC1091" && rule != "DL4006" {
		return false
	}
	base := strings.ToLower(filepath.Base(filepath.ToSlash(file)))
	return base == "dockerfile" || strings.HasPrefix(base, "dockerfile.")
}

func isNestedLoopNonSourceNoise(source, rule, file string) bool {
	if !strings.EqualFold(rule, "OPT-NESTED-LOOP") {
		return false
	}
	lower := strings.ToLower(filepath.ToSlash(file))
	base := filepath.Base(lower)
	switch {
	case strings.HasSuffix(lower, ".md"), strings.HasSuffix(lower, ".txt"), strings.HasSuffix(lower, ".rst"):
		return true
	case base == "license", base == "copying", base == "notice", base == "authors":
		return true
	case strings.HasPrefix(lower, "docs/"), strings.Contains(lower, "/docs/"):
		return true
	default:
		return false
	}
}

func isGolangciExportDataNoise(source, rule, title string) bool {
	if !strings.EqualFold(source, "golangci-lint") && !strings.EqualFold(rule, "LINT-GO-typecheck") {
		return false
	}
	if !strings.EqualFold(rule, "LINT-GO-typecheck") {
		return false
	}
	blob := strings.ToLower(title)
	return strings.Contains(blob, "export data") ||
		strings.Contains(blob, "unsupported version") ||
		strings.Contains(blob, "could not import") ||
		strings.Contains(blob, "undefined:") ||
		strings.Contains(blob, "has no field or method")
}

func isToolingPathFileRead(source, rule, file string) bool {
	if !strings.EqualFold(source, "gosec") || !strings.EqualFold(rule, "G304") {
		return false
	}
	lower := strings.ToLower(filepath.ToSlash(file))
	for _, prefix := range []string{
		"scanners/", "patcher/", "preinstall/", "sbom/", "analyzers/", "graph/", "health/",
		"containers/", "operator/", "scripts/",
	} {
		if strings.HasPrefix(lower, prefix) || strings.Contains(lower, "/"+prefix) {
			return true
		}
	}
	return false
}

func isCalibrationSelfReference(source, rule, file string) bool {
	if !strings.EqualFold(rule, "REL-INTERNAL-INFRA-REF") {
		return false
	}
	lower := strings.ToLower(filepath.ToSlash(file))
	return strings.Contains(lower, "calibration/knownsafe")
}

func isEnvExampleCommentBlock(source, rule, file string) bool {
	if !strings.EqualFold(rule, "HEALTH-COMMENT-BLOCK") {
		return false
	}
	lower := strings.ToLower(filepath.ToSlash(file))
	base := filepath.Base(lower)
	return strings.Contains(base, ".env") || strings.HasSuffix(base, ".example") ||
		strings.HasSuffix(base, ".template")
}

func isDebugLiteralInDocs(source, rule, file string) bool {
	if !strings.EqualFold(rule, "QUAL-DEBUG") {
		return false
	}
	lower := strings.ToLower(filepath.ToSlash(file))
	return strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".txt") ||
		strings.HasSuffix(lower, ".rst") || strings.HasPrefix(lower, "docs/") ||
		strings.Contains(lower, "/docs/") || strings.Contains(lower, "wiki")
}
