package store

import (
	"encoding/json"
	"time"
)

const (
	RemediationStatusProposed   = "proposed"
	RemediationStatusApproved   = "approved"
	RemediationStatusRejected   = "rejected"
	RemediationStatusSuperseded = "superseded"
)

// RemediationPlanRecord is a persisted remediation plan.
type RemediationPlanRecord struct {
	ID                     int64
	PlanID                 string
	FindingID              *int64
	RepositoryID           *int64
	AuditID                *string
	Fingerprint            string
	Category               string
	Severity               string
	Source                 string
	RuleID                 string
	Title                  string
	Summary                string
	FixStrategy            string
	AffectedFilesJSON      json.RawMessage
	RequiredTestsJSON      json.RawMessage
	ValidationCommandsJSON json.RawMessage
	RegressionRisk         string
	FixComplexity          string
	SafeForAutoPR          bool
	RequiresHumanReview    bool
	BlockedReasonsJSON     json.RawMessage
	Advisory               bool
	Status                 string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// RemediationSummary counts plans by status for dashboard widgets.
type RemediationSummary struct {
	Candidates      int
	HumanReview     int
	ApprovedWaiting int
}
