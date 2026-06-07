package store

import (
	"encoding/json"
	"time"
)

// LearningEvent is an auditable outcome used for continuous improvement.
type LearningEvent struct {
	ID              int64           `json:"id"`
	RepositoryID    int64           `json:"repository_id"`
	ScanID          string          `json:"scan_id"`
	FindingID       *int64          `json:"finding_id,omitempty"`
	Fingerprint     string          `json:"fingerprint"`
	Source          string          `json:"source"`
	RuleID          string          `json:"rule_id"`
	EventType       string          `json:"event_type"`
	EvidenceJSON    json.RawMessage `json:"evidence_json"`
	CreatedAt       time.Time       `json:"created_at"`
	CreatedBy       string          `json:"created_by"`
	ConfidenceDelta float64         `json:"confidence_delta"`
	IdempotencyKey  string          `json:"idempotency_key"`
}

// RepoCalibrationRule is an operator-approved per-repo calibration rule.
type RepoCalibrationRule struct {
	ID               int64      `json:"id"`
	RepositoryID     *int64     `json:"repository_id,omitempty"`
	ProjectGroupID   *int64     `json:"project_group_id,omitempty"`
	Scope            string     `json:"scope"`
	Source           string     `json:"source"`
	RuleID           string     `json:"rule_id"`
	PathPattern      string     `json:"path_pattern"`
	FindingCategory  string     `json:"finding_category"`
	Action           string     `json:"action"`
	Reason           string     `json:"reason"`
	EvidenceCount    int        `json:"evidence_count"`
	FalsePositiveRate float64   `json:"false_positive_rate"`
	TruePositiveRate  float64   `json:"true_positive_rate"`
	DuplicateRate     float64   `json:"duplicate_rate"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	Active           bool       `json:"active"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	RecommendationID *int64     `json:"recommendation_id,omitempty"`
}

// RuleReliabilityStat is per-repo (or global when RepositoryID nil) rule metrics.
type RuleReliabilityStat struct {
	ID                    int64     `json:"id"`
	RepositoryID          *int64    `json:"repository_id,omitempty"`
	ProjectGroupID        *int64    `json:"project_group_id,omitempty"`
	Source                string    `json:"source"`
	RuleID                string    `json:"rule_id"`
	ScansSeen             int       `json:"scans_seen"`
	FindingsSeen          int       `json:"findings_seen"`
	TruePositiveCount     int       `json:"true_positive_count"`
	FalsePositiveCount    int       `json:"false_positive_count"`
	ResolvedVerifiedCount int       `json:"resolved_verified_count"`
	DuplicateCount        int       `json:"duplicate_count"`
	ReappearedCount       int       `json:"reappeared_count"`
	IssueCreatedCount     int       `json:"issue_created_count"`
	IssueClosedCount      int       `json:"issue_closed_count"`
	ScannerFailureCount   int       `json:"scanner_failure_count"`
	LastSeenAt            time.Time `json:"last_seen_at"`
	ReliabilityScore      float64   `json:"reliability_score"`
	ActionabilityScore    float64   `json:"actionability_score"`
}

// ScannerHealthRecord captures one scanner run outcome for learning.
type ScannerHealthRecord struct {
	ID           int64     `json:"id"`
	RepositoryID int64     `json:"repository_id"`
	ScanID       string    `json:"scan_id"`
	Scanner      string    `json:"scanner"`
	Status       string    `json:"status"`
	Version      string    `json:"version"`
	DurationMs   int       `json:"duration_ms"`
	FindingCount int       `json:"finding_count"`
	ErrorClass   string    `json:"error_class"`
	CreatedAt    time.Time `json:"created_at"`
}

// LearningHealthSummary aggregates operator-facing learning metrics.
type LearningHealthSummary struct {
	EventsTotal              int     `json:"events_total"`
	PendingRecommendations   int     `json:"pending_recommendations"`
	ActiveRepoRules          int     `json:"active_repo_rules"`
	ExpiredRepoRules         int     `json:"expired_repo_rules"`
	GroupedFindings          int     `json:"grouped_findings"`
	AvgFalsePositiveRate     float64 `json:"avg_false_positive_rate"`
	ScannerFailureRate       float64 `json:"scanner_failure_rate"`
}
