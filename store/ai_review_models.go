package store

import (
	"encoding/json"
	"time"
)

// AI advisory review status values.
const (
	AIReviewStatusQueued    = "queued"
	AIReviewStatusRunning   = "running"
	AIReviewStatusCompleted = "completed"
	AIReviewStatusFailed    = "failed"
	AIReviewStatusSkipped   = "skipped"
	AIReviewStatusTimeout   = "timeout"
)

// AIAdvisoryReview is one OpenClaw advisory review run.
type AIAdvisoryReview struct {
	ID                   int64           `json:"id"`
	ReviewID             string          `json:"review_id"`
	ScanID               string          `json:"scan_id"`
	RepositoryID         int64           `json:"repository_id"`
	ScanType             string          `json:"scan_type"`
	Status               string          `json:"status"`
	FindingsSent         int             `json:"findings_sent"`
	RedactionCount       int             `json:"redaction_count"`
	RecommendationsCount int             `json:"recommendations_count"`
	OverallAssessment    string          `json:"overall_assessment"`
	PacketJSON           json.RawMessage `json:"packet_json,omitempty"`
	ResponseJSON         json.RawMessage `json:"response_json,omitempty"`
	ErrorMessage         string          `json:"error_message,omitempty"`
	Model                string          `json:"model"`
	StartedAt            time.Time       `json:"started_at"`
	FinishedAt           *time.Time      `json:"finished_at,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
}

// AIAdvisoryRecommendation is one advisory suggestion from OpenClaw.
type AIAdvisoryRecommendation struct {
	ID                  int64     `json:"id"`
	ReviewID            string    `json:"review_id"`
	FindingFingerprint  string    `json:"finding_fingerprint"`
	Classification      string    `json:"classification"`
	SuggestedAction     string    `json:"suggested_action"`
	SuggestedSeverity   string    `json:"suggested_severity"`
	SuggestedConfidence string    `json:"suggested_confidence"`
	Reason              string    `json:"reason"`
	EvidenceGapsJSON    string    `json:"evidence_gaps_json"`
	OperatorStatus      string    `json:"operator_status"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
