package store

import (
	"encoding/json"
	"time"
)

const (
	ForgeTypeGitea   = "gitea"
	ForgeTypeGitHub  = "github"

	ScanStatusStarted              = "started"
	ScanStatusAnalysisComplete     = "analysis_complete"
	ScanStatusCompleted            = "completed"
	ScanStatusPersistenceIncomplete = "persistence_incomplete"
	ScanStatusFailed               = "failed"
	ScanStatusCancelled = "cancelled"
	ScanStatusSkipped   = "skipped"

	TriggerPush       = "push"
	TriggerPR         = "pr"
	TriggerManual     = "manual"
	TriggerScheduled  = "scheduled"
	TriggerPreInstall = "pre_install"

	FindingStatusOpen             = "open"
	FindingStatusSuppressed       = "suppressed"
	FindingStatusFalsePositive    = "false_positive"
	FindingStatusResolvedVerified = "resolved_verified"
	FindingStatusStillPresent     = "still_present"
	FindingStatusClosureBlocked   = "closure_blocked"
	FindingStatusPendingRescan    = "pending_rescan"

	SuppressionScopeRepo    = "repo"
	SuppressionScopeGlobal  = "global"

	LifecycleEventSuppressed          = "suppressed"
	LifecycleEventUnsuppressed          = "unsuppressed"
	LifecycleEventFalsePositiveMarked   = "false_positive_marked"
	LifecycleEventReconciled                    = "reconciled"
	LifecycleEventExternalIssueMappingBackfilled = "external_issue_mapping_backfilled"
)

// Repository is a tracked forge repository.
type Repository struct {
	ID            int64
	ForgeType     string
	Owner         string
	Name          string
	FullName      string
	CloneURL      string
	DefaultBranch string
	ConnectedRepo bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// RepoSettings holds per-repository control-plane settings.
// Nullable pointer fields mean "inherit global config" when nil.
type RepoSettings struct {
	RepositoryID      int64
	ScanProfile       *string
	Enabled           *bool
	PolicyLevel       *string
	WorkspaceMode     *string
	AnalysisDepth     *int
	EnableLLMAuditors *bool
	EnableTrivy       *bool
	EnableGrype       *bool
	EnableGitleaks    *bool
	EnableSemgrep     *bool
	EnableGovulncheck *bool
	EnableGosec       *bool
	EnableStaticcheck *bool
	EnableHadolint    *bool
	EnableCheckov     *bool
	EnableLinters     *bool
	SeverityGate      *string
	ConfidenceGate    *float64
	IssuePolicy       *string
	RemediationPolicy *string
	RunnerPolicy      *string
	ScheduleEnabled   *bool
	ScheduleCron      *string
	AIPolicy          *string
	EnableHealthChecks          *bool
	EnableTechDebtChecks        *bool
	EnableReliabilityChecks     *bool
	EnableMaintainabilityChecks *bool
	EnableTestGapChecks         *bool
	EnablePerformanceChecks     *bool
	EnableAIRiskChecks          *bool
	HealthMaxFindings           *int
	HealthLargeFileLines        *int
	HealthLargeFunctionLines    *int
	HealthMaxNestingDepth       *int
	HealthMaxFunctionParams     *int
	EnableCodeGraph             *bool
	GraphMaxNodes               *int
	GraphMaxEdges               *int
	GraphTimeoutSeconds         *int
	GraphIncludeFunctions       *bool
	GraphIncludeFindings        *bool
	GovulncheckTimeoutSeconds   *int
	GosecTimeoutSeconds         *int
	StaticcheckTimeoutSeconds   *int
	GoScannerMaxFindings        *int
	HadolintTimeoutSeconds      *int
	CheckovTimeoutSeconds       *int
	IACScannerMaxFindings       *int
	NotificationsEnabled          *bool
	NotificationMinSeverity       *string
	NotificationEvents            *string
	NotificationCooldownSeconds   *int
	UpdatedAt         time.Time
}

// EffectiveSettings is the resolved configuration for a repository scan.
type EffectiveSettings struct {
	ScanProfile       string
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
	RunnerPolicy      string
	ScheduleEnabled   bool
	ScheduleCron      string
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

// GlobalSettingsSnapshot captures global YAML/env defaults for merge.
type GlobalSettingsSnapshot struct {
	ScanProfile       string
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
	RunnerPolicy      string
	ScheduleEnabled   bool
	ScheduleCron      string
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

type Scan struct {
	ID                string
	RepositoryID      int64
	TriggerType       string
	Ref               string
	CommitSHA         string
	PRNumber          int
	WorkspaceModeUsed string
	CommitPinned      bool
	Status            string
	StartedAt         time.Time
	FinishedAt        *time.Time
	SummaryJSON       json.RawMessage
	Error             string
}

// ScanResult is written when a scan finishes.
type ScanResult struct {
	Status            string
	FinishedAt        time.Time
	SummaryJSON       json.RawMessage
	Error             string
	WorkspaceModeUsed string
	CommitPinned      bool
	CommitSHA         string
}

// ScannerResultRecord is one scanner outcome for a scan.
type ScannerResultRecord struct {
	ID            int64
	ScanID        string
	ScannerName   string
	Status        string
	FindingsCount int
	DurationMS    int64
	Detail        string
	Error         string
}

// ScannerFailureEvent is a recent scanner_results failure for operator drill-down.
type ScannerFailureEvent struct {
	ScannerName  string
	Status       string
	Error        string
	Detail       string
	ScanID       string
	RepositoryID int64
	RepoFullName string
	StartedAt    time.Time
	DurationMS   int64
}

// Finding is a deduplicated finding indexed by fingerprint.
type Finding struct {
	ID              int64
	RepositoryID    int64
	Fingerprint     string
	Category        string
	Severity        string
	Confidence      float64
	Source          string
	RuleID          string
	PackageName     string
	FilePath        string
	Line            int
	Title           string
	Status          string
	FirstSeenScanID string
	LastSeenScanID  string
	FirstSeenAt     time.Time
	LastSeenAt      time.Time
	StructuralHash      string
	CanonicalFindingID  *int64
	CalibrationNote     string
}

// FindingInstance is one occurrence of a finding in a scan.
type FindingInstance struct {
	ID               int64
	FindingID        int64
	ScanID           string
	EvidenceRedacted string
	LocationJSON     json.RawMessage
	RawMetadataJSON  json.RawMessage
	CreatedAt        time.Time
}

// ExternalIssue maps a local finding to a forge issue.
type ExternalIssue struct {
	ID          int64
	FindingID   int64
	ForgeType   string
	IssueNumber int
	IssueURL    string
	State       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// LifecycleEvent records finding/scan lifecycle history.
type LifecycleEvent struct {
	ID           int64
	FindingID    *int64
	ScanID       string
	EventType    string
	Message      string
	MetadataJSON json.RawMessage
	CreatedAt    time.Time
}

// FindingSuppression calibrates noisy findings without deleting history.
type FindingSuppression struct {
	ID           int64
	RepositoryID *int64
	Fingerprint  string
	Source       string
	RuleID       string
	Category     string
	Severity     string
	Scope        string
	Reason       string
	CreatedBy    string
	ExpiresAt    *time.Time
	Active       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// SuppressionFilter lists suppressions for repo settings and API.
type SuppressionFilter struct {
	RepositoryID int64
	Scope        string
	ActiveOnly   bool
	Limit        int
	Offset       int
}

// FindingMatchInput is used to test whether a finding or scan issue is suppressed.
type FindingMatchInput struct {
	RepositoryID int64
	Fingerprint  string
	Source       string
	RuleID       string
	Category     string
	Severity     string
}

// ScanQualityReport aggregates dogfood calibration metrics.
type ScanQualityReport struct {
	ReposScanned              int
	TotalFindings             int
	OpenFindings              int
	SuppressedFindings        int
	FalsePositiveFindings     int
	FindingsBySeverity        map[string]int
	FindingsByCategory        map[string]int
	FindingsBySource          map[string]int
	ExternalIssuesOpen        int
	RemediationPlansGenerated int
	PatchAttemptsOpened       int
	PatchAttemptsVerified     int
	ScannerFailures           int
	ReposWithNoFindings       int
	ReposWithCriticalHigh     int
	ActionableFindings        int
	ActionableRatio           float64
	StrictActionableFindings  int
	StrictActionableRatio     float64
	GraphFindingsOpen         int
	ReportOnlyEstimate        int
	EnabledMissingScanners    int
	TopNoisyRules             []RuleCount
	TopSuppressedRules        []RuleCount
	ScannerFailureBreakdown   []ScannerStatusCount
}

// RuleCount pairs a rule with a count for calibration reports.
type RuleCount struct {
	RuleID string `json:"rule_id"`
	Source string `json:"source,omitempty"`
	Count  int    `json:"count"`
}

// ScannerStatusCount groups scanner failures by tool and status.
type ScannerStatusCount struct {
	ScannerName string `json:"scanner_name"`
	Status      string `json:"status"`
	Count       int    `json:"count"`
}

// CalibrationRuleStat aggregates deterministic learning metrics per rule.
type CalibrationRuleStat struct {
	Source                  string    `json:"source"`
	RuleID                  string    `json:"rule_id"`
	Category                string    `json:"category"`
	TotalFindings           int       `json:"total_findings"`
	IssuesCreated           int       `json:"issues_created"`
	Suppressions            int       `json:"suppressions"`
	FalsePositives          int       `json:"false_positives"`
	VerifiedFixes           int       `json:"verified_fixes"`
	StillPresent            int       `json:"still_present"`
	LastSeenAt              time.Time `json:"last_seen_at"`
	ActionableRate          float64   `json:"actionable_rate"`
	FalsePositiveRate       float64   `json:"false_positive_rate"`
	RecommendedDefaultAction string   `json:"recommended_default_action"`
}

// CalibrationRecommendation is a proposed calibration change.
type CalibrationRecommendation struct {
	ID                 int64     `json:"id"`
	Scope              string    `json:"scope"`
	RepositoryID       *int64    `json:"repository_id,omitempty"`
	RecommendationType string    `json:"recommendation_type"`
	Source             string    `json:"source"`
	RuleID             string    `json:"rule_id"`
	Category           string    `json:"category"`
	CurrentAction      string    `json:"current_action"`
	RecommendedAction  string    `json:"recommended_action"`
	Reason             string    `json:"reason"`
	Confidence         float64   `json:"confidence"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// ReconciliationRun records an issue reconciliation pass.
type ReconciliationRun struct {
	RunID        string    `json:"run_id"`
	RepositoryID int64     `json:"repository_id"`
	Preview      bool      `json:"preview"`
	ItemCount    int       `json:"item_count"`
	Applied      int       `json:"applied"`
	CreatedAt    time.Time `json:"created_at"`
}

// ReconciliationItemRecord persists one reconciled issue outcome.
type ReconciliationItemRecord struct {
	RunID          string `json:"run_id"`
	IssueNumber    int    `json:"issue_number"`
	FindingID      int64  `json:"finding_id"`
	Status         string `json:"status"`
	ProposedAction string `json:"proposed_action"`
	Reason         string `json:"reason"`
}
