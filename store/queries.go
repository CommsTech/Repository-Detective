package store

import (
	"context"
	"time"
)

// RepositorySummary includes aggregate stats for list views.
type RepositorySummary struct {
	Repository
	LastScanAt         *time.Time
	LastScanStatus     string
	OpenFindingsCount  int
	TotalFindingsCount int
}

// ScanWithRepo attaches repository metadata to a scan.
type ScanWithRepo struct {
	Scan
	RepoFullName string
}

// FindingFilter filters finding list queries.
type FindingFilter struct {
	RepositoryID      int64
	Severity          string
	Category          string
	Status            string
	Source            string
	IncludeSuppressed bool
	OnlySuppressed    bool
	Limit             int
	Offset            int
}

// FindingListItem is a finding row for list views.
type FindingListItem struct {
	Finding
	RepoFullName        string
	ExternalIssueNumber int
	ExternalIssueURL    string
	Suppressed          bool
	SuppressionReason   string
}

// FindingDetail is a full finding with related records.
type FindingDetail struct {
	FindingListItem
	Instances       []FindingInstance
	ExternalIssues  []ExternalIssue
	LifecycleEvents []LifecycleEvent
}

// DashboardSummary powers the operator dashboard.
type DashboardSummary struct {
	TotalRepositories        int
	RecentScans              []ScanWithRepo
	FailedScansCount         int
	ScannerFailuresCount     int
	ScannerToolsMissingCount int
	OpenFindingsCount        int
	SuppressedFindingsCount  int
	IssuesDetectedInScans    int
	OpenFindingsBySeverity   map[string]int
	OpenFindingsByCategory   map[string]int
	RecentLifecycleEvents    []LifecycleEvent
	ScheduledScansCount      int
	LastScheduledScanAt      *time.Time
	RecentScheduledScans     []ScanWithRepo
	RunnerJobsByStatus       map[string]int
	Remediation              RemediationSummary
	Closure                  ClosureSummary
	Lifecycle                LifecycleSummary
	Backlog                  FindingBacklogSummary
	Platform                 ScannerPlatformSummary
	ScanHealth               ScanHealthSummary
	RemediationInsight       RemediationInsight
	platformRollups          map[string]scannerDBRollup
}

// ListOptions bounds list query size.
type ListOptions struct {
	Limit  int
	Offset int
}

// NormalizeListOptions applies defaults and caps.
func NormalizeListOptions(opts ListOptions) ListOptions {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	if opts.Limit > 200 {
		opts.Limit = 200
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}
	return opts
}

// QueryStore extends Store with read APIs for the control plane.
type QueryStore interface {
	Store
	AuthStore

	GetRepository(ctx context.Context, id int64) (Repository, error)

	ListRepositoriesWithSummary(ctx context.Context, opts ListOptions) ([]RepositorySummary, error)
	ListScansByRepository(ctx context.Context, repositoryID int64, opts ListOptions) ([]Scan, error)
	ListScannerResultsByScan(ctx context.Context, scanID string) ([]ScannerResultRecord, error)

	ListFindings(ctx context.Context, filter FindingFilter) ([]FindingListItem, error)
	CountFindings(ctx context.Context, filter FindingFilter) (int, error)
	OpenFindingsBySeverityForRepository(ctx context.Context, repositoryID int64) (map[string]int, error)
	OpenFindingsByCategoryForRepository(ctx context.Context, repositoryID int64) (map[string]int, error)
	OpenFindingsConfidenceBandsForRepository(ctx context.Context, repositoryID int64, confidenceGate float64) (map[string]int, error)
	GetFindingDetail(ctx context.Context, id int64) (FindingDetail, error)
	ListFindingsByIDs(ctx context.Context, ids []int64) (map[int64]Finding, error)
	ListLifecycleEventsByFinding(ctx context.Context, findingID int64) ([]LifecycleEvent, error)

	DashboardSummary(ctx context.Context, recentLimit int) (DashboardSummary, error)
	ListRecentScans(ctx context.Context, opts ListOptions) ([]ScanWithRepo, error)
	CountActiveScans(ctx context.Context) (int, error)
	ListExternalIssuesByRepository(ctx context.Context, repositoryID int64, opts ListOptions) ([]ExternalIssue, error)
	ListExternalIssuesByFinding(ctx context.Context, findingID int64) ([]ExternalIssue, error)

	ListScheduledRepositories(ctx context.Context) ([]ScheduledRepository, error)
	HasRunningScanForRepository(ctx context.Context, repositoryID int64) (bool, error)
	GetLastScheduledScanFinishedAt(ctx context.Context, repositoryID int64) (*time.Time, error)
	ListRecentScheduledScans(ctx context.Context, limit int) ([]ScanWithRepo, error)
	CountScheduledScansSince(ctx context.Context, since time.Time) (int, error)
	ReapStaleScans(ctx context.Context, olderThan time.Duration) (int, error)

	CreateAuditRequest(ctx context.Context, req AuditRequest) (AuditRequest, error)
	UpdateAuditRequest(ctx context.Context, req AuditRequest) error
	GetAuditRequest(ctx context.Context, auditID string) (AuditRequest, error)
	ListAuditRequests(ctx context.Context, opts ListOptions) ([]AuditRequest, error)
	AddAuditFindings(ctx context.Context, findings []AuditFinding) error
	ListAuditFindings(ctx context.Context, auditID string) ([]AuditFinding, error)
	AddDisclosureReport(ctx context.Context, report DisclosureReport) (DisclosureReport, error)
	ListDisclosureReports(ctx context.Context, auditID string) ([]DisclosureReport, error)
	GetDisclosureReport(ctx context.Context, id int64) (DisclosureReport, error)
	MarkDisclosureReportReviewed(ctx context.Context, id int64) error

	ListRunnerJobs(ctx context.Context, opts ListOptions) ([]RunnerJob, error)
	ListRunnerJobsByRepository(ctx context.Context, repositoryID int64, opts ListOptions) ([]RunnerJob, error)
	GetRunnerJob(ctx context.Context, jobID string) (RunnerJob, error)
	GetRunnerJobByScanID(ctx context.Context, scanID string) (RunnerJob, error)
	CountRunnerJobsByStatus(ctx context.Context) (map[string]int, error)

	SaveRemediationPlan(ctx context.Context, plan RemediationPlanRecord) (RemediationPlanRecord, error)
	GetRemediationPlanByPlanID(ctx context.Context, planID string) (RemediationPlanRecord, error)
	GetLatestRemediationPlanByFindingID(ctx context.Context, findingID int64) (RemediationPlanRecord, error)
	UpdateRemediationPlanStatus(ctx context.Context, planID, status string) error
	SupersedeRemediationPlansForFinding(ctx context.Context, findingID int64) error
	RemediationSummary(ctx context.Context) (RemediationSummary, error)

	SavePatchAttempt(ctx context.Context, attempt PatchAttemptRecord) (PatchAttemptRecord, error)
	UpdatePatchAttempt(ctx context.Context, attempt PatchAttemptRecord) error
	GetPatchAttemptByAttemptID(ctx context.Context, attemptID string) (PatchAttemptRecord, error)
	ListPatchAttemptsByPlanID(ctx context.Context, planID string) ([]PatchAttemptRecord, error)

	SaveClosureEvidence(ctx context.Context, rec ClosureEvidenceRecord) (ClosureEvidenceRecord, error)
	UpdateClosureEvidence(ctx context.Context, rec ClosureEvidenceRecord) error
	GetLatestClosureEvidenceByFindingID(ctx context.Context, findingID int64) (ClosureEvidenceRecord, error)
	ListClosureEvidenceByRepositoryAndStatus(ctx context.Context, repositoryID int64, status string) ([]ClosureEvidenceRecord, error)
	ClosureSummary(ctx context.Context) (ClosureSummary, error)
	LifecycleSummary(ctx context.Context) (LifecycleSummary, error)
	UpdateFindingStatus(ctx context.Context, findingID int64, status string) error

	CreateFindingSuppression(ctx context.Context, sup FindingSuppression) (FindingSuppression, error)
	DisableFindingSuppression(ctx context.Context, id int64) (FindingSuppression, error)
	GetFindingSuppression(ctx context.Context, id int64) (FindingSuppression, error)
	ListFindingSuppressions(ctx context.Context, filter SuppressionFilter) ([]FindingSuppression, error)
	ListActiveSuppressionsForRepository(ctx context.Context, repositoryID int64) ([]FindingSuppression, error)
	CountSuppressedFindings(ctx context.Context) (int, error)
	ScanQualityReport(ctx context.Context) (ScanQualityReport, error)
	ListPatchAttemptsByRepositoryAndStatus(ctx context.Context, repositoryID int64, status string) ([]PatchAttemptRecord, error)
	GetPatchAttemptForClosure(ctx context.Context, attemptID string) (PatchAttemptRecord, Finding, error)
	UpdatePatchAttemptMerged(ctx context.Context, attemptID, mergeSHA string, mergedAt time.Time) error

	GetLatestCompletedScanForRepository(ctx context.Context, repositoryID int64) (Scan, error)
	ListFingerprintsInScan(ctx context.Context, scanID string, repositoryID int64) (map[string]bool, error)
	SaveReconciliationRun(ctx context.Context, run ReconciliationRun, items []ReconciliationItemRecord) error
	GetReconciliationRun(ctx context.Context, runID string) (ReconciliationRun, []ReconciliationItemRecord, error)
	RecomputeCalibrationRuleStats(ctx context.Context) (int, error)
	ListCalibrationRuleStats(ctx context.Context, limit int) ([]CalibrationRuleStat, error)
	GenerateCalibrationRecommendations(ctx context.Context, minFindings int) (int, error)
	ListCalibrationRecommendations(ctx context.Context, status string, limit int) ([]CalibrationRecommendation, error)
	UpdateCalibrationRecommendationStatus(ctx context.Context, id int64, status string) error
	CalibrationSummary(ctx context.Context) (map[string]any, error)
}
