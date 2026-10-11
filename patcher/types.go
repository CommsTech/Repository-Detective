package patcher

import (
	"time"

	"git.commsnet.org/commstech/repository-detective/remediation"
)

const (
	StatusProposed  = "proposed"
	StatusRunning   = "running"
	StatusFailed    = "failed"
	StatusPROpened  = "pr_opened"
	StatusCancelled = "cancelled"

	AttemptIDPrefix = "pa-"
)

// Config holds safe remediation PR settings.
type Config struct {
	Enabled                          bool
	BranchPrefix                     string
	RequireApproval                  bool
	MaxFilesChanged                  int
	MaxDiffLines                     int
	ValidationTimeoutSec             int
	RequireTests                     bool
	UseRunnerVerification            bool
	BlockHighCriticalWithoutOverride bool
	AllowedSeverities                []string
}

// TestResult records one validation command outcome.
type TestResult struct {
	Command string `json:"command"`
	Status  string `json:"status"`
	Detail  string `json:"detail,omitempty"`
}

// PatchAttempt is a remediation PR attempt record.
type PatchAttempt struct {
	ID                string
	PlanID            string
	RepositoryID      int64
	FindingID         int64
	BranchName        string
	BaseRef           string
	CommitSHA         string
	Status            string
	DiffSummary       string
	FilesChanged      []string
	TestsRun          []TestResult
	ValidationSummary string
	PullRequestNumber *int
	PullRequestURL    string
	Error             string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// EligibilityResult explains whether a plan may attempt a PR.
type EligibilityResult struct {
	Eligible       bool
	BlockedReasons []string
	Checklist      map[string]bool
}

// RepoContext is repository metadata for patch attempts.
type RepoContext struct {
	Owner         string
	Name          string
	FullName      string
	CloneURL      string
	DefaultBranch string
	ConnectedRepo bool
}

// AttemptInput bundles plan + repo for execution.
type AttemptInput struct {
	Plan remediation.Plan
	Repo RepoContext
}
