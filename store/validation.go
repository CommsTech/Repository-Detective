package store

import (
	"fmt"
	"strings"
)

var (
	AllowedPolicyLevels = []string{
		"monitor_only", "issue_only", "gate_pr", "suggest_fix",
		"auto_pr_with_approval", "auto_pr_low_risk",
	}
	AllowedWorkspaceModes      = []string{"api", "archive", "auto"}
	AllowedAnalysisDepths      = []int{1, 2, 3}
	AllowedSeverities          = []string{"critical", "high", "medium", "low", "info"}
	AllowedIssuePolicies       = []string{"off", "fingerprint", "all"}
	AllowedRemediationPolicies = []string{"off", "suggest", "approval", "auto_low_risk"}
	AllowedRunnerPolicies      = []string{"core", "gitea_actions", "auto"}
	AllowedAIPolicies          = []string{"allowed", "disabled"}
)

const (
	HealthMaxFindingsMin        = 1
	HealthMaxFindingsMax        = 1000
	HealthLargeFileLinesMin     = 100
	HealthLargeFileLinesMax     = 50000
	HealthLargeFunctionLinesMin = 20
	HealthLargeFunctionLinesMax = 5000
	HealthMaxNestingDepthMin    = 2
	HealthMaxNestingDepthMax    = 50
	HealthMaxFunctionParamsMin  = 1
	HealthMaxFunctionParamsMax  = 50

	GraphMaxNodesMin       = 100
	GraphMaxNodesMax       = 50000
	GraphMaxEdgesMin       = 100
	GraphMaxEdgesMax       = 200000
	GraphTimeoutSecondsMin = 5
	GraphTimeoutSecondsMax = 1800

	GoScannerTimeoutMin     = 0
	GoScannerTimeoutMax     = 3600
	GoScannerMaxFindingsMin = 1
	GoScannerMaxFindingsMax = 1000

	IACScannerTimeoutMin     = 0
	IACScannerTimeoutMax     = 3600
	IACScannerMaxFindingsMin = 1
	IACScannerMaxFindingsMax = 1000
)

// SettingsUpdate is the API payload for updating repo settings.
type SettingsUpdate struct {
	ScanProfile                 *string  `json:"scan_profile"`
	Enabled                     *bool    `json:"enabled"`
	PolicyLevel                 *string  `json:"policy_level"`
	WorkspaceMode               *string  `json:"workspace_mode"`
	AnalysisDepth               *int     `json:"analysis_depth"`
	EnableLLMAuditors           *bool    `json:"enable_llm_auditors"`
	EnableTrivy                 *bool    `json:"enable_trivy"`
	EnableGrype                 *bool    `json:"enable_grype"`
	EnableGitleaks              *bool    `json:"enable_gitleaks"`
	EnableSemgrep               *bool    `json:"enable_semgrep"`
	EnableGovulncheck           *bool    `json:"enable_govulncheck"`
	EnableGosec                 *bool    `json:"enable_gosec"`
	EnableStaticcheck           *bool    `json:"enable_staticcheck"`
	EnableHadolint              *bool    `json:"enable_hadolint"`
	EnableCheckov               *bool    `json:"enable_checkov"`
	EnableLinters               *bool    `json:"enable_linters"`
	SeverityGate                *string  `json:"severity_gate"`
	ConfidenceGate              *float64 `json:"confidence_gate"`
	IssuePolicy                 *string  `json:"issue_policy"`
	RemediationPolicy           *string  `json:"remediation_policy"`
	RunnerPolicy                *string  `json:"runner_policy"`
	ScheduleEnabled             *bool    `json:"schedule_enabled"`
	ScheduleCron                *string  `json:"schedule_cron"`
	AIPolicy                    *string  `json:"ai_policy"`
	EnableHealthChecks          *bool    `json:"enable_health_checks"`
	EnableTechDebtChecks        *bool    `json:"enable_tech_debt_checks"`
	EnableReliabilityChecks     *bool    `json:"enable_reliability_checks"`
	EnableMaintainabilityChecks *bool    `json:"enable_maintainability_checks"`
	EnableTestGapChecks         *bool    `json:"enable_test_gap_checks"`
	EnablePerformanceChecks     *bool    `json:"enable_performance_checks"`
	EnableAIRiskChecks          *bool    `json:"enable_ai_risk_checks"`
	HealthMaxFindings           *int     `json:"health_max_findings"`
	HealthLargeFileLines        *int     `json:"health_large_file_lines"`
	HealthLargeFunctionLines    *int     `json:"health_large_function_lines"`
	HealthMaxNestingDepth       *int     `json:"health_max_nesting_depth"`
	HealthMaxFunctionParams     *int     `json:"health_max_function_params"`
	EnableCodeGraph             *bool    `json:"enable_code_graph"`
	GraphMaxNodes               *int     `json:"graph_max_nodes"`
	GraphMaxEdges               *int     `json:"graph_max_edges"`
	GraphTimeoutSeconds         *int     `json:"graph_timeout_seconds"`
	GraphIncludeFunctions       *bool    `json:"graph_include_functions"`
	GraphIncludeFindings        *bool    `json:"graph_include_findings"`
	GovulncheckTimeoutSeconds   *int     `json:"govulncheck_timeout_seconds"`
	GosecTimeoutSeconds         *int     `json:"gosec_timeout_seconds"`
	StaticcheckTimeoutSeconds   *int     `json:"staticcheck_timeout_seconds"`
	GoScannerMaxFindings        *int     `json:"go_scanner_max_findings"`
	HadolintTimeoutSeconds      *int     `json:"hadolint_timeout_seconds"`
	CheckovTimeoutSeconds       *int     `json:"checkov_timeout_seconds"`
	IACScannerMaxFindings       *int     `json:"iac_scanner_max_findings"`
	NotificationsEnabled        *bool    `json:"notifications_enabled"`
	NotificationMinSeverity     *string  `json:"notification_min_severity"`
	NotificationEvents          *string  `json:"notification_events"`
	NotificationCooldownSeconds *int     `json:"notification_cooldown_seconds"`
}

// ValidateSettingsUpdate returns an error describing the first invalid field.
func ValidateSettingsUpdate(u SettingsUpdate) error {
	if u.ScanProfile != nil {
		if err := ValidateScanProfile(*u.ScanProfile); err != nil {
			return err
		}
	}
	if u.PolicyLevel != nil && !containsString(AllowedPolicyLevels, *u.PolicyLevel) {
		return fmt.Errorf("invalid policy_level %q", *u.PolicyLevel)
	}
	if u.WorkspaceMode != nil && !containsString(AllowedWorkspaceModes, *u.WorkspaceMode) {
		return fmt.Errorf("invalid workspace_mode %q", *u.WorkspaceMode)
	}
	if u.AnalysisDepth != nil && !containsInt(AllowedAnalysisDepths, *u.AnalysisDepth) {
		return fmt.Errorf("invalid analysis_depth %d (allowed: 1, 2, 3)", *u.AnalysisDepth)
	}
	if u.SeverityGate != nil && !containsString(AllowedSeverities, strings.ToLower(*u.SeverityGate)) {
		return fmt.Errorf("invalid severity_gate %q", *u.SeverityGate)
	}
	if u.ConfidenceGate != nil && (*u.ConfidenceGate < 0 || *u.ConfidenceGate > 1) {
		return fmt.Errorf("confidence_gate must be between 0 and 1")
	}
	if u.IssuePolicy != nil && !containsString(AllowedIssuePolicies, *u.IssuePolicy) {
		return fmt.Errorf("invalid issue_policy %q", *u.IssuePolicy)
	}
	if u.RemediationPolicy != nil && !containsString(AllowedRemediationPolicies, *u.RemediationPolicy) {
		return fmt.Errorf("invalid remediation_policy %q", *u.RemediationPolicy)
	}
	if u.RunnerPolicy != nil && !containsString(AllowedRunnerPolicies, *u.RunnerPolicy) {
		return fmt.Errorf("invalid runner_policy %q", *u.RunnerPolicy)
	}
	if u.AIPolicy != nil && !containsString(AllowedAIPolicies, *u.AIPolicy) {
		return fmt.Errorf("invalid ai_policy %q", *u.AIPolicy)
	}
	if u.ScheduleCron != nil {
		cron := strings.TrimSpace(*u.ScheduleCron)
		if cron != "" {
			if err := ValidateCronExpression(cron); err != nil {
				return fmt.Errorf("invalid schedule_cron: %w", err)
			}
		}
	}
	if err := validateHealthThresholdUpdate(u.HealthMaxFindings, "health_max_findings", HealthMaxFindingsMin, HealthMaxFindingsMax); err != nil {
		return err
	}
	if err := validateHealthThresholdUpdate(u.HealthLargeFileLines, "health_large_file_lines", HealthLargeFileLinesMin, HealthLargeFileLinesMax); err != nil {
		return err
	}
	if err := validateHealthThresholdUpdate(u.HealthLargeFunctionLines, "health_large_function_lines", HealthLargeFunctionLinesMin, HealthLargeFunctionLinesMax); err != nil {
		return err
	}
	if err := validateHealthThresholdUpdate(u.HealthMaxNestingDepth, "health_max_nesting_depth", HealthMaxNestingDepthMin, HealthMaxNestingDepthMax); err != nil {
		return err
	}
	if err := validateHealthThresholdUpdate(u.HealthMaxFunctionParams, "health_max_function_params", HealthMaxFunctionParamsMin, HealthMaxFunctionParamsMax); err != nil {
		return err
	}
	if err := validateHealthThresholdUpdate(u.GraphMaxNodes, "graph_max_nodes", GraphMaxNodesMin, GraphMaxNodesMax); err != nil {
		return err
	}
	if err := validateHealthThresholdUpdate(u.GraphMaxEdges, "graph_max_edges", GraphMaxEdgesMin, GraphMaxEdgesMax); err != nil {
		return err
	}
	if err := validateHealthThresholdUpdate(u.GraphTimeoutSeconds, "graph_timeout_seconds", GraphTimeoutSecondsMin, GraphTimeoutSecondsMax); err != nil {
		return err
	}
	if err := validateHealthThresholdUpdate(u.GovulncheckTimeoutSeconds, "govulncheck_timeout_seconds", GoScannerTimeoutMin, GoScannerTimeoutMax); err != nil {
		return err
	}
	if err := validateHealthThresholdUpdate(u.GosecTimeoutSeconds, "gosec_timeout_seconds", GoScannerTimeoutMin, GoScannerTimeoutMax); err != nil {
		return err
	}
	if err := validateHealthThresholdUpdate(u.StaticcheckTimeoutSeconds, "staticcheck_timeout_seconds", GoScannerTimeoutMin, GoScannerTimeoutMax); err != nil {
		return err
	}
	if err := validateHealthThresholdUpdate(u.GoScannerMaxFindings, "go_scanner_max_findings", GoScannerMaxFindingsMin, GoScannerMaxFindingsMax); err != nil {
		return err
	}
	if err := validateHealthThresholdUpdate(u.HadolintTimeoutSeconds, "hadolint_timeout_seconds", IACScannerTimeoutMin, IACScannerTimeoutMax); err != nil {
		return err
	}
	if err := validateHealthThresholdUpdate(u.CheckovTimeoutSeconds, "checkov_timeout_seconds", IACScannerTimeoutMin, IACScannerTimeoutMax); err != nil {
		return err
	}
	if err := validateHealthThresholdUpdate(u.IACScannerMaxFindings, "iac_scanner_max_findings", IACScannerMaxFindingsMin, IACScannerMaxFindingsMax); err != nil {
		return err
	}
	if u.NotificationMinSeverity != nil {
		sev := strings.ToLower(strings.TrimSpace(*u.NotificationMinSeverity))
		if sev != "" && !containsString(AllowedSeverities, sev) {
			return fmt.Errorf("invalid notification_min_severity %q", sev)
		}
	}
	if u.NotificationEvents != nil {
		if err := ValidateNotificationEventsCSV(*u.NotificationEvents); err != nil {
			return err
		}
	}
	if u.NotificationCooldownSeconds != nil && (*u.NotificationCooldownSeconds < 0 || *u.NotificationCooldownSeconds > 86400) {
		return fmt.Errorf("notification_cooldown_seconds must be between 0 and 86400")
	}
	return nil
}

func validateHealthThresholdUpdate(v *int, name string, min, max int) error {
	if v == nil {
		return nil
	}
	if *v < min || *v > max {
		return fmt.Errorf("%s must be between %d and %d", name, min, max)
	}
	return nil
}

// ValidateRepoSettings validates merged repo settings (e.g. after apply).
func ValidateRepoSettings(settings RepoSettings) error {
	if settings.ScheduleEnabled != nil && *settings.ScheduleEnabled {
		cron := ""
		if settings.ScheduleCron != nil {
			cron = strings.TrimSpace(*settings.ScheduleCron)
		}
		if cron == "" {
			return fmt.Errorf("schedule_cron required when schedule_enabled is true")
		}
		if err := ValidateCronExpression(cron); err != nil {
			return fmt.Errorf("invalid schedule_cron: %w", err)
		}
	}
	if settings.ScheduleCron != nil {
		cron := strings.TrimSpace(*settings.ScheduleCron)
		if cron != "" {
			if err := ValidateCronExpression(cron); err != nil {
				return fmt.Errorf("invalid schedule_cron: %w", err)
			}
		}
	}
	return nil
}

// ApplySettingsUpdate merges a validated update onto existing settings.
func ApplySettingsUpdate(existing RepoSettings, u SettingsUpdate) RepoSettings {
	existing.ScanProfile = u.ScanProfile
	existing.Enabled = u.Enabled
	existing.PolicyLevel = u.PolicyLevel
	existing.WorkspaceMode = u.WorkspaceMode
	existing.AnalysisDepth = u.AnalysisDepth
	existing.EnableLLMAuditors = u.EnableLLMAuditors
	existing.EnableTrivy = u.EnableTrivy
	existing.EnableGrype = u.EnableGrype
	existing.EnableGitleaks = u.EnableGitleaks
	existing.EnableSemgrep = u.EnableSemgrep
	existing.EnableGovulncheck = u.EnableGovulncheck
	existing.EnableGosec = u.EnableGosec
	existing.EnableStaticcheck = u.EnableStaticcheck
	existing.EnableHadolint = u.EnableHadolint
	existing.EnableCheckov = u.EnableCheckov
	existing.EnableLinters = u.EnableLinters
	if u.SeverityGate != nil {
		lower := strings.ToLower(*u.SeverityGate)
		existing.SeverityGate = &lower
	} else {
		existing.SeverityGate = u.SeverityGate
	}
	existing.ConfidenceGate = u.ConfidenceGate
	existing.IssuePolicy = u.IssuePolicy
	existing.RemediationPolicy = u.RemediationPolicy
	existing.RunnerPolicy = u.RunnerPolicy
	existing.ScheduleEnabled = u.ScheduleEnabled
	existing.ScheduleCron = u.ScheduleCron
	existing.AIPolicy = u.AIPolicy
	existing.EnableHealthChecks = u.EnableHealthChecks
	existing.EnableTechDebtChecks = u.EnableTechDebtChecks
	existing.EnableReliabilityChecks = u.EnableReliabilityChecks
	existing.EnableMaintainabilityChecks = u.EnableMaintainabilityChecks
	existing.EnableTestGapChecks = u.EnableTestGapChecks
	existing.EnablePerformanceChecks = u.EnablePerformanceChecks
	existing.EnableAIRiskChecks = u.EnableAIRiskChecks
	existing.HealthMaxFindings = u.HealthMaxFindings
	existing.HealthLargeFileLines = u.HealthLargeFileLines
	existing.HealthLargeFunctionLines = u.HealthLargeFunctionLines
	existing.HealthMaxNestingDepth = u.HealthMaxNestingDepth
	existing.HealthMaxFunctionParams = u.HealthMaxFunctionParams
	existing.EnableCodeGraph = u.EnableCodeGraph
	existing.GraphMaxNodes = u.GraphMaxNodes
	existing.GraphMaxEdges = u.GraphMaxEdges
	existing.GraphTimeoutSeconds = u.GraphTimeoutSeconds
	existing.GraphIncludeFunctions = u.GraphIncludeFunctions
	existing.GraphIncludeFindings = u.GraphIncludeFindings
	existing.GovulncheckTimeoutSeconds = u.GovulncheckTimeoutSeconds
	existing.GosecTimeoutSeconds = u.GosecTimeoutSeconds
	existing.StaticcheckTimeoutSeconds = u.StaticcheckTimeoutSeconds
	existing.GoScannerMaxFindings = u.GoScannerMaxFindings
	existing.HadolintTimeoutSeconds = u.HadolintTimeoutSeconds
	existing.CheckovTimeoutSeconds = u.CheckovTimeoutSeconds
	existing.IACScannerMaxFindings = u.IACScannerMaxFindings
	existing.NotificationsEnabled = u.NotificationsEnabled
	existing.NotificationMinSeverity = u.NotificationMinSeverity
	existing.NotificationEvents = u.NotificationEvents
	existing.NotificationCooldownSeconds = u.NotificationCooldownSeconds
	return existing
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func containsInt(values []int, target int) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
