package store

import "time"

const (
	ClosureStatusPendingRescan = "pending_rescan"
	ClosureStatusVerified      = "verified"
	ClosureStatusBlocked       = "blocked"
	ClosureStatusStillPresent  = "still_present"
)

// ClosureEvidenceRecord tracks evidence for verified finding closure.
type ClosureEvidenceRecord struct {
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
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ClosureSummary counts closure evidence by status.
type ClosureSummary struct {
	PendingRescan int
	Verified      int
	Blocked       int
}
