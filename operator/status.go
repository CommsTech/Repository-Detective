package operator

// FeatureFlags holds safe boolean feature toggles for status endpoints.
type FeatureFlags struct {
	DatabaseEnabled           bool   `json:"database_enabled"`
	DatabaseHealthy           bool   `json:"database_healthy"`
	SchedulerEnabled          bool   `json:"scheduler_enabled"`
	RunnerDelegationEnabled   bool   `json:"runner_delegation_enabled"`
	NotificationsEnabled      bool   `json:"notifications_enabled"`
	PreinstallAuditEnabled    bool   `json:"preinstall_audit_enabled"`
	RemediationPlannerEnabled bool   `json:"remediation_planner_enabled"`
	RemediationPREnabled      bool   `json:"remediation_pr_enabled"`
	EvidenceClosureEnabled    bool   `json:"evidence_closure_enabled"`
	PublicURLConfigured       bool   `json:"public_url_configured"`
	UIEnabled                 bool   `json:"ui_enabled"`
	ScanProfile               string `json:"scan_profile"`
}

// Readiness is a safe operator-facing readiness summary.
type Readiness struct {
	ProductName string       `json:"product_name"`
	Tagline     string       `json:"tagline"`
	Version     string       `json:"version"`
	Commit      string       `json:"commit,omitempty"`
	BuildDate   string       `json:"build_date,omitempty"`
	Service     string       `json:"service"`
	Status      string       `json:"status"`
	Timestamp   string       `json:"timestamp"`
	Features    FeatureFlags `json:"features"`
	Tools       []ToolStatus `json:"tools"`
	AIProvider  string       `json:"ai_provider,omitempty"`
	AIModel     string       `json:"ai_model,omitempty"`
	// AIAnalysis is operator-facing: "Disabled" | "Enabled" (never implies security assurance).
	AIAnalysis       string `json:"ai_analysis,omitempty"`
	PrivacyMode      string `json:"privacy_mode,omitempty"`
	AIEndpointClass  string `json:"ai_endpoint_class,omitempty"`
	CodeEgressPolicy string `json:"code_egress_policy,omitempty"`
}
