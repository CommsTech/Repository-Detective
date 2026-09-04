package store

import (
	"fmt"
	"strings"
)

// Canonical operator-facing scan profiles.
const (
	ScanProfileLight    = "light"
	ScanProfileStandard = "standard"
	ScanProfileDeep     = "deep"
	ScanProfileCustom   = "custom"
)

// Legacy profile IDs — still accepted and normalized to a canonical profile.
const (
	ScanProfileFast                  = "fast"
	ScanProfileStandardDeterministic = "standard_deterministic"
	ScanProfileStrictSecurity        = "strict_security"
	ScanProfileMaintainerDeep        = "maintainer_deep"
	ScanProfilePreinstallCautious    = "preinstall_cautious"
	ScanProfileBetaStandard          = "beta_standard"
	ScanProfileHomelabInfra          = "homelab_infra"
)

// AllowedScanProfiles is the operator-facing profile list (UI selects).
var AllowedScanProfiles = []string{
	ScanProfileLight,
	ScanProfileStandard,
	ScanProfileDeep,
	ScanProfileCustom,
}

// ScanProfileOption is one labeled profile choice for UI/API docs.
type ScanProfileOption struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Summary     string `json:"summary"`
	Issues      string `json:"issues"`
	AI          string `json:"ai"`
	Speed       string `json:"speed"`
}

// PrimaryScanProfileOptions is the ordered operator picker.
var PrimaryScanProfileOptions = []ScanProfileOption{
	{
		ID:      ScanProfileLight,
		Label:   "Light",
		Summary: "Fast read-only scan — secrets + vulns quick check. No forge issue submissions.",
		Issues:  "off",
		AI:      "off",
		Speed:   "fast",
	},
	{
		ID:      ScanProfileStandard,
		Label:   "Standard",
		Summary: "Full deterministic scan across security, Go, IaC, health, and graph — files forge issues when policy allows.",
		Issues:  "on",
		AI:      "off",
		Speed:   "normal",
	},
	{
		ID:      ScanProfileDeep,
		Label:   "Deep",
		Summary: "Heavy scan with full workspace analysis and AI cross-checks — slower, highest coverage.",
		Issues:  "on",
		AI:      "on",
		Speed:   "slow",
	},
	{
		ID:      ScanProfileCustom,
		Label:   "Custom",
		Summary: "Manual control — use per-repo / platform toggles only; no preset overrides.",
		Issues:  "as configured",
		AI:      "as configured",
		Speed:   "as configured",
	},
}

// ProfileDescriptions gives a short UI/API summary per canonical profile.
var ProfileDescriptions = map[string]string{
	ScanProfileLight:    PrimaryScanProfileOptions[0].Summary,
	ScanProfileStandard: PrimaryScanProfileOptions[1].Summary,
	ScanProfileDeep:     PrimaryScanProfileOptions[2].Summary,
	ScanProfileCustom:   PrimaryScanProfileOptions[3].Summary,
}

// ProfileLabels are short display names for canonical profiles.
var ProfileLabels = map[string]string{
	ScanProfileLight:    "Light",
	ScanProfileStandard: "Standard",
	ScanProfileDeep:     "Deep",
	ScanProfileCustom:   "Custom",
}

// EffectiveProfileSummary is a high-level view of resolved scan policy.
type EffectiveProfileSummary struct {
	SecurityScanners bool   `json:"security_scanners"`
	GoScanners       bool   `json:"go_scanners"`
	IACScanners      bool   `json:"iac_scanners"`
	HealthChecks     bool   `json:"health_checks"`
	CodeGraph        bool   `json:"code_graph"`
	AI               string `json:"ai"`
	RunnerEligible   bool   `json:"runner_eligible"`
}

// EffectiveSettingsMeta captures profile resolution metadata.
type EffectiveSettingsMeta struct {
	ScanProfile             string
	ProfileModified         bool
	ProfileSource           string
	EffectiveProfileSummary EffectiveProfileSummary
}

// NormalizeScanProfile lowercases, trims, and maps legacy names to canonical profiles.
func NormalizeScanProfile(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	switch n {
	case "", ScanProfileCustom:
		if n == "" {
			return ""
		}
		return ScanProfileCustom
	case ScanProfileLight, ScanProfileFast, ScanProfilePreinstallCautious:
		return ScanProfileLight
	case ScanProfileStandard, ScanProfileStandardDeterministic, ScanProfileBetaStandard,
		ScanProfileHomelabInfra, ScanProfileStrictSecurity, "issue_only":
		return ScanProfileStandard
	case ScanProfileDeep, ScanProfileMaintainerDeep:
		return ScanProfileDeep
	default:
		return n
	}
}

// ScanProfileLabel returns the operator-facing display name for a profile id.
func ScanProfileLabel(name string) string {
	n := NormalizeScanProfile(name)
	if n == "" {
		return ""
	}
	if label, ok := ProfileLabels[n]; ok {
		return label
	}
	return n
}

// ScanProfileDescription returns the short summary for a profile id.
func ScanProfileDescription(name string) string {
	n := NormalizeScanProfile(name)
	if desc, ok := ProfileDescriptions[n]; ok {
		return desc
	}
	return ""
}

// IsValidScanProfile reports whether name is a known (or legacy-aliased) profile.
func IsValidScanProfile(name string) bool {
	n := NormalizeScanProfile(name)
	if n == "" {
		return false
	}
	return containsString(AllowedScanProfiles, n)
}

// ValidateScanProfile returns an error for unknown profile names.
func ValidateScanProfile(name string) error {
	if strings.TrimSpace(name) == "" {
		return nil
	}
	if !IsValidScanProfile(name) {
		return fmt.Errorf("invalid scan_profile %q (allowed: %s)", name, strings.Join(AllowedScanProfiles, ", "))
	}
	return nil
}

// ProfileDefaults returns built-in defaults for a profile layered on DefaultGlobalSettings.
func ProfileDefaults(profile string) EffectiveSettings {
	base := effectiveFromGlobalSnapshot(DefaultGlobalSettings())
	switch NormalizeScanProfile(profile) {
	case ScanProfileLight:
		// Fast read-only — quick scanners, no forge issue submissions.
		base.AnalysisDepth = 2
		base.EnableLLMAuditors = false
		base.AIPolicy = AIPolicyDisabled
		base.PolicyLevel = PolicyMonitorOnly
		base.IssuePolicy = IssuePolicyOff
		base.RemediationPolicy = "off"
		base.EnableTrivy = true
		base.EnableGrype = false
		base.EnableGitleaks = true
		base.EnableSemgrep = false
		base.EnableGovulncheck = false
		base.EnableGosec = false
		base.EnableStaticcheck = false
		base.EnableHadolint = false
		base.EnableCheckov = false
		base.EnableLinters = false
		base.EnableHealthChecks = true
		base.EnableTechDebtChecks = false
		base.EnableReliabilityChecks = false
		base.EnableMaintainabilityChecks = false
		base.EnableTestGapChecks = false
		base.EnablePerformanceChecks = false
		base.EnableAIRiskChecks = false
		base.EnableCodeGraph = false
		return base
	case ScanProfileStandard:
		// Full deterministic coverage with forge issue filing when policy allows.
		base.AnalysisDepth = 2
		base.EnableLLMAuditors = false
		base.AIPolicy = AIPolicyDisabled
		base.SeverityGate = "high"
		base.ConfidenceGate = 0.85
		base.IssuePolicy = IssuePolicyAll
		base.RemediationPolicy = "suggest"
		base.EnableTrivy = true
		base.EnableGrype = true
		base.EnableGitleaks = true
		base.EnableSemgrep = true
		base.EnableGovulncheck = true
		base.EnableGosec = true
		base.EnableStaticcheck = true
		base.EnableHadolint = true
		base.EnableCheckov = true
		base.EnableLinters = true
		base.EnableHealthChecks = true
		base.EnableTechDebtChecks = true
		base.EnableReliabilityChecks = true
		base.EnableMaintainabilityChecks = true
		base.EnableTestGapChecks = true
		base.EnablePerformanceChecks = false
		base.EnableAIRiskChecks = false
		base.EnableCodeGraph = true
		base.GraphIncludeFunctions = false
		base.GraphIncludeFindings = true
		return base
	case ScanProfileDeep:
		// Heavy scan with AI cross-checks.
		base.WorkspaceMode = "auto"
		base.AnalysisDepth = 3
		base.EnableLLMAuditors = true
		base.AIPolicy = AIPolicyAllowed
		base.SeverityGate = "high"
		base.ConfidenceGate = 0.75
		base.IssuePolicy = IssuePolicyAll
		base.RemediationPolicy = "suggest"
		base.EnableTrivy = true
		base.EnableGrype = true
		base.EnableGitleaks = true
		base.EnableSemgrep = true
		base.EnableGovulncheck = true
		base.EnableGosec = true
		base.EnableStaticcheck = true
		base.EnableHadolint = true
		base.EnableCheckov = true
		base.EnableLinters = true
		base.EnableHealthChecks = true
		base.EnableTechDebtChecks = true
		base.EnableReliabilityChecks = true
		base.EnableMaintainabilityChecks = true
		base.EnableTestGapChecks = true
		base.EnablePerformanceChecks = true
		base.EnableAIRiskChecks = false
		base.EnableCodeGraph = true
		base.GraphIncludeFunctions = true
		base.GraphIncludeFindings = true
		base.RunnerPolicy = "auto"
		return base
	default:
		// custom / unknown — leave DefaultGlobalSettings snapshot as-is.
		return base
	}
}

// BuildEffectiveProfileSummary derives badge summary from resolved settings.
func BuildEffectiveProfileSummary(e EffectiveSettings) EffectiveProfileSummary {
	ai := "disabled"
	if e.AIPolicy != AIPolicyDisabled && e.EnableLLMAuditors {
		if e.AnalysisDepth >= 3 {
			ai = "enabled"
		} else {
			ai = "advisory"
		}
	} else if e.AIPolicy == AIPolicyAllowed {
		ai = "advisory"
	}

	security := e.EnableTrivy || e.EnableGrype || e.EnableGitleaks || e.EnableSemgrep
	goScanners := e.EnableGovulncheck || e.EnableGosec || e.EnableStaticcheck
	iac := e.EnableHadolint || e.EnableCheckov
	health := e.EnableHealthChecks
	runnerEligible := e.RunnerPolicy != "core"

	return EffectiveProfileSummary{
		SecurityScanners: security,
		GoScanners:       goScanners,
		IACScanners:      iac,
		HealthChecks:     health,
		CodeGraph:        e.EnableCodeGraph,
		AI:               ai,
		RunnerEligible:   runnerEligible,
	}
}

// EffectiveFromGlobalSnapshot converts a global snapshot to effective settings.
func EffectiveFromGlobalSnapshot(g GlobalSettingsSnapshot) EffectiveSettings {
	return effectiveFromGlobalSnapshot(g)
}

func effectiveFromGlobalSnapshot(g GlobalSettingsSnapshot) EffectiveSettings {
	return EffectiveSettings{
		ScanProfile:                 NormalizeScanProfile(g.ScanProfile),
		Enabled:                     g.Enabled,
		PolicyLevel:                 g.PolicyLevel,
		WorkspaceMode:               g.WorkspaceMode,
		AnalysisDepth:               g.AnalysisDepth,
		EnableLLMAuditors:           g.EnableLLMAuditors,
		EnableTrivy:                 g.EnableTrivy,
		EnableGrype:                 g.EnableGrype,
		EnableGitleaks:              g.EnableGitleaks,
		EnableSemgrep:               g.EnableSemgrep,
		EnableGovulncheck:           g.EnableGovulncheck,
		EnableGosec:                 g.EnableGosec,
		EnableStaticcheck:           g.EnableStaticcheck,
		EnableHadolint:              g.EnableHadolint,
		EnableCheckov:               g.EnableCheckov,
		EnableLinters:               g.EnableLinters,
		SeverityGate:                g.SeverityGate,
		ConfidenceGate:              g.ConfidenceGate,
		IssuePolicy:                 g.IssuePolicy,
		RemediationPolicy:           g.RemediationPolicy,
		RunnerPolicy:                g.RunnerPolicy,
		ScheduleEnabled:             g.ScheduleEnabled,
		ScheduleCron:                g.ScheduleCron,
		AIPolicy:                    g.AIPolicy,
		EnableHealthChecks:          g.EnableHealthChecks,
		EnableTechDebtChecks:        g.EnableTechDebtChecks,
		EnableReliabilityChecks:     g.EnableReliabilityChecks,
		EnableMaintainabilityChecks: g.EnableMaintainabilityChecks,
		EnableTestGapChecks:         g.EnableTestGapChecks,
		EnablePerformanceChecks:     g.EnablePerformanceChecks,
		EnableAIRiskChecks:          g.EnableAIRiskChecks,
		HealthMaxFindings:           g.HealthMaxFindings,
		HealthLargeFileLines:        g.HealthLargeFileLines,
		HealthLargeFunctionLines:    g.HealthLargeFunctionLines,
		HealthMaxNestingDepth:       g.HealthMaxNestingDepth,
		HealthMaxFunctionParams:     g.HealthMaxFunctionParams,
		EnableCodeGraph:             g.EnableCodeGraph,
		GraphMaxNodes:               g.GraphMaxNodes,
		GraphMaxEdges:               g.GraphMaxEdges,
		GraphTimeoutSeconds:         g.GraphTimeoutSeconds,
		GraphIncludeFunctions:       g.GraphIncludeFunctions,
		GraphIncludeFindings:        g.GraphIncludeFindings,
		GovulncheckTimeoutSeconds:   g.GovulncheckTimeoutSeconds,
		GosecTimeoutSeconds:         g.GosecTimeoutSeconds,
		StaticcheckTimeoutSeconds:   g.StaticcheckTimeoutSeconds,
		GoScannerMaxFindings:        g.GoScannerMaxFindings,
		HadolintTimeoutSeconds:      g.HadolintTimeoutSeconds,
		CheckovTimeoutSeconds:       g.CheckovTimeoutSeconds,
		IACScannerMaxFindings:       g.IACScannerMaxFindings,
	}
}
