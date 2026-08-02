package store

import (
	"fmt"
	"strings"
)

// PlatformSettings holds operator-editable platform defaults persisted in SQLite.
// Secrets are never stored here — only non-secret toggles and numeric/string policy.
type PlatformSettings struct {
	ScanProfile string `json:"scan_profile,omitempty"`

	SchedulerEnabled     *bool  `json:"scheduler_enabled,omitempty"`
	NotificationsEnabled *bool  `json:"notifications_enabled,omitempty"`
	PreinstallAuditEnabled *bool `json:"preinstall_audit_enabled,omitempty"`
	RemediationPlannerEnabled *bool `json:"remediation_planner_enabled,omitempty"`
	RemediationPREnabled *bool `json:"remediation_pr_enabled,omitempty"`
	EvidenceClosureEnabled *bool `json:"evidence_closure_enabled,omitempty"`

	AutoCreateIssues *bool   `json:"auto_create_issues,omitempty"`
	IssuePolicy      string  `json:"issue_policy,omitempty"`
	RemediationPolicy string `json:"remediation_policy,omitempty"`
	SeverityGate     string  `json:"severity_gate,omitempty"`
	ConfidenceGate   *float64 `json:"confidence_gate,omitempty"`
	AnalysisDepth    *int    `json:"analysis_depth,omitempty"`
	ScheduleCron     string  `json:"schedule_cron,omitempty"`
	ScheduleEnabled  *bool   `json:"schedule_enabled,omitempty"`

	EnableTrivy       *bool `json:"enable_trivy,omitempty"`
	EnableGrype       *bool `json:"enable_grype,omitempty"`
	EnableGitleaks    *bool `json:"enable_gitleaks,omitempty"`
	EnableSemgrep     *bool `json:"enable_semgrep,omitempty"`
	EnableGovulncheck *bool `json:"enable_govulncheck,omitempty"`
	EnableGosec       *bool `json:"enable_gosec,omitempty"`
	EnableStaticcheck *bool `json:"enable_staticcheck,omitempty"`
	EnableHadolint    *bool `json:"enable_hadolint,omitempty"`
	EnableCheckov     *bool `json:"enable_checkov,omitempty"`
	EnableLinters     *bool `json:"enable_linters,omitempty"`
	EnablePerformanceChecks *bool `json:"enable_performance_checks,omitempty"`
	EnableCodeGraph   *bool `json:"enable_code_graph,omitempty"`

	AIRecommendationsEnabled           *bool `json:"ai_recommendations_enabled,omitempty"`
	AIRecommendationsMaxTokensPerScan  *int  `json:"ai_recommendations_max_tokens_per_scan,omitempty"`
	AIRecommendationsTokenBudgetPerScan *int `json:"ai_recommendations_token_budget_per_scan,omitempty"`
	AIRecommendationsMaxFindingsPerScan *int `json:"ai_recommendations_max_findings_per_scan,omitempty"`

	UpdatedAt string `json:"-"`
	UpdatedBy string `json:"-"`
}

// ValidatePlatformSettings checks operator-submitted platform settings.
func ValidatePlatformSettings(s PlatformSettings) error {
	if p := strings.TrimSpace(s.ScanProfile); p != "" {
		if err := ValidateScanProfile(p); err != nil {
			return err
		}
	}
	if s.IssuePolicy != "" {
		p := strings.ToLower(strings.TrimSpace(s.IssuePolicy))
		switch p {
		case IssuePolicyAll, IssuePolicyOff, IssuePolicyFingerprint:
			s.IssuePolicy = p
		default:
			return fmt.Errorf("invalid issue_policy %q (allowed: all, off, fingerprint)", s.IssuePolicy)
		}
	}
	if s.RemediationPolicy != "" {
		switch strings.ToLower(strings.TrimSpace(s.RemediationPolicy)) {
		case "off", "suggest", "auto":
			// ok
		default:
			return fmt.Errorf("invalid remediation_policy %q", s.RemediationPolicy)
		}
	}
	if s.SeverityGate != "" {
		switch strings.ToLower(strings.TrimSpace(s.SeverityGate)) {
		case "critical", "high", "medium", "low", "info":
			// ok
		default:
			return fmt.Errorf("invalid severity_gate %q", s.SeverityGate)
		}
	}
	if s.AnalysisDepth != nil && (*s.AnalysisDepth < 1 || *s.AnalysisDepth > 3) {
		return fmt.Errorf("analysis_depth must be 1–3")
	}
	if s.ConfidenceGate != nil && (*s.ConfidenceGate < 0 || *s.ConfidenceGate > 1) {
		return fmt.Errorf("confidence_gate must be between 0 and 1")
	}
	if s.AIRecommendationsMaxTokensPerScan != nil && *s.AIRecommendationsMaxTokensPerScan < 0 {
		return fmt.Errorf("ai_recommendations_max_tokens_per_scan must be >= 0")
	}
	if s.AIRecommendationsTokenBudgetPerScan != nil && *s.AIRecommendationsTokenBudgetPerScan < 0 {
		return fmt.Errorf("ai_recommendations_token_budget_per_scan must be >= 0")
	}
	if s.AIRecommendationsMaxFindingsPerScan != nil && *s.AIRecommendationsMaxFindingsPerScan < 0 {
		return fmt.Errorf("ai_recommendations_max_findings_per_scan must be >= 0")
	}
	return nil
}

// ApplyPlatformSettingsToGlobal overlays platform settings onto a global snapshot.
func ApplyPlatformSettingsToGlobal(base GlobalSettingsSnapshot, s PlatformSettings) GlobalSettingsSnapshot {
	out := base
	if p := strings.TrimSpace(s.ScanProfile); p != "" {
		out.ScanProfile = NormalizeScanProfile(p)
	}
	if s.AnalysisDepth != nil {
		out.AnalysisDepth = *s.AnalysisDepth
	}
	if s.SeverityGate != "" {
		out.SeverityGate = strings.ToLower(strings.TrimSpace(s.SeverityGate))
	}
	if s.ConfidenceGate != nil {
		out.ConfidenceGate = *s.ConfidenceGate
	}
	if s.IssuePolicy != "" {
		out.IssuePolicy = strings.ToLower(strings.TrimSpace(s.IssuePolicy))
	} else if s.AutoCreateIssues != nil {
		if *s.AutoCreateIssues {
			out.IssuePolicy = IssuePolicyAll
		} else {
			out.IssuePolicy = IssuePolicyOff
		}
	}
	if s.RemediationPolicy != "" {
		out.RemediationPolicy = strings.ToLower(strings.TrimSpace(s.RemediationPolicy))
	}
	if s.ScheduleEnabled != nil {
		out.ScheduleEnabled = *s.ScheduleEnabled
	}
	if cron := strings.TrimSpace(s.ScheduleCron); cron != "" {
		out.ScheduleCron = cron
	}
	applyBool(&out.EnableTrivy, s.EnableTrivy)
	applyBool(&out.EnableGrype, s.EnableGrype)
	applyBool(&out.EnableGitleaks, s.EnableGitleaks)
	applyBool(&out.EnableSemgrep, s.EnableSemgrep)
	applyBool(&out.EnableGovulncheck, s.EnableGovulncheck)
	applyBool(&out.EnableGosec, s.EnableGosec)
	applyBool(&out.EnableStaticcheck, s.EnableStaticcheck)
	applyBool(&out.EnableHadolint, s.EnableHadolint)
	applyBool(&out.EnableCheckov, s.EnableCheckov)
	applyBool(&out.EnableLinters, s.EnableLinters)
	applyBool(&out.EnablePerformanceChecks, s.EnablePerformanceChecks)
	applyBool(&out.EnableCodeGraph, s.EnableCodeGraph)
	return out
}

func applyBool(dst *bool, src *bool) {
	if src != nil {
		*dst = *src
	}
}

// BoolPtr returns a pointer to b.
func BoolPtr(b bool) *bool { return &b }

// IntPtr returns a pointer to n.
func IntPtr(n int) *int { return &n }

// Float64Ptr returns a pointer to f.
func Float64Ptr(f float64) *float64 { return &f }
