package analyzers

import (
	"context"

	"git.commsnet.org/commstech/repository-detective/health"
	"git.commsnet.org/commstech/repository-detective/graph"
	"git.commsnet.org/commstech/repository-detective/scanners"
)

// ConfigFromPolicy merges per-scan policy onto the engine base config.
func ConfigFromPolicy(base *Config, policy ScanPolicy, globalAIConfigured bool) *Config {
	if base == nil {
		return &Config{}
	}
	cfg := *base
	cfg.AnalysisDepth = policy.AnalysisDepth
	if cfg.AnalysisDepth <= 0 {
		cfg.AnalysisDepth = 3
	}
	cfg.EnableLLMAuditors = llmEnabledForPolicy(policy, globalAIConfigured)

	cfg.Scanners = base.Scanners
	cfg.Scanners.EnableTrivy = policy.EnableTrivy
	cfg.Scanners.EnableGrype = policy.EnableGrype
	cfg.Scanners.EnableGitleaks = policy.EnableGitleaks
	cfg.Scanners.EnableSemgrep = policy.EnableSemgrep
	cfg.Scanners.EnableGovulncheck = policy.EnableGovulncheck
	cfg.Scanners.EnableGosec = policy.EnableGosec
	cfg.Scanners.EnableStaticcheck = policy.EnableStaticcheck
	cfg.Scanners.EnableHadolint = policy.EnableHadolint
	cfg.Scanners.EnableCheckov = policy.EnableCheckov
	cfg.Scanners.EnableLinters = policy.EnableLinters
	cfg.Scanners.GovulncheckTimeoutSeconds = policyScannerInt(policy.GovulncheckTimeoutSeconds, base.Scanners.GovulncheckTimeoutSeconds)
	cfg.Scanners.GosecTimeoutSeconds = policyScannerInt(policy.GosecTimeoutSeconds, base.Scanners.GosecTimeoutSeconds)
	cfg.Scanners.StaticcheckTimeoutSeconds = policyScannerInt(policy.StaticcheckTimeoutSeconds, base.Scanners.StaticcheckTimeoutSeconds)
	cfg.Scanners.GoScannerMaxFindings = policyScannerInt(policy.GoScannerMaxFindings, base.Scanners.GoScannerMaxFindings)
	cfg.Scanners.HadolintTimeoutSeconds = policyScannerInt(policy.HadolintTimeoutSeconds, base.Scanners.HadolintTimeoutSeconds)
	cfg.Scanners.CheckovTimeoutSeconds = policyScannerInt(policy.CheckovTimeoutSeconds, base.Scanners.CheckovTimeoutSeconds)
	cfg.Scanners.IACScannerMaxFindings = policyScannerInt(policy.IACScannerMaxFindings, base.Scanners.IACScannerMaxFindings)

	cfg.Workspace = scanners.WorkspaceConfig{
		Mode:                  policy.WorkspaceMode,
		MaxSizeMB:             base.Workspace.MaxSizeMB,
		MaxFiles:              base.Workspace.MaxFiles,
		ArchiveTimeoutSeconds: base.Workspace.ArchiveTimeoutSeconds,
	}

	cfg.Health = healthConfigFromPolicy(base.Health, policy)
	cfg.Graph = graphConfigFromPolicy(base.Graph, policy)

	return &cfg
}

func graphConfigFromPolicy(base graph.Config, policy ScanPolicy) graph.Config {
	return graph.Config{
		Enabled:          policy.EnableCodeGraph,
		MaxNodes:         policyGraphInt(policy.GraphMaxNodes, base.MaxNodes),
		MaxEdges:         policyGraphInt(policy.GraphMaxEdges, base.MaxEdges),
		TimeoutSeconds:   policyGraphInt(policy.GraphTimeoutSeconds, base.TimeoutSeconds),
		IncludeFunctions: policy.GraphIncludeFunctions,
		IncludeFindings:  policy.GraphIncludeFindings,
	}
}

func policyGraphInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func healthConfigFromPolicy(base health.Config, policy ScanPolicy) health.Config {
	return health.Config{
		Enabled:               policy.EnableHealthChecks,
		EnableTechDebt:        policy.EnableTechDebtChecks,
		EnableReliability:     policy.EnableReliabilityChecks,
		EnableMaintainability: policy.EnableMaintainabilityChecks,
		EnableTestGap:         policy.EnableTestGapChecks,
		EnablePerformance:     policy.EnablePerformanceChecks,
		EnableAIRisk:          policy.EnableAIRiskChecks,
		MaxFindings:           policyHealthInt(policy.HealthMaxFindings, base.MaxFindings),
		LargeFileLines:        policyHealthInt(policy.HealthLargeFileLines, base.LargeFileLines),
		LargeFunctionLines:    policyHealthInt(policy.HealthLargeFunctionLines, base.LargeFunctionLines),
		MaxNestingDepth:       policyHealthInt(policy.HealthMaxNestingDepth, base.MaxNestingDepth),
		MaxFunctionParams:     policyHealthInt(policy.HealthMaxFunctionParams, base.MaxFunctionParams),
	}
}

func policyHealthInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func policyScannerInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

// ScannersConfigFromSnapshot merges runner job policy onto a base scanner config.
func ScannersConfigFromSnapshot(base scanners.Config, snap PolicySnapshot) scanners.Config {
	cfg := base
	cfg.EnableTrivy = scannerEnabled(snap.EnabledScanners, "trivy")
	cfg.EnableGrype = scannerEnabled(snap.EnabledScanners, "grype")
	cfg.EnableGitleaks = scannerEnabled(snap.EnabledScanners, "gitleaks")
	cfg.EnableSemgrep = scannerEnabled(snap.EnabledScanners, "semgrep")
	cfg.EnableGovulncheck = snap.EnableGovulncheck || scannerEnabled(snap.EnabledScanners, "govulncheck")
	cfg.EnableGosec = snap.EnableGosec || scannerEnabled(snap.EnabledScanners, "gosec")
	cfg.EnableStaticcheck = snap.EnableStaticcheck || scannerEnabled(snap.EnabledScanners, "staticcheck")
	cfg.EnableHadolint = snap.EnableHadolint || scannerEnabled(snap.EnabledScanners, "hadolint")
	cfg.EnableCheckov = snap.EnableCheckov || scannerEnabled(snap.EnabledScanners, "checkov")
	cfg.EnableLinters = scannerEnabled(snap.EnabledScanners, "linters")
	cfg.GovulncheckTimeoutSeconds = policyScannerInt(snap.GovulncheckTimeoutSeconds, base.GovulncheckTimeoutSeconds)
	cfg.GosecTimeoutSeconds = policyScannerInt(snap.GosecTimeoutSeconds, base.GosecTimeoutSeconds)
	cfg.StaticcheckTimeoutSeconds = policyScannerInt(snap.StaticcheckTimeoutSeconds, base.StaticcheckTimeoutSeconds)
	cfg.GoScannerMaxFindings = policyScannerInt(snap.GoScannerMaxFindings, base.GoScannerMaxFindings)
	cfg.HadolintTimeoutSeconds = policyScannerInt(snap.HadolintTimeoutSeconds, base.HadolintTimeoutSeconds)
	cfg.CheckovTimeoutSeconds = policyScannerInt(snap.CheckovTimeoutSeconds, base.CheckovTimeoutSeconds)
	cfg.IACScannerMaxFindings = policyScannerInt(snap.IACScannerMaxFindings, base.IACScannerMaxFindings)
	return cfg
}

func scannerEnabled(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}

// ConfigFor returns the effective analyzer config for a scan context.
func (e *Engine) ConfigFor(ctx context.Context) *Config {
	return e.configFor(ctx)
}

func (e *Engine) configFor(ctx context.Context) *Config {
	policy, ok := ScanPolicyFromContext(ctx)
	if !ok || e.config == nil {
		return e.config
	}
	return ConfigFromPolicy(e.config, policy, e.aiClient != nil)
}

func (e *Engine) llmEnabledFor(ctx context.Context) bool {
	if policy, ok := ScanPolicyFromContext(ctx); ok {
		return llmEnabledForPolicy(policy, e.aiClient != nil)
	}
	return LLMEnabled(e.config, e.aiClient != nil)
}
