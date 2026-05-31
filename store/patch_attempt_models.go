package store

import (
	"encoding/json"
	"time"
)

const (
	PatchAttemptStatusProposed  = "proposed"
	PatchAttemptStatusRunning   = "running"
	PatchAttemptStatusFailed    = "failed"
	PatchAttemptStatusPROpened  = "pr_opened"
	PatchAttemptStatusPRMerged  = "pr_merged"
	PatchAttemptStatusCancelled = "cancelled"
)

// PatchAttemptRecord is a persisted remediation PR attempt.
type PatchAttemptRecord struct {
	ID                int64
	AttemptID         string
	PlanID            string
	RepositoryID      int64
	FindingID         *int64
	BranchName        string
	BaseRef           string
	BaseCommitSHA     string
	Status            string
	DiffSummary       string
	FilesChangedJSON  json.RawMessage
	TestsRunJSON      json.RawMessage
	ValidationSummary string
	PullRequestNumber *int
	PullRequestURL    string
	Error             string
	MergedAt          *time.Time
	MergeCommitSHA    string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
