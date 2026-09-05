package api

import (
	"encoding/json"
	"time"

	"git.commsnet.org/commstech/repository-detective/notify"
	"git.commsnet.org/commstech/repository-detective/store"
)

const settingsNotice = "Per-repo settings are enforced on scans (Phase 8). Runner policy is enforced for scheduled and manual full scans when runner delegation is enabled (Phase 12)."

type repositoryResponse struct {
	ID            int64     `json:"id"`
	ForgeType     string    `json:"forge_type"`
	Owner         string    `json:"owner"`
	Name          string    `json:"name"`
	FullName      string    `json:"full_name"`
	CloneURL      string    `json:"clone_url,omitempty"`
	DefaultBranch string    `json:"default_branch,omitempty"`
	ConnectedRepo bool      `json:"connected_repo"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type repositorySummaryResponse struct {
	repositoryResponse
	LastScanAt         *time.Time `json:"last_scan_at,omitempty"`
	LastScanStatus     string     `json:"last_scan_status,omitempty"`
	OpenFindingsCount  int        `json:"open_findings_count"`
	TotalFindingsCount int        `json:"total_findings_count"`
}

type settingsResponse struct {
	RepositoryID            int64                           `json:"repository_id"`
	ScanProfile             string                          `json:"scan_profile"`
	ProfileModified         bool                            `json:"profile_modified"`
	ProfileSource           string                          `json:"profile_source"`
	EffectiveProfileSummary effectiveProfileSummaryResponse `json:"effective_profile_summary"`
	NotificationGlobal      notificationGlobalResponse      `json:"notification_global"`
	EffectiveNotifications  effectiveNotificationResponse   `json:"effective_notifications"`
	Stored                  repoSettingsFields              `json:"stored"`
	Effective               repoSettingsFields              `json:"effective"`
	Notice                  string                          `json:"notice"`
	UpdatedAt               *time.Time                      `json:"updated_at,omitempty"`
	ScheduleCron            *store.CronDescription          `json:"schedule_cron_info,omitempty"`
}

type effectiveProfileSummaryResponse struct {
	SecurityScanners bool   `json:"security_scanners"`
	GoScanners       bool   `json:"go_scanners"`
	IACScanners      bool   `json:"iac_scanners"`
	HealthChecks     bool   `json:"health_checks"`
	CodeGraph        bool   `json:"code_graph"`
	AI               string `json:"ai"`
	RunnerEligible   bool   `json:"runner_eligible"`
}

type repoSettingsFields struct {
	ScanProfile                 *string  `json:"scan_profile,omitempty"`
	Enabled                     *bool    `json:"enabled,omitempty"`
	PolicyLevel                 *string  `json:"policy_level,omitempty"`
	WorkspaceMode               *string  `json:"workspace_mode,omitempty"`
	AnalysisDepth               *int     `json:"analysis_depth,omitempty"`
	EnableLLMAuditors           *bool    `json:"enable_llm_auditors,omitempty"`
	EnableTrivy                 *bool    `json:"enable_trivy,omitempty"`
	EnableGrype                 *bool    `json:"enable_grype,omitempty"`
	EnableGitleaks              *bool    `json:"enable_gitleaks,omitempty"`
	EnableSemgrep               *bool    `json:"enable_semgrep,omitempty"`
	EnableGovulncheck           *bool    `json:"enable_govulncheck,omitempty"`
	EnableGosec                 *bool    `json:"enable_gosec,omitempty"`
	EnableStaticcheck           *bool    `json:"enable_staticcheck,omitempty"`
	EnableHadolint              *bool    `json:"enable_hadolint,omitempty"`
	EnableCheckov               *bool    `json:"enable_checkov,omitempty"`
	EnableLinters               *bool    `json:"enable_linters,omitempty"`
	SeverityGate                *string  `json:"severity_gate,omitempty"`
	ConfidenceGate              *float64 `json:"confidence_gate,omitempty"`
	IssuePolicy                 *string  `json:"issue_policy,omitempty"`
	RemediationPolicy           *string  `json:"remediation_policy,omitempty"`
	RunnerPolicy                *string  `json:"runner_policy,omitempty"`
	ScheduleEnabled             *bool    `json:"schedule_enabled,omitempty"`
	ScheduleCron                *string  `json:"schedule_cron,omitempty"`
	AIPolicy                    *string  `json:"ai_policy,omitempty"`
	EnableHealthChecks          *bool    `json:"enable_health_checks,omitempty"`
	EnableTechDebtChecks        *bool    `json:"enable_tech_debt_checks,omitempty"`
	EnableReliabilityChecks     *bool    `json:"enable_reliability_checks,omitempty"`
	EnableMaintainabilityChecks *bool    `json:"enable_maintainability_checks,omitempty"`
	EnableTestGapChecks         *bool    `json:"enable_test_gap_checks,omitempty"`
	EnablePerformanceChecks     *bool    `json:"enable_performance_checks,omitempty"`
	EnableAIRiskChecks          *bool    `json:"enable_ai_risk_checks,omitempty"`
	HealthMaxFindings           *int     `json:"health_max_findings,omitempty"`
	HealthLargeFileLines        *int     `json:"health_large_file_lines,omitempty"`
	HealthLargeFunctionLines    *int     `json:"health_large_function_lines,omitempty"`
	HealthMaxNestingDepth       *int     `json:"health_max_nesting_depth,omitempty"`
	HealthMaxFunctionParams     *int     `json:"health_max_function_params,omitempty"`
	EnableCodeGraph             *bool    `json:"enable_code_graph,omitempty"`
	GraphMaxNodes               *int     `json:"graph_max_nodes,omitempty"`
	GraphMaxEdges               *int     `json:"graph_max_edges,omitempty"`
	GraphTimeoutSeconds         *int     `json:"graph_timeout_seconds,omitempty"`
	GraphIncludeFunctions       *bool    `json:"graph_include_functions,omitempty"`
	GraphIncludeFindings        *bool    `json:"graph_include_findings,omitempty"`
	GovulncheckTimeoutSeconds   *int     `json:"govulncheck_timeout_seconds,omitempty"`
	GosecTimeoutSeconds         *int     `json:"gosec_timeout_seconds,omitempty"`
	StaticcheckTimeoutSeconds   *int     `json:"staticcheck_timeout_seconds,omitempty"`
	GoScannerMaxFindings        *int     `json:"go_scanner_max_findings,omitempty"`
	HadolintTimeoutSeconds      *int     `json:"hadolint_timeout_seconds,omitempty"`
	CheckovTimeoutSeconds       *int     `json:"checkov_timeout_seconds,omitempty"`
	IACScannerMaxFindings       *int     `json:"iac_scanner_max_findings,omitempty"`
	NotificationsEnabled        *bool    `json:"notifications_enabled,omitempty"`
	NotificationMinSeverity     *string  `json:"notification_min_severity,omitempty"`
	NotificationEvents          *string  `json:"notification_events,omitempty"`
	NotificationCooldownSeconds *int     `json:"notification_cooldown_seconds,omitempty"`
}

type notificationGlobalResponse struct {
	Enabled         bool     `json:"enabled"`
	MinSeverity     string   `json:"min_severity"`
	CooldownSeconds int      `json:"cooldown_seconds"`
	ChannelsEnabled []string `json:"channels_enabled,omitempty"`
	TelegramEnabled bool     `json:"telegram_enabled"`
	SlackEnabled    bool     `json:"slack_enabled"`
	DiscordEnabled  bool     `json:"discord_enabled"`
	WebhookEnabled  bool     `json:"webhook_enabled"`
}

type effectiveNotificationResponse struct {
	Enabled         bool     `json:"enabled"`
	MinSeverity     string   `json:"min_severity"`
	CooldownSeconds int      `json:"cooldown_seconds"`
	Events          []string `json:"events,omitempty"`
}

type scanResponse struct {
	ID                string          `json:"id"`
	RepositoryID      int64           `json:"repository_id"`
	RepoFullName      string          `json:"repo_full_name,omitempty"`
	TriggerType       string          `json:"trigger_type"`
	Ref               string          `json:"ref"`
	CommitSHA         string          `json:"commit_sha,omitempty"`
	PRNumber          int             `json:"pr_number,omitempty"`
	WorkspaceModeUsed string          `json:"workspace_mode_used,omitempty"`
	CommitPinned      bool            `json:"commit_pinned"`
	Status            string          `json:"status"`
	StartedAt         time.Time       `json:"started_at"`
	FinishedAt        *time.Time      `json:"finished_at,omitempty"`
	Summary           json.RawMessage `json:"summary,omitempty"`
	Error             string          `json:"error,omitempty"`
}

type scannerResultResponse struct {
	ScannerName   string `json:"scanner_name"`
	Status        string `json:"status"`
	FindingsCount int    `json:"findings_count"`
	DurationMS    int64  `json:"duration_ms,omitempty"`
	Detail        string `json:"detail,omitempty"`
	Error         string `json:"error,omitempty"`
}

type findingListResponse struct {
	ID                  int64     `json:"id"`
	RepositoryID        int64     `json:"repository_id"`
	RepoFullName        string    `json:"repo_full_name"`
	Fingerprint         string    `json:"fingerprint"`
	Category            string    `json:"category"`
	Severity            string    `json:"severity"`
	Confidence          float64   `json:"confidence"`
	Source              string    `json:"source"`
	RuleID              string    `json:"rule_id,omitempty"`
	Title               string    `json:"title"`
	Status              string    `json:"status"`
	FirstSeenAt         time.Time `json:"first_seen_at"`
	LastSeenAt          time.Time `json:"last_seen_at"`
	ExternalIssueNumber int       `json:"external_issue_number,omitempty"`
	ExternalIssueURL    string    `json:"external_issue_url,omitempty"`
	Suppressed          bool      `json:"suppressed"`
	SuppressionReason   string    `json:"suppression_reason,omitempty"`
}

type findingDetailResponse struct {
	findingListResponse
	FilePath        string                    `json:"file_path,omitempty"`
	Line            int                       `json:"line,omitempty"`
	PackageName     string                    `json:"package_name,omitempty"`
	FirstSeenScanID string                    `json:"first_seen_scan_id,omitempty"`
	LastSeenScanID  string                    `json:"last_seen_scan_id,omitempty"`
	Instances       []findingInstanceResponse `json:"instances"`
	ExternalIssues  []externalIssueResponse   `json:"external_issues"`
	LifecycleEvents []lifecycleEventResponse  `json:"lifecycle_events"`
}

type findingInstanceResponse struct {
	ID               int64           `json:"id"`
	ScanID           string          `json:"scan_id"`
	EvidenceRedacted string          `json:"evidence_redacted,omitempty"`
	Location         json.RawMessage `json:"location,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

type externalIssueResponse struct {
	ForgeType   string `json:"forge_type"`
	IssueNumber int    `json:"issue_number"`
	IssueURL    string `json:"issue_url"`
	State       string `json:"state"`
}

type lifecycleEventResponse struct {
	ID        int64           `json:"id"`
	FindingID *int64          `json:"finding_id,omitempty"`
	ScanID    string          `json:"scan_id,omitempty"`
	EventType string          `json:"event_type"`
	Message   string          `json:"message,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type dashboardSummaryResponse struct {
	TotalRepositories          int                      `json:"total_repositories"`
	FailedScansCount           int                      `json:"failed_scans_count"`
	ActionableFailedScansCount int                      `json:"actionable_failed_scans_count"`
	StaleReapedScansCount      int                      `json:"stale_reaped_scans_count"`
	UnhealthyReposCount        int                      `json:"unhealthy_repos_count"`
	ScannerFailuresCount       int                      `json:"scanner_failures_count"`
	ScannerParseFailedCount    int                      `json:"scanner_parse_failed_count"`
	ScannerToolsMissingCount   int                      `json:"scanner_tools_missing_count"`
	OpenFindingsCount          int                      `json:"open_findings_count"`
	SuppressedFindingsCount    int                      `json:"suppressed_findings_count"`
	IssuesDetectedInScans      int                      `json:"issues_detected_in_scans"`
	OpenFindingsBySeverity     map[string]int           `json:"open_findings_by_severity"`
	RecentScans                []scanResponse           `json:"recent_scans"`
	RecentLifecycleEvents      []lifecycleEventResponse `json:"recent_lifecycle_events"`
	ScheduledScansCount        int                      `json:"scheduled_scans_count"`
	LastScheduledScanAt        *time.Time               `json:"last_scheduled_scan_at,omitempty"`
	RecentScheduledScans       []scanResponse           `json:"recent_scheduled_scans"`
	RunnerJobsByStatus         map[string]int           `json:"runner_jobs_by_status,omitempty"`
	RemediationCandidates      int                      `json:"remediation_candidates,omitempty"`
	RemediationHumanReview     int                      `json:"remediation_human_review,omitempty"`
	RemediationApproved        int                      `json:"remediation_approved,omitempty"`
	OpenUniqueFindings         int                      `json:"open_unique_findings,omitempty"`
	NewFindingsLast7Days       int                      `json:"new_findings_last_7_days,omitempty"`
	RegressionsLast7Days       int                      `json:"regressions_last_7_days,omitempty"`
	RawDetectorHits7d          int                      `json:"raw_detector_hits_7d,omitempty"`
	RawInstances7d             int                      `json:"raw_instances_7d,omitempty"`
	UniqueMissingScanners      int                      `json:"unique_missing_scanners,omitempty"`
	RawMissingToolEvents       int                      `json:"raw_missing_tool_events,omitempty"`
}

func toRepositoryResponse(repo store.Repository) repositoryResponse {
	return repositoryResponse{
		ID: repo.ID, ForgeType: repo.ForgeType, Owner: repo.Owner, Name: repo.Name,
		FullName: repo.FullName, CloneURL: repo.CloneURL, DefaultBranch: repo.DefaultBranch,
		ConnectedRepo: repo.ConnectedRepo, CreatedAt: repo.CreatedAt, UpdatedAt: repo.UpdatedAt,
	}
}

func toRepositorySummaryResponse(repo store.RepositorySummary) repositorySummaryResponse {
	return repositorySummaryResponse{
		repositoryResponse: toRepositoryResponse(repo.Repository),
		LastScanAt:         repo.LastScanAt,
		LastScanStatus:     repo.LastScanStatus,
		OpenFindingsCount:  repo.OpenFindingsCount,
		TotalFindingsCount: repo.TotalFindingsCount,
	}
}

func toSettingsResponse(id int64, settings store.RepoSettings, global store.GlobalSettingsSnapshot, notifyGlobal notify.Config) settingsResponse {
	effective, meta := store.ResolveEffectiveSettingsFull(global, settings)
	notifyEff := notify.ResolveEffective(notifyGlobal, settings)
	var updatedAt *time.Time
	if !settings.UpdatedAt.IsZero() {
		t := settings.UpdatedAt
		updatedAt = &t
	}
	resp := settingsResponse{
		RepositoryID:    id,
		ScanProfile:     meta.ScanProfile,
		ProfileModified: meta.ProfileModified,
		ProfileSource:   meta.ProfileSource,
		EffectiveProfileSummary: effectiveProfileSummaryResponse{
			SecurityScanners: meta.EffectiveProfileSummary.SecurityScanners,
			GoScanners:       meta.EffectiveProfileSummary.GoScanners,
			IACScanners:      meta.EffectiveProfileSummary.IACScanners,
			HealthChecks:     meta.EffectiveProfileSummary.HealthChecks,
			CodeGraph:        meta.EffectiveProfileSummary.CodeGraph,
			AI:               meta.EffectiveProfileSummary.AI,
			RunnerEligible:   meta.EffectiveProfileSummary.RunnerEligible,
		},
		NotificationGlobal:     toNotificationGlobalResponse(notifyGlobal),
		EffectiveNotifications: toEffectiveNotificationResponse(notifyEff),
		Stored:                 fromRepoSettings(settings),
		Effective:              fromEffectiveSettings(effective),
		Notice:                 settingsNotice,
		UpdatedAt:              updatedAt,
	}
	if effective.ScheduleEnabled && effective.ScheduleCron != "" {
		desc := store.DescribeCron(effective.ScheduleCron, time.Now().UTC())
		resp.ScheduleCron = &desc
	}
	return resp
}

func fromRepoSettings(s store.RepoSettings) repoSettingsFields {
	return repoSettingsFields{
		ScanProfile: s.ScanProfile,
		Enabled:     s.Enabled, PolicyLevel: s.PolicyLevel, WorkspaceMode: s.WorkspaceMode,
		AnalysisDepth: s.AnalysisDepth, EnableLLMAuditors: s.EnableLLMAuditors,
		EnableTrivy: s.EnableTrivy, EnableGrype: s.EnableGrype, EnableGitleaks: s.EnableGitleaks,
		EnableSemgrep: s.EnableSemgrep, EnableGovulncheck: s.EnableGovulncheck,
		EnableGosec: s.EnableGosec, EnableStaticcheck: s.EnableStaticcheck,
		EnableHadolint: s.EnableHadolint, EnableCheckov: s.EnableCheckov,
		EnableLinters: s.EnableLinters,
		SeverityGate:  s.SeverityGate, ConfidenceGate: s.ConfidenceGate,
		IssuePolicy: s.IssuePolicy, RemediationPolicy: s.RemediationPolicy,
		RunnerPolicy: s.RunnerPolicy, ScheduleEnabled: s.ScheduleEnabled,
		ScheduleCron: s.ScheduleCron, AIPolicy: s.AIPolicy,
		EnableHealthChecks: s.EnableHealthChecks, EnableTechDebtChecks: s.EnableTechDebtChecks,
		EnableReliabilityChecks: s.EnableReliabilityChecks, EnableMaintainabilityChecks: s.EnableMaintainabilityChecks,
		EnableTestGapChecks: s.EnableTestGapChecks, EnablePerformanceChecks: s.EnablePerformanceChecks,
		EnableAIRiskChecks: s.EnableAIRiskChecks,
		HealthMaxFindings:  s.HealthMaxFindings, HealthLargeFileLines: s.HealthLargeFileLines,
		HealthLargeFunctionLines: s.HealthLargeFunctionLines, HealthMaxNestingDepth: s.HealthMaxNestingDepth,
		HealthMaxFunctionParams: s.HealthMaxFunctionParams,
		EnableCodeGraph:         s.EnableCodeGraph, GraphMaxNodes: s.GraphMaxNodes, GraphMaxEdges: s.GraphMaxEdges,
		GraphTimeoutSeconds: s.GraphTimeoutSeconds, GraphIncludeFunctions: s.GraphIncludeFunctions,
		GraphIncludeFindings:        s.GraphIncludeFindings,
		GovulncheckTimeoutSeconds:   s.GovulncheckTimeoutSeconds,
		GosecTimeoutSeconds:         s.GosecTimeoutSeconds,
		StaticcheckTimeoutSeconds:   s.StaticcheckTimeoutSeconds,
		GoScannerMaxFindings:        s.GoScannerMaxFindings,
		HadolintTimeoutSeconds:      s.HadolintTimeoutSeconds,
		CheckovTimeoutSeconds:       s.CheckovTimeoutSeconds,
		IACScannerMaxFindings:       s.IACScannerMaxFindings,
		NotificationsEnabled:        s.NotificationsEnabled,
		NotificationMinSeverity:     s.NotificationMinSeverity,
		NotificationEvents:          s.NotificationEvents,
		NotificationCooldownSeconds: s.NotificationCooldownSeconds,
	}
}

func toNotificationGlobalResponse(cfg notify.Config) notificationGlobalResponse {
	channels := notify.ChannelsConfigured(cfg)
	return notificationGlobalResponse{
		Enabled:         cfg.Enabled,
		MinSeverity:     cfg.MinSeverity,
		CooldownSeconds: cfg.CooldownSeconds,
		ChannelsEnabled: channels,
		TelegramEnabled: cfg.TelegramEnabled && cfg.TelegramBotToken != "" && cfg.TelegramChatID != "",
		SlackEnabled:    cfg.SlackEnabled && cfg.SlackWebhookURL != "",
		DiscordEnabled:  cfg.DiscordEnabled && cfg.DiscordWebhookURL != "",
		WebhookEnabled:  cfg.WebhookEnabled && cfg.WebhookURL != "",
	}
}

func toEffectiveNotificationResponse(eff notify.EffectiveSettings) effectiveNotificationResponse {
	return effectiveNotificationResponse{
		Enabled:         eff.Enabled,
		MinSeverity:     eff.MinSeverity,
		CooldownSeconds: eff.CooldownSeconds,
		Events:          notify.EventList(eff.Events),
	}
}

func fromEffectiveSettings(e store.EffectiveSettings) repoSettingsFields {
	enabled := e.Enabled
	policy := e.PolicyLevel
	workspace := e.WorkspaceMode
	depth := e.AnalysisDepth
	llm := e.EnableLLMAuditors
	trivy := e.EnableTrivy
	grype := e.EnableGrype
	gitleaks := e.EnableGitleaks
	semgrep := e.EnableSemgrep
	govulncheck := e.EnableGovulncheck
	gosec := e.EnableGosec
	staticcheck := e.EnableStaticcheck
	hadolint := e.EnableHadolint
	checkov := e.EnableCheckov
	linters := e.EnableLinters
	severity := e.SeverityGate
	confidence := e.ConfidenceGate
	issue := e.IssuePolicy
	remediation := e.RemediationPolicy
	runner := e.RunnerPolicy
	schedule := e.ScheduleEnabled
	cron := e.ScheduleCron
	ai := e.AIPolicy
	healthEnabled := e.EnableHealthChecks
	techDebt := e.EnableTechDebtChecks
	reliability := e.EnableReliabilityChecks
	maintainability := e.EnableMaintainabilityChecks
	testGap := e.EnableTestGapChecks
	performance := e.EnablePerformanceChecks
	aiRisk := e.EnableAIRiskChecks
	maxFindings := e.HealthMaxFindings
	largeFile := e.HealthLargeFileLines
	largeFunc := e.HealthLargeFunctionLines
	nesting := e.HealthMaxNestingDepth
	maxParams := e.HealthMaxFunctionParams
	codeGraph := e.EnableCodeGraph
	graphNodes := e.GraphMaxNodes
	graphEdges := e.GraphMaxEdges
	graphTimeout := e.GraphTimeoutSeconds
	graphFunctions := e.GraphIncludeFunctions
	graphFindings := e.GraphIncludeFindings
	govulncheckTimeout := e.GovulncheckTimeoutSeconds
	gosecTimeout := e.GosecTimeoutSeconds
	staticcheckTimeout := e.StaticcheckTimeoutSeconds
	goScannerMaxFindings := e.GoScannerMaxFindings
	hadolintTimeout := e.HadolintTimeoutSeconds
	checkovTimeout := e.CheckovTimeoutSeconds
	iacScannerMaxFindings := e.IACScannerMaxFindings
	return repoSettingsFields{
		Enabled: &enabled, PolicyLevel: &policy, WorkspaceMode: &workspace, AnalysisDepth: &depth,
		EnableLLMAuditors: &llm, EnableTrivy: &trivy, EnableGrype: &grype,
		EnableGitleaks: &gitleaks, EnableSemgrep: &semgrep,
		EnableGovulncheck: &govulncheck, EnableGosec: &gosec, EnableStaticcheck: &staticcheck,
		EnableHadolint: &hadolint, EnableCheckov: &checkov,
		EnableLinters: &linters,
		SeverityGate:  &severity, ConfidenceGate: &confidence, IssuePolicy: &issue,
		RemediationPolicy: &remediation, RunnerPolicy: &runner, ScheduleEnabled: &schedule,
		ScheduleCron: &cron, AIPolicy: &ai,
		EnableHealthChecks: &healthEnabled, EnableTechDebtChecks: &techDebt,
		EnableReliabilityChecks: &reliability, EnableMaintainabilityChecks: &maintainability,
		EnableTestGapChecks: &testGap, EnablePerformanceChecks: &performance,
		EnableAIRiskChecks: &aiRisk,
		HealthMaxFindings:  &maxFindings, HealthLargeFileLines: &largeFile,
		HealthLargeFunctionLines: &largeFunc, HealthMaxNestingDepth: &nesting,
		HealthMaxFunctionParams: &maxParams,
		EnableCodeGraph:         &codeGraph, GraphMaxNodes: &graphNodes, GraphMaxEdges: &graphEdges,
		GraphTimeoutSeconds: &graphTimeout, GraphIncludeFunctions: &graphFunctions,
		GraphIncludeFindings:      &graphFindings,
		GovulncheckTimeoutSeconds: &govulncheckTimeout,
		GosecTimeoutSeconds:       &gosecTimeout,
		StaticcheckTimeoutSeconds: &staticcheckTimeout,
		GoScannerMaxFindings:      &goScannerMaxFindings,
		HadolintTimeoutSeconds:    &hadolintTimeout,
		CheckovTimeoutSeconds:     &checkovTimeout,
		IACScannerMaxFindings:     &iacScannerMaxFindings,
	}
}

func toScanResponse(scan store.Scan) scanResponse {
	return scanResponse{
		ID: scan.ID, RepositoryID: scan.RepositoryID, TriggerType: scan.TriggerType,
		Ref: scan.Ref, CommitSHA: scan.CommitSHA, PRNumber: scan.PRNumber,
		WorkspaceModeUsed: scan.WorkspaceModeUsed, CommitPinned: scan.CommitPinned,
		Status: scan.Status, StartedAt: scan.StartedAt, FinishedAt: scan.FinishedAt,
		Summary: scan.SummaryJSON, Error: scan.Error,
	}
}

func toScanWithRepoResponse(scan store.ScanWithRepo) scanResponse {
	resp := toScanResponse(scan.Scan)
	resp.RepoFullName = scan.RepoFullName
	return resp
}

func toScannerResultResponse(r store.ScannerResultRecord) scannerResultResponse {
	return scannerResultResponse{
		ScannerName: r.ScannerName, Status: r.Status, FindingsCount: r.FindingsCount,
		DurationMS: r.DurationMS, Detail: r.Detail, Error: r.Error,
	}
}

func toFindingListResponse(f store.FindingListItem) findingListResponse {
	return findingListResponse{
		ID: f.ID, RepositoryID: f.RepositoryID, RepoFullName: f.RepoFullName,
		Fingerprint: f.Fingerprint, Category: f.Category, Severity: f.Severity,
		Confidence: f.Confidence, Source: f.Source, RuleID: f.RuleID, Title: f.Title,
		Status: f.Status, FirstSeenAt: f.FirstSeenAt, LastSeenAt: f.LastSeenAt,
		ExternalIssueNumber: f.ExternalIssueNumber, ExternalIssueURL: f.ExternalIssueURL,
		Suppressed: f.Suppressed, SuppressionReason: f.SuppressionReason,
	}
}

func toFindingDetailResponse(d store.FindingDetail) findingDetailResponse {
	resp := findingDetailResponse{
		findingListResponse: toFindingListResponse(d.FindingListItem),
		FilePath:            d.FilePath, Line: d.Line, PackageName: d.PackageName,
		FirstSeenScanID: d.FirstSeenScanID, LastSeenScanID: d.LastSeenScanID,
	}
	for _, inst := range d.Instances {
		resp.Instances = append(resp.Instances, findingInstanceResponse{
			ID: inst.ID, ScanID: inst.ScanID, EvidenceRedacted: inst.EvidenceRedacted,
			Location: inst.LocationJSON, CreatedAt: inst.CreatedAt,
		})
	}
	for _, ei := range d.ExternalIssues {
		resp.ExternalIssues = append(resp.ExternalIssues, externalIssueResponse{
			ForgeType: ei.ForgeType, IssueNumber: ei.IssueNumber, IssueURL: ei.IssueURL, State: ei.State,
		})
	}
	for _, ev := range d.LifecycleEvents {
		resp.LifecycleEvents = append(resp.LifecycleEvents, toLifecycleEventResponse(ev))
	}
	return resp
}

func toLifecycleEventResponse(ev store.LifecycleEvent) lifecycleEventResponse {
	return lifecycleEventResponse{
		ID: ev.ID, FindingID: ev.FindingID, ScanID: ev.ScanID, EventType: ev.EventType,
		Message: ev.Message, Metadata: ev.MetadataJSON, CreatedAt: ev.CreatedAt,
	}
}

func toDashboardSummaryResponse(s store.DashboardSummary) dashboardSummaryResponse {
	resp := dashboardSummaryResponse{
		TotalRepositories: s.TotalRepositories, FailedScansCount: s.FailedScansCount,
		ActionableFailedScansCount: s.ActionableFailedScansCount, StaleReapedScansCount: s.StaleReapedScansCount,
		UnhealthyReposCount:  s.UnhealthyReposCount,
		ScannerFailuresCount: s.ScannerFailuresCount, ScannerParseFailedCount: s.ScannerParseFailedCount,
		ScannerToolsMissingCount: s.ScannerToolsMissingCount,
		OpenFindingsCount:        s.OpenFindingsCount, SuppressedFindingsCount: s.SuppressedFindingsCount,
		IssuesDetectedInScans:  s.IssuesDetectedInScans,
		OpenFindingsBySeverity: s.OpenFindingsBySeverity,
		ScheduledScansCount:    s.ScheduledScansCount, LastScheduledScanAt: s.LastScheduledScanAt,
		RunnerJobsByStatus:     s.RunnerJobsByStatus,
		RemediationCandidates:  s.Remediation.Candidates,
		RemediationHumanReview: s.Remediation.HumanReview,
		RemediationApproved:    s.Remediation.ApprovedWaiting,
		OpenUniqueFindings:     s.Backlog.OpenUnique,
		NewFindingsLast7Days:   s.Backlog.NewLast7Days,
		RegressionsLast7Days:   s.Backlog.RegressionsLast7Days,
		RawDetectorHits7d:      s.Backlog.RawDetectorHits7d,
		RawInstances7d:         s.Backlog.RawInstances7d,
		UniqueMissingScanners:  s.Platform.UniqueMissingTools,
		RawMissingToolEvents:   s.Platform.RawMissingEvents,
	}
	for _, scan := range s.RecentScans {
		resp.RecentScans = append(resp.RecentScans, toScanWithRepoResponse(scan))
	}
	for _, scan := range s.RecentScheduledScans {
		resp.RecentScheduledScans = append(resp.RecentScheduledScans, toScanWithRepoResponse(scan))
	}
	for _, ev := range s.RecentLifecycleEvents {
		resp.RecentLifecycleEvents = append(resp.RecentLifecycleEvents, toLifecycleEventResponse(ev))
	}
	return resp
}
