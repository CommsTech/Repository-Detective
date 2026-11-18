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
	if result.Version != ContractVersion {
		return fmt.Errorf("unsupported runner result contract version %d", result.Version)
	}
	if result.Status != JobStatusCompleted && result.Status != JobStatusFailed {
		return fmt.Errorf("invalid runner result status %q", result.Status)
	}
	if result.FilesAnalyzed < 0 {
		return fmt.Errorf("negative runner files_analyzed")
	}
	for _, sr := range result.ScannerResults {
		if sr.FindingsCount < 0 {
			return fmt.Errorf("negative finding count for scanner %q", sr.Scanner)
		}
	}
	if !result.StartedAt.IsZero() && !result.FinishedAt.IsZero() && result.FinishedAt.Before(result.StartedAt) {
		return fmt.Errorf("runner finished_at precedes started_at")
	}
	return nil
}

// JobViewFromStore builds a validation view from persisted job IDs.
func JobViewFromStore(jobID, scanID string) JobView {
	return JobView{JobID: jobID, ScanID: scanID}
}
