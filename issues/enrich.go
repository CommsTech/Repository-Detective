package issues

import (
	"strings"

	"git.commsnet.org/commstech/repository-detective/ai"
)

// BuildLabels returns Repository Detective labels for Gitea issue submission.
func BuildLabels(base []string, issue *ai.CodeIssue) []string {
	labels := append([]string{}, DefaultIssueBaseLabels()...)
	labels = append(labels, base...)

	category := NormalizeCategory(issue.Category, issue.Source)
	labels = append(labels, CategoryLabelForWrite(category))

	if severityLabel := SeverityLabel(issue.Severity); severityLabel != "" {
		labels = append(labels, severityLabel)
	}

	lifecycle := issue.LifecycleState
	switch {
	case ConfidenceNeedsHumanReview(issue.Confidence):
		labels = append(labels, ExpandLifecycleLabels(LifecycleNeedsHumanReview)...)
	case lifecycle != "":
		labels = append(labels, ExpandLifecycleLabels(lifecycle)...)
	default:
		labels = append(labels, ExpandLifecycleLabels(LifecycleOpen)...)
	}

	return uniqueStrings(labels)
}

// SeverityLabel maps issue severity to a label slug.
func SeverityLabel(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return "severity/critical"
	case "high":
		return "severity/high"
	case "medium", "warning", "warn":
		return "severity/medium"
	case "low":
		return "severity/low"
	case "info", "informational", "note":
		return "severity/info"
	default:
		return ""
	}
}

// EnrichIssues normalizes every issue in a scan result (fingerprints, categories, etc.).
func EnrichIssues(repository string, scanID string, codeIssues []ai.CodeIssue) {
	for i := range codeIssues {
		EnrichIssue(repository, &codeIssues[i], scanID)
	}
}

// EnrichIssue normalizes category, fingerprint, and remediation metadata on an issue.
func EnrichIssue(repository string, issue *ai.CodeIssue, scanID string) {
	if issue == nil {
		return
	}

	if issue.Source == "" {
		issue.Source = "unknown"
	}

	issue.Category = NormalizeCategory(issue.Category, issue.Source)

	if issue.RuleID == "" {
		issue.RuleID = issue.ClusterID
	}
	issue.RuleID = strings.Trim(strings.TrimSpace(issue.RuleID), "`\"'")

	if issue.Fingerprint == "" {
		issue.Fingerprint = ComputeFingerprint(FingerprintFromIssue(repository, issue))
	}

	if issue.ScanID == "" {
		issue.ScanID = scanID
	}

	issue.CodeSnippet = SanitizeSecretEvidence(issue.CodeSnippet)

	if issue.RegressionRisk == "" {
		issue.RegressionRisk = defaultRegressionRisk(issue)
	}
	if issue.Fixable == "" {
		issue.Fixable = defaultFixable(issue)
	}
	if issue.FixComplexity == "" {
		issue.FixComplexity = defaultFixComplexity(issue)
	}
	if issue.RequiredTests == "" {
		issue.RequiredTests = defaultRequiredTests(issue)
	}
	if issue.SuggestedPatchStrategy == "" {
		issue.SuggestedPatchStrategy = defaultPatchStrategy(issue)
	}

	issue.SafeForAutoPR = defaultSafeForAutoPR(issue)

	if ConfidenceNeedsHumanReview(issue.Confidence) {
		issue.LifecycleState = LifecycleNeedsHumanReview
	} else if issue.LifecycleState == "" {
		issue.LifecycleState = LifecycleOpen
	}

	issue.FromAI = IsAIAuditorSource(issue.Source)
}

// IsAIAuditorSource reports whether findings from this source are AI-generated rather than deterministic scanners.
func IsAIAuditorSource(source string) bool {
	return isAIAuditor(source)
}

func isAIAuditor(source string) bool {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "static", "trivy", "grype", "gitleaks", "semgrep", "golangci-lint", "ruff", "shellcheck",
		"hadolint", "staticcheck", "checkov", "gosec", "govulncheck":
		return false
	default:
		return source != "" && source != "unknown"
	}
}

func defaultRegressionRisk(issue *ai.CodeIssue) string {
	switch strings.ToLower(issue.Severity) {
	case "critical", "high":
		return "high"
	case "medium":
		return "medium"
	default:
		return "low"
	}
}

func defaultFixable(issue *ai.CodeIssue) string {
	switch NormalizeCategory(issue.Category, issue.Source) {
	case CategorySecret, CategoryDependency, CategoryCodeQuality, CategoryMaintainability:
		return "true"
	case CategorySecurity:
		if issue.FromAI {
			return "unknown"
		}
		return "unknown"
	default:
		return "unknown"
	}
}

func defaultFixComplexity(issue *ai.CodeIssue) string {
	switch NormalizeCategory(issue.Category, issue.Source) {
	case CategoryCodeQuality, CategoryMaintainability:
		return "small"
	case CategoryDependency:
		return "small"
	case CategorySecret:
		return "medium"
	default:
		return "medium"
	}
}

func defaultRequiredTests(issue *ai.CodeIssue) string {
	if issue.File == "" {
		return "Run the project's standard test suite after applying a fix."
	}
	return "Add or run tests covering `" + issue.File + "` and verify the reported behavior is resolved."
}

func defaultPatchStrategy(issue *ai.CodeIssue) string {
	switch NormalizeCategory(issue.Category, issue.Source) {
	case CategorySecret:
		return "Remove secret, rotate credential, inject via secret manager"
	case CategoryDependency:
		return "Bump dependency version and verify lockfiles"
	case CategoryCodeQuality, CategoryMaintainability:
		return "Apply lint-guided refactor with no behavior change"
	default:
		return "Minimal targeted fix with regression test"
	}
}

func defaultSafeForAutoPR(issue *ai.CodeIssue) bool {
	if issue.FromAI {
		return false
	}
	if ConfidenceNeedsHumanReview(issue.Confidence) {
		return false
	}
	switch NormalizeCategory(issue.Category, issue.Source) {
	case CategoryCodeQuality, CategoryMaintainability, CategoryDependency:
		return issue.FixComplexity == "small"
	default:
		return false
	}
}
