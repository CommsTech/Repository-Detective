package store

import (
	"encoding/json"
	"time"
)

const (
	ForgeTypeGitea = "gitea"

	ScanStatusStarted   = "started"
	ScanStatusCompleted = "completed"
	ScanStatusFailed    = "failed"
	ScanStatusCancelled = "cancelled"
	ScanStatusSkipped   = "skipped"

	TriggerPush       = "push"
	TriggerPR         = "pr"
	TriggerManual     = "manual"
	TriggerScheduled  = "scheduled"
	TriggerPreInstall = "pre_install"

	FindingStatusOpen             = "open"
	FindingStatusResolvedVerified = "resolved_verified"
	FindingStatusStillPresent     = "still_present"
	FindingStatusClosureBlocked   = "closure_blocked"
	FindingStatusPendingRescan    = "pending_rescan"
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
