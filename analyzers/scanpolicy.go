package analyzers

import "context"

// ScanPolicy is the per-scan effective settings applied to the analysis engine.
type ScanPolicy struct {
	ScanProfile       string
	ProfileModified   bool
	ProfileSource     string
	Enabled           bool
	PolicyLevel       string
	WorkspaceMode     string
	AnalysisDepth     int
	EnableLLMAuditors bool
	EnableTrivy       bool
	EnableGrype       bool
	EnableGitleaks    bool
	EnableSemgrep     bool
	EnableGovulncheck bool
	EnableGosec       bool
	EnableStaticcheck bool
	EnableHadolint    bool
	EnableCheckov     bool
	EnableLinters     bool
	SeverityGate      string
	ConfidenceGate    float64
	IssuePolicy       string
	RemediationPolicy string
	AIPolicy          string
	EnableHealthChecks          bool
	EnableTechDebtChecks        bool
	EnableReliabilityChecks     bool
	EnableMaintainabilityChecks bool
	EnableTestGapChecks         bool
	EnablePerformanceChecks     bool
	EnableAIRiskChecks          bool
	HealthMaxFindings           int
	HealthLargeFileLines        int
	HealthLargeFunctionLines    int
	HealthMaxNestingDepth       int
	HealthMaxFunctionParams     int
	EnableCodeGraph             bool
	GraphMaxNodes               int
	GraphMaxEdges               int
	GraphTimeoutSeconds         int
	GraphIncludeFunctions       bool
	GraphIncludeFindings        bool
	GovulncheckTimeoutSeconds   int
	GosecTimeoutSeconds         int
	StaticcheckTimeoutSeconds   int
	GoScannerMaxFindings        int
	HadolintTimeoutSeconds      int
	CheckovTimeoutSeconds       int
	IACScannerMaxFindings       int
}

// PolicySnapshot is persisted on scan rows for auditability.
type PolicySnapshot struct {
	ScanProfile       string  `json:"scan_profile,omitempty"`
	ProfileModified   bool    `json:"profile_modified,omitempty"`
	ProfileSource     string  `json:"profile_source,omitempty"`
	WorkspaceMode    string   `json:"workspace_mode"`
	AnalysisDepth    int      `json:"analysis_depth"`
	AIPolicy         string   `json:"ai_policy"`
	EnabledScanners  []string `json:"enabled_scanners"`
	PolicyLevel      string   `json:"policy_level"`
	IssuePolicy      string   `json:"issue_policy"`
	SeverityGate     string   `json:"severity_gate"`
	ConfidenceGate   float64  `json:"confidence_gate"`
	RemediationPolicy string  `json:"remediation_policy,omitempty"`
	EnableHealthChecks          bool `json:"enable_health_checks,omitempty"`
	EnableTechDebtChecks        bool `json:"enable_tech_debt_checks,omitempty"`
	EnableReliabilityChecks     bool `json:"enable_reliability_checks,omitempty"`
	EnableMaintainabilityChecks bool `json:"enable_maintainability_checks,omitempty"`
	EnableTestGapChecks         bool `json:"enable_test_gap_checks,omitempty"`
	EnablePerformanceChecks     bool `json:"enable_performance_checks,omitempty"`
	EnableAIRiskChecks          bool `json:"enable_ai_risk_checks,omitempty"`
	HealthMaxFindings           int  `json:"health_max_findings,omitempty"`
	HealthLargeFileLines        int  `json:"health_large_file_lines,omitempty"`
	HealthLargeFunctionLines    int  `json:"health_large_function_lines,omitempty"`
	HealthMaxNestingDepth       int  `json:"health_max_nesting_depth,omitempty"`
	HealthMaxFunctionParams     int  `json:"health_max_function_params,omitempty"`
	EnableCodeGraph             bool `json:"enable_code_graph,omitempty"`
	GraphMaxNodes               int  `json:"graph_max_nodes,omitempty"`
	GraphMaxEdges               int  `json:"graph_max_edges,omitempty"`
	GraphTimeoutSeconds         int  `json:"graph_timeout_seconds,omitempty"`
	GraphIncludeFunctions       bool `json:"graph_include_functions,omitempty"`
	GraphIncludeFindings        bool `json:"graph_include_findings,omitempty"`
	EnableGovulncheck             bool `json:"enable_govulncheck,omitempty"`
	EnableGosec                   bool `json:"enable_gosec,omitempty"`
	EnableStaticcheck             bool `json:"enable_staticcheck,omitempty"`
	GovulncheckTimeoutSeconds     int  `json:"govulncheck_timeout_seconds,omitempty"`
	GosecTimeoutSeconds           int  `json:"gosec_timeout_seconds,omitempty"`
	StaticcheckTimeoutSeconds     int  `json:"staticcheck_timeout_seconds,omitempty"`
	GoScannerMaxFindings          int  `json:"go_scanner_max_findings,omitempty"`
	EnableHadolint                bool `json:"enable_hadolint,omitempty"`
	EnableCheckov                 bool `json:"enable_checkov,omitempty"`
	HadolintTimeoutSeconds        int  `json:"hadolint_timeout_seconds,omitempty"`
	CheckovTimeoutSeconds         int  `json:"checkov_timeout_seconds,omitempty"`
	IACScannerMaxFindings         int  `json:"iac_scanner_max_findings,omitempty"`
}

type scanPolicyKey struct{}

// WithScanPolicy attaches per-repo scan policy to the context.
func WithScanPolicy(ctx context.Context, policy ScanPolicy) context.Context {
	return context.WithValue(ctx, scanPolicyKey{}, policy)
}

// ScanPolicyFromContext returns scan policy when present.
func ScanPolicyFromContext(ctx context.Context) (ScanPolicy, bool) {
	if ctx == nil {
		return ScanPolicy{}, false
	}
	v, ok := ctx.Value(scanPolicyKey{}).(ScanPolicy)
	return v, ok
}

// SnapshotFromPolicyWithMeta builds a JSON snapshot from scan policy and profile metadata.
func SnapshotFromPolicyWithMeta(p ScanPolicy, scanProfile string, profileModified bool, profileSource string) PolicySnapshot {
	snap := SnapshotFromPolicy(p)
	snap.ScanProfile = scanProfile
	snap.ProfileModified = profileModified
	snap.ProfileSource = profileSource
	return snap
}

// SnapshotFromPolicy builds a JSON snapshot from scan policy.
func SnapshotFromPolicy(p ScanPolicy) PolicySnapshot {
	return PolicySnapshot{
		ScanProfile:     p.ScanProfile,
		ProfileModified: p.ProfileModified,
		ProfileSource:   p.ProfileSource,
		WorkspaceMode:     p.WorkspaceMode,
		AnalysisDepth:     p.AnalysisDepth,
		AIPolicy:          p.AIPolicy,
		EnabledScanners:   enabledScannersList(p),
		PolicyLevel:       p.PolicyLevel,
		IssuePolicy:       p.IssuePolicy,
		SeverityGate:      p.SeverityGate,
		ConfidenceGate:    p.ConfidenceGate,
		RemediationPolicy: p.RemediationPolicy,
		EnableHealthChecks:          p.EnableHealthChecks,
		EnableTechDebtChecks:        p.EnableTechDebtChecks,
		EnableReliabilityChecks:     p.EnableReliabilityChecks,
		EnableMaintainabilityChecks: p.EnableMaintainabilityChecks,
		EnableTestGapChecks:         p.EnableTestGapChecks,
		EnablePerformanceChecks:     p.EnablePerformanceChecks,
		EnableAIRiskChecks:          p.EnableAIRiskChecks,
		HealthMaxFindings:           p.HealthMaxFindings,
		HealthLargeFileLines:        p.HealthLargeFileLines,
		HealthLargeFunctionLines:    p.HealthLargeFunctionLines,
		HealthMaxNestingDepth:       p.HealthMaxNestingDepth,
		HealthMaxFunctionParams:     p.HealthMaxFunctionParams,
		EnableCodeGraph:             p.EnableCodeGraph,
		GraphMaxNodes:               p.GraphMaxNodes,
		GraphMaxEdges:               p.GraphMaxEdges,
		GraphTimeoutSeconds:         p.GraphTimeoutSeconds,
		GraphIncludeFunctions:       p.GraphIncludeFunctions,
		GraphIncludeFindings:        p.GraphIncludeFindings,
		EnableGovulncheck:           p.EnableGovulncheck,
		EnableGosec:                 p.EnableGosec,
		EnableStaticcheck:           p.EnableStaticcheck,
		GovulncheckTimeoutSeconds:   p.GovulncheckTimeoutSeconds,
		GosecTimeoutSeconds:         p.GosecTimeoutSeconds,
		StaticcheckTimeoutSeconds:   p.StaticcheckTimeoutSeconds,
		GoScannerMaxFindings:        p.GoScannerMaxFindings,
		EnableHadolint:              p.EnableHadolint,
		EnableCheckov:               p.EnableCheckov,
		HadolintTimeoutSeconds:      p.HadolintTimeoutSeconds,
		CheckovTimeoutSeconds:       p.CheckovTimeoutSeconds,
		IACScannerMaxFindings:       p.IACScannerMaxFindings,
	}
}

func enabledScannersList(p ScanPolicy) []string {
	var out []string
	if p.EnableTrivy {
		out = append(out, "trivy")
	}
	if p.EnableGrype {
		out = append(out, "grype")
	}
	if p.EnableGitleaks {
		out = append(out, "gitleaks")
	}
	if p.EnableSemgrep {
		out = append(out, "semgrep")
	}
	if p.EnableGovulncheck {
		out = append(out, "govulncheck")
	}
	if p.EnableGosec {
		out = append(out, "gosec")
	}
	if p.EnableStaticcheck {
		out = append(out, "staticcheck")
	}
	if p.EnableHadolint {
		out = append(out, "hadolint")
	}
	if p.EnableCheckov {
		out = append(out, "checkov")
	}
	if p.EnableLinters {
		out = append(out, "linters")
	}
	return out
}

func LLMEnabledForPolicy(p ScanPolicy, globalAIConfigured bool) bool {
	return llmEnabledForPolicy(p, globalAIConfigured)
}

func llmEnabledForPolicy(p ScanPolicy, globalAIConfigured bool) bool {
	if p.AIPolicy == "disabled" || !globalAIConfigured || !p.EnableLLMAuditors {
		return false
	}
	depth := p.AnalysisDepth
	if depth <= 0 {
		depth = 3
	}
	return depth >= 3
}
