package closure

import "time"

const (
	StatusPendingRescan = "pending_rescan"
	StatusVerified      = "verified"
	StatusBlocked       = "blocked"
	StatusStillPresent  = "still_present"

	EventFixPRMerged             = "fix_pr_merged"
	EventClosureVerified         = "closure_verified"
	EventClosureBlocked          = "closure_blocked"
	EventRemediationStillPresent = "remediation_still_present"
	EventClosureIssueCloseFailed = "closure_issue_close_failed"
)

// Config holds evidence-based closure settings.
type Config struct {
	Enabled               bool
	CloseIssues           bool
	Comment               bool
	RequireScannerSuccess bool
}

// Evidence is closure verification state for a finding.
type Evidence struct {
	ID                 int64
	FindingID          int64
	PatchAttemptID     string
	RepositoryID       int64
	Fingerprint        string
	MergeCommitSHA     string
	VerificationScanID string
	OriginalSource     string
	ScannerStatus      string
	FingerprintPresent bool
	Status             string
	Reason             string
	Blockers           []string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ScanContext carries scan data for verification.
type ScanContext struct {
	ScanID           string
	RepositoryID     int64
	Owner            string
	Repo             string
	FingerprintsSeen map[string]struct{}
	ScannerResults   map[string]string // scanner name -> status
}

// MergeInfo records PR merge detection outcome.
type MergeInfo struct {
	Merged         bool
	MergeCommitSHA string
	MergedAt       time.Time
}
