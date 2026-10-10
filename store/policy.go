package store

import (
	"strings"
)

const (
	PolicyMonitorOnly        = "monitor_only"
	PolicyIssueOnly          = "issue_only"
	PolicyGatePR             = "gate_pr"
	PolicySuggestFix         = "suggest_fix"
	PolicyAutoPRWithApproval = "auto_pr_with_approval"
	PolicyAutoPRLowRisk      = "auto_pr_low_risk"

	IssuePolicyOff         = "off"
	IssuePolicyFingerprint = "fingerprint"
	IssuePolicyAll         = "all"

	AIPolicyAllowed  = "allowed"
	AIPolicyDisabled = "disabled"
)

// ResolveEffectiveSettings merges profile defaults, global config, repo profile, and overrides with sanitization.
func ResolveEffectiveSettings(global GlobalSettingsSnapshot, repoSettings RepoSettings) EffectiveSettings {
	effective, _ := ResolveEffectiveSettingsWithMeta(global, repoSettings)
	return SanitizeEffectiveSettings(effective, global)
}

// ResolveEffectiveSettingsFull returns effective settings plus profile metadata.
func ResolveEffectiveSettingsFull(global GlobalSettingsSnapshot, repoSettings RepoSettings) (EffectiveSettings, EffectiveSettingsMeta) {
	effective, meta := ResolveEffectiveSettingsWithMeta(global, repoSettings)
	return SanitizeEffectiveSettings(effective, global), meta
}

// SanitizeEffectiveSettings validates enum fields and falls back to global on invalid values.
func SanitizeEffectiveSettings(e EffectiveSettings, global GlobalSettingsSnapshot) EffectiveSettings {
	if !containsString(AllowedPolicyLevels, e.PolicyLevel) {
		e.PolicyLevel = global.PolicyLevel
	}
	if !containsString(AllowedWorkspaceModes, e.WorkspaceMode) {
		e.WorkspaceMode = global.WorkspaceMode
	}
	if !containsInt(AllowedAnalysisDepths, e.AnalysisDepth) {
		e.AnalysisDepth = global.AnalysisDepth
	}
	if !containsString(AllowedSeverities, strings.ToLower(e.SeverityGate)) {
		e.SeverityGate = global.SeverityGate
	}
	if e.ConfidenceGate < 0 || e.ConfidenceGate > 1 {
		e.ConfidenceGate = global.ConfidenceGate
	}
	if !containsString(AllowedIssuePolicies, e.IssuePolicy) {
		e.IssuePolicy = global.IssuePolicy
	}
	if !containsString(AllowedRemediationPolicies, e.RemediationPolicy) {
		e.RemediationPolicy = global.RemediationPolicy
	}
	if !containsString(AllowedRunnerPolicies, e.RunnerPolicy) {
		e.RunnerPolicy = global.RunnerPolicy
	}
	if !containsString(AllowedAIPolicies, e.AIPolicy) {
		e.AIPolicy = global.AIPolicy
	}
	sanitizeHealthThresholds(&e, global)
	sanitizeGraphThresholds(&e, global)
	return e
}

func sanitizeGraphThresholds(e *EffectiveSettings, global GlobalSettingsSnapshot) {
	if e.GraphMaxNodes < GraphMaxNodesMin || e.GraphMaxNodes > GraphMaxNodesMax {
		e.GraphMaxNodes = global.GraphMaxNodes
	}
	if e.GraphMaxEdges < GraphMaxEdgesMin || e.GraphMaxEdges > GraphMaxEdgesMax {
		e.GraphMaxEdges = global.GraphMaxEdges
	}
	if e.GraphTimeoutSeconds < GraphTimeoutSecondsMin || e.GraphTimeoutSeconds > GraphTimeoutSecondsMax {
		e.GraphTimeoutSeconds = global.GraphTimeoutSeconds
	}
}

func sanitizeHealthThresholds(e *EffectiveSettings, global GlobalSettingsSnapshot) {
	if e.HealthMaxFindings < HealthMaxFindingsMin || e.HealthMaxFindings > HealthMaxFindingsMax {
		e.HealthMaxFindings = global.HealthMaxFindings
	}
	if e.HealthLargeFileLines < HealthLargeFileLinesMin || e.HealthLargeFileLines > HealthLargeFileLinesMax {
		e.HealthLargeFileLines = global.HealthLargeFileLines
	}
	if e.HealthLargeFunctionLines < HealthLargeFunctionLinesMin || e.HealthLargeFunctionLines > HealthLargeFunctionLinesMax {
		e.HealthLargeFunctionLines = global.HealthLargeFunctionLines
	}
	if e.HealthMaxNestingDepth < HealthMaxNestingDepthMin || e.HealthMaxNestingDepth > HealthMaxNestingDepthMax {
		e.HealthMaxNestingDepth = global.HealthMaxNestingDepth
	}
	if e.HealthMaxFunctionParams < HealthMaxFunctionParamsMin || e.HealthMaxFunctionParams > HealthMaxFunctionParamsMax {
		e.HealthMaxFunctionParams = global.HealthMaxFunctionParams
	}
}

// LLMEnabledForSettings reports whether LLM pipeline stages should run for a repo.
func LLMEnabledForSettings(e EffectiveSettings, globalAIConfigured bool) bool {
	if e.AIPolicy == AIPolicyDisabled || !globalAIConfigured {
		return false
	}
	if !e.EnableLLMAuditors {
		return false
	}
	depth := e.AnalysisDepth
	if depth <= 0 {
		depth = 3
	}
	return depth >= 3
}

// ShouldCreateForgeIssues reports whether Gitea issues should be created/updated.
func ShouldCreateForgeIssues(e EffectiveSettings) bool {
	if !e.Enabled {
		return false
	}
	if e.PolicyLevel == PolicyMonitorOnly {
		return false
	}
	return e.IssuePolicy != IssuePolicyOff
}

// ApplyReportOnlyDryRunSettings forces analyze+persist without forge filing or remediation.
func ApplyReportOnlyDryRunSettings(e *EffectiveSettings) {
	if e == nil {
		return
	}
	e.PolicyLevel = PolicyMonitorOnly
	e.IssuePolicy = IssuePolicyOff
	e.AIPolicy = AIPolicyDisabled
	e.EnableLLMAuditors = false
	e.EnableAIRiskChecks = false
	e.RemediationPolicy = "off"
}

// ShouldFailCommitStatus reports whether findings can fail commit status.
func ShouldFailCommitStatus(e EffectiveSettings) bool {
	switch normalizePolicyLevel(e.PolicyLevel) {
	case PolicyGatePR, PolicySuggestFix, PolicyAutoPRWithApproval, PolicyAutoPRLowRisk:
		return true
	default:
		return false
	}
}

// IsRemediationPolicyLevel reports reserved remediation policy levels.
func IsRemediationPolicyLevel(level string) bool {
	switch normalizePolicyLevel(level) {
	case PolicySuggestFix, PolicyAutoPRWithApproval, PolicyAutoPRLowRisk:
		return true
	default:
		return false
	}
}

func normalizePolicyLevel(level string) string {
	return strings.ToLower(strings.TrimSpace(level))
}

// SeverityRank orders severities for gate comparison.
func SeverityRank(severity string) int {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical", "crit":
		return 5
	case "high", "error":
		return 4
	case "medium", "warning", "warn":
		return 3
	case "low", "info", "note":
		return 2
	case "informational":
		return 1
	default:
		return 0
	}
}

// PassesIssueGates reports whether a finding meets confidence and severity gates.
func PassesIssueGates(severity string, confidence float64, e EffectiveSettings) bool {
	minConfidence := e.ConfidenceGate
	if minConfidence <= 0 {
		minConfidence = 0.5
	}
	if confidence > 0 && confidence < minConfidence {
		return false
	}
	threshold := strings.ToLower(strings.TrimSpace(e.SeverityGate))
	if threshold == "" {
		threshold = "high"
	}
	return SeverityRank(severity) >= SeverityRank(threshold)
}

// SeveritiesForStatus returns severities that should affect commit status / policy evaluation.
// Filtering uses confidence and severity gates; Observe/Warn/Enforce decide blocking in gitea policy evaluation.
func SeveritiesForStatus(severities []string, confidences []float64, e EffectiveSettings) []string {
	out := make([]string, 0, len(severities))
	for i, severity := range severities {
		conf := 0.0
		if i < len(confidences) {
			conf = confidences[i]
		}
		if PassesIssueGates(severity, conf, e) {
			out = append(out, severity)
		}
	}
	return out
}

// EnabledScannersList returns enabled scanner names for display/persistence.
func EnabledScannersList(e EffectiveSettings) []string {
	var out []string
	if e.EnableTrivy {
		out = append(out, "trivy")
	}
	if e.EnableGrype {
		out = append(out, "grype")
	}
	if e.EnableGitleaks {
		out = append(out, "gitleaks")
	}
	if e.EnableSemgrep {
		out = append(out, "semgrep")
	}
	if e.EnableGovulncheck {
		out = append(out, "govulncheck")
	}
	if e.EnableGosec {
		out = append(out, "gosec")
	}
	if e.EnableStaticcheck {
		out = append(out, "staticcheck")
	}
	if e.EnableHadolint {
		out = append(out, "hadolint")
	}
	if e.EnableCheckov {
		out = append(out, "checkov")
	}
	if e.EnableLinters {
		out = append(out, "linters")
	}
	return out
}
