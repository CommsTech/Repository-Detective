package store

import (
	"fmt"
	"strings"
)

const (
	ScanProfileFast                  = "fast"
	ScanProfileStandardDeterministic = "standard_deterministic"
	ScanProfileStrictSecurity        = "strict_security"
	ScanProfileMaintainerDeep        = "maintainer_deep"
	ScanProfilePreinstallCautious    = "preinstall_cautious"
	ScanProfileBetaStandard          = "beta_standard"
	ScanProfileHomelabInfra          = "homelab_infra"
	ScanProfileCustom                = "custom"
)

// AllowedScanProfiles lists valid scan profile names.
var AllowedScanProfiles = []string{
	ScanProfileFast,
	ScanProfileStandardDeterministic,
	ScanProfileBetaStandard,
	ScanProfileHomelabInfra,
	ScanProfileStrictSecurity,
	ScanProfileMaintainerDeep,
	ScanProfilePreinstallCautious,
	ScanProfileCustom,
}

// ProfileDescriptions gives a short UI/API summary per profile.
var ProfileDescriptions = map[string]string{
	ScanProfileFast:                  "Quick feedback, low noise — gitleaks + trivy, minimal health, no AI",
	ScanProfileStandardDeterministic: "Default deterministic scan — security, Go, IaC, health, and graph",
	ScanProfileBetaStandard:          "Private beta default — deterministic, low issue noise, graph on dashboard only",
	ScanProfileHomelabInfra:          "Homelab/infra repos — deterministic scan with internal-ref and graph noise calibration",
	ScanProfileStrictSecurity:        "Strong PR/security gate — all scanners, medium severity gate, status gate",
	ScanProfileMaintainerDeep:        "Deep maintenance — full workspace, scanners, health, graph, and LLM auditors",
	ScanProfilePreinstallCautious:    "Third-party trust assessment — no issues/AI, conservative scanners",
	ScanProfileCustom:                "Manual control — explicit repo toggles only, no profile overrides",
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

// NormalizeScanProfile lowercases and trims a profile name.
func NormalizeScanProfile(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// IsValidScanProfile reports whether name is a known profile.
func IsValidScanProfile(name string) bool {
	return containsString(AllowedScanProfiles, NormalizeScanProfile(name))
}

// ValidateScanProfile returns an error for unknown profile names.
func ValidateScanProfile(name string) error {
	if name == "" {
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
	case ScanProfileFast:
		base.AnalysisDepth = 2
		base.EnableLLMAuditors = false
		base.AIPolicy = AIPolicyDisabled
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
	case ScanProfileStandardDeterministic:
		base.AnalysisDepth = 2
		base.EnableLLMAuditors = false
		base.AIPolicy = AIPolicyDisabled
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
		return base
	case ScanProfileBetaStandard:
		base = ProfileDefaults(ScanProfileStandardDeterministic)
		base.AnalysisDepth = 2
		base.EnableLLMAuditors = false
		base.AIPolicy = AIPolicyDisabled
		base.SeverityGate = "high"
		base.ConfidenceGate = 0.85
		base.IssuePolicy = IssuePolicyAll
		base.RemediationPolicy = "suggest"
		return base
	case ScanProfileHomelabInfra:
		base = ProfileDefaults(ScanProfileStandardDeterministic)
		base.AnalysisDepth = 2
		base.EnableLLMAuditors = false
		base.AIPolicy = AIPolicyDisabled
		base.SeverityGate = "high"
		base.ConfidenceGate = 0.75
		base.IssuePolicy = IssuePolicyAll
		base.RemediationPolicy = "off"
		base.EnableCodeGraph = true
		base.GraphIncludeFunctions = true
		base.GraphIncludeFindings = true
		return base
	case ScanProfileStrictSecurity:
		base.AnalysisDepth = 2
		base.PolicyLevel = PolicyGatePR
		base.EnableLLMAuditors = false
		base.AIPolicy = AIPolicyAllowed
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
		base.SeverityGate = "medium"
		base.ConfidenceGate = 0.85
		base.EnableHealthChecks = true
		base.EnableTechDebtChecks = true
		base.EnableReliabilityChecks = true
		base.EnableMaintainabilityChecks = true
		base.EnableTestGapChecks = false
		base.EnablePerformanceChecks = false
		base.EnableAIRiskChecks = false
		base.EnableCodeGraph = true
		return base
	case ScanProfileMaintainerDeep:
		base.WorkspaceMode = "auto"
		base.AnalysisDepth = 3
		base.EnableLLMAuditors = true
		base.AIPolicy = AIPolicyAllowed
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
	case ScanProfilePreinstallCautious:
		base.AnalysisDepth = 2
		base.PolicyLevel = PolicyMonitorOnly
		base.EnableLLMAuditors = false
		base.AIPolicy = AIPolicyDisabled
		base.IssuePolicy = IssuePolicyOff
		base.EnableTrivy = true
		base.EnableGrype = false
		base.EnableGitleaks = true
		base.EnableSemgrep = true
		base.EnableGovulncheck = true
		base.EnableGosec = true
		base.EnableStaticcheck = true
		base.EnableHadolint = true
		base.EnableCheckov = true
		base.EnableLinters = false
		base.EnableHealthChecks = true
		base.EnableTechDebtChecks = true
		base.EnableReliabilityChecks = true
		base.EnableMaintainabilityChecks = true
		base.EnableTestGapChecks = false
		base.EnablePerformanceChecks = false
		base.EnableAIRiskChecks = false
		base.EnableCodeGraph = true
		base.GraphIncludeFunctions = true
		return base
	default:
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
