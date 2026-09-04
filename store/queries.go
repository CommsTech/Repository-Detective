package store

import (
	"context"
	"encoding/json"
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
	TotalRepositories          int
	RecentScans                []ScanWithRepo
	FailedScansCount           int
	ActionableFailedScansCount int
	StaleReapedScansCount      int
	UnhealthyReposCount        int
	ScannerFailuresCount       int
	ScannerParseFailedCount    int
	ScannerToolsMissingCount   int
	OpenFindingsCount          int
	SuppressedFindingsCount    int
	IssuesDetectedInScans      int
	OpenFindingsBySeverity     map[string]int
	OpenFindingsByCategory     map[string]int
	RecentLifecycleEvents      []LifecycleEvent
	ScheduledScansCount        int
	LastScheduledScanAt        *time.Time
	RecentScheduledScans       []ScanWithRepo
	RunnerJobsByStatus         map[string]int
	Remediation                RemediationSummary
	Closure                    ClosureSummary
	Lifecycle                  LifecycleSummary
	Backlog                    FindingBacklogSummary
	Platform                   ScannerPlatformSummary
	ScanHealth                 ScanHealthSummary
	RemediationInsight         RemediationInsight
	platformRollups            map[string]scannerDBRollup
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
	ListRepositoryControlRows(ctx context.Context, opts ListOptions) ([]RepositoryControlRow, error)
	ListScansByRepository(ctx context.Context, repositoryID int64, opts ListOptions) ([]Scan, error)
	ListScannerResultsByScan(ctx context.Context, scanID string) ([]ScannerResultRecord, error)
	ListRecentScannerFailures(ctx context.Context, limit int) ([]ScannerFailureEvent, error)

	ListFindings(ctx context.Context, filter FindingFilter) ([]FindingListItem, error)
	CountFindings(ctx context.Context, filter FindingFilter) (int, error)
	OpenFindingsBySeverityForRepository(ctx context.Context, repositoryID int64) (map[string]int, error)
	OpenFindingsByCategoryForRepository(ctx context.Context, repositoryID int64) (map[string]int, error)
	OpenFindingsByCategoryForRepositories(ctx context.Context, repositoryIDs []int64) (map[int64]map[string]int, error)
	OpenFindingsConfidenceBandsForRepository(ctx context.Context, repositoryID int64, confidenceGate float64) (map[string]int, error)
	GetFindingDetail(ctx context.Context, id int64) (FindingDetail, error)
	ListFindingsByIDs(ctx context.Context, ids []int64) (map[int64]Finding, error)
	ListLifecycleEventsByFinding(ctx context.Context, findingID int64) ([]LifecycleEvent, error)

	DashboardSummary(ctx context.Context, recentLimit int) (DashboardSummary, error)
	ListRecentScans(ctx context.Context, opts ListOptions) ([]ScanWithRepo, error)
	CountCompletedScansByDay(ctx context.Context, since time.Time) (map[string]int, error)
	CountAutoRemediatedFindingsByDay(ctx context.Context, since time.Time) (map[string]int, error)
	CountRemediationPlansByDay(ctx context.Context, since time.Time) (map[string]int, error)
	CountActiveScans(ctx context.Context) (int, error)
	ListExternalIssuesByRepository(ctx context.Context, repositoryID int64, opts ListOptions) ([]ExternalIssue, error)
	ListExternalIssuesByFinding(ctx context.Context, findingID int64) ([]ExternalIssue, error)
	GetExternalIssueByFingerprint(ctx context.Context, repositoryID int64, forgeType, fingerprint string) (ExternalIssue, error)
	GetExternalIssueByIssueNumber(ctx context.Context, repositoryID int64, forgeType string, issueNumber int) (ExternalIssue, error)

	ListScheduledRepositories(ctx context.Context) ([]ScheduledRepository, error)
	HasRunningScanForRepository(ctx context.Context, repositoryID int64) (bool, error)
	GetLastScanStartedAt(ctx context.Context, repositoryID int64) (*time.Time, error)
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
	GetLatestReconcilableScanForRepository(ctx context.Context, repositoryID int64) (Scan, error)
	ReconciliationSummaryForRepository(ctx context.Context, repositoryID int64, issueFilingEnabled bool) (ReconciliationSummary, error)
	ReconciliationSummaryForScan(ctx context.Context, repositoryID int64, scanID string, issueFilingEnabled bool) (ReconciliationSummary, error)
	CountFindingInstancesForScan(ctx context.Context, scanID string) (int, error)
	UpdateScanPipelineState(ctx context.Context, scanID string, status string, fields map[string]any) error
	ListFingerprintsInScan(ctx context.Context, scanID string, repositoryID int64) (map[string]bool, error)
	SaveReconciliationRun(ctx context.Context, run ReconciliationRun, items []ReconciliationItemRecord) error
	GetReconciliationRun(ctx context.Context, runID string) (ReconciliationRun, []ReconciliationItemRecord, error)
	RecomputeCalibrationRuleStats(ctx context.Context) (int, error)
	ListCalibrationRuleStats(ctx context.Context, limit int) ([]CalibrationRuleStat, error)
	GenerateCalibrationRecommendations(ctx context.Context, minFindings int) (int, error)
	ListCalibrationRecommendations(ctx context.Context, status string, limit int) ([]CalibrationRecommendation, error)
	UpdateCalibrationRecommendationStatus(ctx context.Context, id int64, status string) error
	CalibrationSummary(ctx context.Context) (map[string]any, error)

	ListProjectGroups(ctx context.Context) ([]ProjectGroup, error)
	CreateProjectGroup(ctx context.Context, g ProjectGroup) (ProjectGroup, error)

	SaveSBOMArtifact(ctx context.Context, rec SBOMArtifact) error
	GetSBOMArtifactForScan(ctx context.Context, scanID string) (SBOMArtifact, error)
	GetLatestSBOMArtifactForRepository(ctx context.Context, repoID int64) (SBOMArtifact, error)

	RecordLearningEvent(ctx context.Context, ev LearningEvent) (LearningEvent, error)
	ListLearningEvents(ctx context.Context, repositoryID int64, limit int) ([]LearningEvent, error)
	CountLearningEventsByType(ctx context.Context) (map[string]int, error)
	RecordScannerHealth(ctx context.Context, rec ScannerHealthRecord) error
	CreateRepoCalibrationRule(ctx context.Context, rule RepoCalibrationRule) (RepoCalibrationRule, error)
	ListRepoCalibrationRules(ctx context.Context, repositoryID int64, activeOnly bool) ([]RepoCalibrationRule, error)
	ExpireRepoCalibrationRule(ctx context.Context, ruleID int64) error
	BackfillFalsePositiveLearningEvents(ctx context.Context, limit int) (int, error)
	PurgePoisonedScannerFailureLearningEvents(ctx context.Context, scannerNames []string, before time.Time) (int, error)
	GenerateRepoScopedRecommendations(ctx context.Context, repositoryID int64, minFindings int) (int, error)
	ListRepositoryIDsAffectedByRule(ctx context.Context, source, ruleID string, limit int) ([]int64, error)
	LearningHealthSummary(ctx context.Context) (LearningHealthSummary, error)
	AssignStructuralGroup(ctx context.Context, repositoryID int64, structuralHash string, findingID int64) error

	UpsertContainerImageReference(ctx context.Context, ref ContainerImageReference) (ContainerImageReference, error)
	ListContainerImageReferences(ctx context.Context, repoID int64) ([]ContainerImageReference, error)
	CreateContainerImageScan(ctx context.Context, scan ContainerImageScan) (ContainerImageScan, error)
	UpdateContainerImageScan(ctx context.Context, id int64, status, digest string, vulnCount int, coverage, warnings json.RawMessage, finished time.Time) error
	ListContainerImageScans(ctx context.Context, repoID int64, limit int) ([]ContainerImageScan, error)

	CreateAIAdvisoryReview(ctx context.Context, rec AIAdvisoryReview) (AIAdvisoryReview, error)
	UpdateAIAdvisoryReview(ctx context.Context, rec AIAdvisoryReview) error
	GetAIAdvisoryReviewByScanID(ctx context.Context, scanID string) (AIAdvisoryReview, error)
	GetAIAdvisoryReview(ctx context.Context, reviewID string) (AIAdvisoryReview, error)
	ListAIAdvisoryRecommendations(ctx context.Context, reviewID string) ([]AIAdvisoryRecommendation, error)
	ListPendingAIAdvisoryRecommendations(ctx context.Context, limit int) ([]AIAdvisoryRecommendation, error)
	CreateAIAdvisoryRecommendation(ctx context.Context, rec AIAdvisoryRecommendation) (AIAdvisoryRecommendation, error)
	UpdateAIAdvisoryRecommendationStatus(ctx context.Context, id int64, status string) error
	ListFindingsForScan(ctx context.Context, scanID string, limit int) ([]Finding, error)
	ListFindingInstancesByScan(ctx context.Context, scanID string) (map[int64]FindingInstance, error)

	GetPlatformSettings(ctx context.Context) (PlatformSettings, error)
	SavePlatformSettings(ctx context.Context, settings PlatformSettings) error

	PutOperatorEvidence(ctx context.Context, key string, value any) error
	GetOperatorEvidenceJSON(ctx context.Context, key string) (string, error)
	GetWebhookDeliveryEvidence(ctx context.Context) (WebhookDeliveryEvidence, bool, error)
	GetFirstScanEvidence(ctx context.Context) (FirstScanEvidence, bool, error)
	RecordWebhookDelivery(ctx context.Context, ev WebhookDeliveryEvidence) error
	RecordFirstScanProven(ctx context.Context, ev FirstScanEvidence) error
}
