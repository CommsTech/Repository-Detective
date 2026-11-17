package runner

import (
	"errors"
	"fmt"
)

var (
	ErrSharedSecretRequired = errors.New("runner delegation enabled but runner_shared_secret is not configured")
	ErrInvalidSignature     = errors.New("invalid runner signature")
	ErrExpiredTimestamp     = errors.New("runner request timestamp expired")
	ErrReplayNonce          = errors.New("runner nonce replay detected")
	ErrResultTooLarge       = errors.New("runner result exceeds size limit")
	ErrUnknownJob           = errors.New("unknown runner job")
	ErrJobExpired           = errors.New("runner job expired")
	ErrScanIDMismatch       = errors.New("runner result scan_id mismatch")
	ErrForbiddenAction      = errors.New("runner result contains forbidden action")
)

// JobView is a minimal job reference for result validation.
type JobView struct {
	JobID  string
	ScanID string
}

// ValidateResultBasic checks structural constraints on a job result.
func ValidateResultBasic(job JobView, result JobResult, maxSizeBytes int64) error {
	if result.JobID != job.JobID {
		return fmt.Errorf("%w: job_id mismatch", ErrUnknownJob)
	}
	if result.ScanID != job.ScanID {
		return ErrScanIDMismatch
	}
	if result.ForbiddenAction != "" {
		return fmt.Errorf("%w: %s", ErrForbiddenAction, result.ForbiddenAction)
	}
	size, err := result.EncodedSize()
	if err != nil {
		return err
	}
	if maxSizeBytes > 0 && size > maxSizeBytes {
		return ErrResultTooLarge
	}
	return nil
}

// JobViewFromStore builds a validation view from persisted job IDs.
func JobViewFromStore(jobID, scanID string) JobView {
	return JobView{JobID: jobID, ScanID: scanID}
}
