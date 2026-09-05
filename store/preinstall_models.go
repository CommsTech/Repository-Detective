package store

import (
	"encoding/json"
	"time"
)

const (
	AuditStatusQueued    = "queued"
	AuditStatusRunning   = "running"
	AuditStatusCompleted = "completed"
	AuditStatusFailed    = "failed"
	AuditStatusCancelled = "cancelled"

	AuditRecommendationSafe         = "safe"
	AuditRecommendationCaution      = "caution"
	AuditRecommendationDoNotInstall = "do_not_install"
	AuditRecommendationUnknown      = "unknown"
	AuditRecommendationAuditFailed  = "audit_failed"

	ReportTypeInstallRiskSummary = "install_risk_summary"
	ReportTypeSecurityDisclosure = "security_disclosure"
	ReportTypeGeneralBug         = "general_bug"
	ReportTypeSupplyChainRisk    = "supply_chain_risk"

	ReportSensitivityPublic          = "public"
	ReportSensitivityPrivateSecurity = "private_security"
	ReportSensitivityInternalReview  = "internal_review"
)

// AuditRequest is a third-party pre-install audit job.
type AuditRequest struct {
	AuditID           string
	RepoURL           string
	NormalizedRepoURL string
	RepoHost          string
	RepoOwner         string
	RepoName          string
	CommitSHA         string
	DefaultBranch     string
	AuditDepth        string
	Status            string
	RiskScore         int
	Recommendation    string
	StartedAt         time.Time
	FinishedAt        *time.Time
	SummaryJSON       json.RawMessage
	Error             string
}

// AuditFinding is a finding from a pre-install audit.
type AuditFinding struct {
	ID               int64
	AuditID          string
	Fingerprint      string
	Category         string
	Severity         string
	Confidence       float64
	Source           string
	RuleID           string
	FilePath         string
	Line             int
	Title            string
	EvidenceRedacted string
	MetadataJSON     json.RawMessage
	CreatedAt        time.Time
}

// DisclosureReport is a copy/paste markdown report draft.
type DisclosureReport struct {
	ID                  int64
	AuditID             string
	FindingID           *int64
	ReportType          string
	Sensitivity         string
	Title               string
	BodyMarkdown        string
	Confidence          float64
	ApprovedByUser      bool
	SubmittedExternally bool
	SubmissionTarget    string
	SubmissionNotes     string
	GeneratedAt         time.Time
}

// AuditScannerResult records scanner outcome for an audit (stored in summary_json).
type AuditScannerResult struct {
	Scanner       string `json:"scanner"`
	Status        string `json:"status"`
	FindingsCount int    `json:"findings_count"`
	Detail        string `json:"detail,omitempty"`
	Error         string `json:"error,omitempty"`
}
