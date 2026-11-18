package runner

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/sirupsen/logrus"
)

// ResultHandler ingests validated runner results into core persistence flows.
type ResultHandler func(ctx context.Context, job store.RunnerJob, result JobResult, repo store.Repository, effective store.EffectiveSettings) error

// JobsExpiredHandler is called when stale runner jobs are expired during claim.
type JobsExpiredHandler func(ctx context.Context, count int64)

// Receiver validates runner callbacks and completes jobs.
type Receiver struct {
	store         store.QueryStore
	cfg           Config
	logger        *logrus.Logger
	onResult      ResultHandler
	onJobsExpired JobsExpiredHandler
}

// NewReceiver creates a runner result receiver.
func NewReceiver(s store.QueryStore, cfg Config, logger *logrus.Logger, onResult ResultHandler) *Receiver {
	return &Receiver{store: s, cfg: cfg.Normalized(), logger: logger, onResult: onResult}
}

// SetJobsExpiredHandler registers a callback when ExpireStaleRunnerJobs marks jobs expired.
func (r *Receiver) SetJobsExpiredHandler(h JobsExpiredHandler) {
	if r != nil {
		r.onJobsExpired = h
	}
}

// ClaimNextJob atomically claims the oldest queued job for a runner worker.
func (r *Receiver) ClaimNextJob(ctx context.Context) (store.RunnerJob, JobSpec, error) {
	if r.store == nil {
		return store.RunnerJob{}, JobSpec{}, fmt.Errorf("database disabled")
	}
	now := time.Now().UTC()
	if n, err := r.store.ExpireStaleRunnerJobs(ctx, now); err == nil && n > 0 && r.onJobsExpired != nil {
		r.onJobsExpired(ctx, n)
	}
	job, err := r.store.ClaimNextRunnerJob(ctx, now)
	if err != nil {
		return store.RunnerJob{}, JobSpec{}, err
	}
	var spec JobSpec
	if err := json.Unmarshal(job.JobSpecJSON, &spec); err != nil {
		return store.RunnerJob{}, JobSpec{}, fmt.Errorf("invalid job spec: %w", err)
	}
	return job, spec, nil
}

// ClaimNextJobForTypes selects queued native work by the worker's capabilities.
// The store must support server-side filtering; do not fall back to claiming an
// arbitrary job when that operation is unavailable.
func (r *Receiver) ClaimNextJobForTypes(ctx context.Context, jobTypes []string) (store.RunnerJob, JobSpec, error) {
	if r == nil || r.store == nil {
		return store.RunnerJob{}, JobSpec{}, fmt.Errorf("database disabled")
	}
	filtered, ok := r.store.(interface {
		ClaimNextRunnerJobForTypes(context.Context, time.Time, []string) (store.RunnerJob, error)
	})
	if !ok {
		return store.RunnerJob{}, JobSpec{}, fmt.Errorf("runner store does not support capability-filtered claims")
	}
	now := time.Now().UTC()
	if n, err := r.store.ExpireStaleRunnerJobs(ctx, now); err != nil {
		return store.RunnerJob{}, JobSpec{}, err
	} else if n > 0 && r.onJobsExpired != nil {
		r.onJobsExpired(ctx, n)
	}
	job, err := filtered.ClaimNextRunnerJobForTypes(ctx, now, jobTypes)
	if err != nil {
		return store.RunnerJob{}, JobSpec{}, err
	}
	var spec JobSpec
	if err := json.Unmarshal(job.JobSpecJSON, &spec); err != nil {
		return store.RunnerJob{}, JobSpec{}, fmt.Errorf("invalid job spec: %w", err)
	}
	if spec.JobID != job.JobID || spec.JobType != job.JobType {
		return store.RunnerJob{}, JobSpec{}, fmt.Errorf("runner job/spec identity mismatch")
	}
	return job, spec, nil
}

// GetJobSpec returns the spec for a known job.
func (r *Receiver) GetJobSpec(ctx context.Context, jobID string) (store.RunnerJob, JobSpec, error) {
	job, err := r.store.GetRunnerJob(ctx, jobID)
	if err != nil {
		return store.RunnerJob{}, JobSpec{}, err
	}
	if job.ExpiresAt != nil && !job.ExpiresAt.After(time.Now().UTC()) {
		return store.RunnerJob{}, JobSpec{}, ErrJobExpired
	}
	var spec JobSpec
	if err := json.Unmarshal(job.JobSpecJSON, &spec); err != nil {
		return store.RunnerJob{}, JobSpec{}, err
	}
	return job, spec, nil
}

// runnerJobTransition is a compare-and-swap persistence operation. It is
// intentionally mandatory for callbacks: falling back to an unguarded UPDATE
// would allow two workers to ingest the same scan concurrently.
type runnerJobTransition interface {
	TransitionRunnerJobStatus(context.Context, store.RunnerJob, string) (bool, error)
}

// SubmitResult preserves callback evidence and ingests it before declaring a
// runner job complete. It does not automatically replay partial side effects:
// ingestion_failed requires operator reconciliation to avoid duplicate forge
// issues or finding lifecycle events.
func (r *Receiver) SubmitResult(ctx context.Context, jobID string, result JobResult) error {
	if r == nil || r.store == nil {
		return fmt.Errorf("database disabled")
	}
	if r.onResult == nil {
		return fmt.Errorf("runner result ingestion handler not configured")
	}
	transitions, ok := r.store.(runnerJobTransition)
	if !ok {
		return fmt.Errorf("runner store lacks atomic result transitions")
	}
	job, err := r.store.GetRunnerJob(ctx, jobID)
	if err != nil {
		return ErrUnknownJob
	}
	maxBytes := int64(r.cfg.ResultMaxSizeMB) * 1024 * 1024
	if err := ValidateResultBasic(JobView{JobID: job.JobID, ScanID: job.ScanID}, result, maxBytes); err != nil {
		return err
	}
	// The original payload is hashed before any status normalization. A worker
	// retry after a successful but unacknowledged callback is idempotently
	// accepted only if it is byte-equivalent as structured JSON.
	raw, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode runner result: %w", err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(raw))
	if job.Status == store.RunnerJobStatusCompleted || job.Status == store.RunnerJobStatusFailed {
		if savedResultDigest(job.ResultSummaryJSON) == digest {
			return nil
		}
		return fmt.Errorf("job already finalized with a different result")
	}
	if job.Status == store.RunnerJobStatusCancelled {
		return fmt.Errorf("job already finalized")
	}
	if job.ExpiresAt != nil && !job.ExpiresAt.After(time.Now().UTC()) {
		return ErrJobExpired
	}
	if job.Status != store.RunnerJobStatusRunning && job.Status != store.RunnerJobStatusDispatched {
		return fmt.Errorf("job cannot accept results in state %q", job.Status)
	}

	repo, err := r.store.GetRepository(ctx, job.RepositoryID)
	if err != nil {
		return err
	}
	var policy analyzers.PolicySnapshot
	if len(job.PolicySnapshotJSON) > 0 {
		if err := json.Unmarshal(job.PolicySnapshotJSON, &policy); err != nil {
			return fmt.Errorf("decode runner job policy snapshot: %w", err)
		}
	}
	effective := policyToEffective(policy)
	if len(result.Errors) > 0 {
		result.Status = JobStatusFailed
	}
	job.ResultSummaryJSON = resultSummaryWithDigest(result, digest)
	job.Status = store.RunnerJobStatusIngesting
	job.FinishedAt = nil
	claimed, err := transitions.TransitionRunnerJobStatus(ctx, job, store.RunnerJobStatusRunning)
	if err != nil {
		return err
	}
	if !claimed {
		claimed, err = transitions.TransitionRunnerJobStatus(ctx, job, store.RunnerJobStatusDispatched)
		if err != nil {
			return err
		}
	}
	if !claimed {
		return fmt.Errorf("runner job is already being ingested or was finalized")
	}

	if err := r.store.SaveRunnerArtifact(ctx, store.RunnerArtifact{
		JobID: job.JobID, ArtifactType: store.RunnerArtifactResultJSON,
		BodyJSON: raw, SizeBytes: int64(len(raw)), SHA256: digest, CreatedAt: time.Now().UTC(),
	}); err != nil {
		return r.markIngestionFailed(ctx, transitions, job, fmt.Errorf("persist runner result artifact: %w", err))
	}
	if result.Status == JobStatusFailed {
		job.Status = store.RunnerJobStatusFailed
		if len(result.Errors) > 0 {
			job.Error = RedactLogLine(result.Errors[0])
		} else {
			job.Error = "runner reported failure"
		}
	} else {
		job.Status = store.RunnerJobStatusCompleted
		job.Error = ""
	}
	if err := r.onResult(ctx, job, result, repo, effective); err != nil {
		return r.markIngestionFailed(ctx, transitions, job, fmt.Errorf("runner result ingestion: %w", err))
	}
	now := time.Now().UTC()
	job.FinishedAt = &now
	finalized, err := transitions.TransitionRunnerJobStatus(ctx, job, store.RunnerJobStatusIngesting)
	if err != nil {
		return fmt.Errorf("runner result ingested but final state could not be saved: %w", err)
	}
	if !finalized {
		return fmt.Errorf("runner job state changed during result ingestion; completion not recorded")
	}
	return nil
}

func resultSummaryWithDigest(result JobResult, digest string) json.RawMessage {
	var summary map[string]any
	if err := json.Unmarshal(result.SummaryJSON(), &summary); err != nil {
		summary = map[string]any{"status": result.Status}
	}
	summary["callback_sha256"] = digest
	raw, _ := json.Marshal(summary)
	return raw
}

func savedResultDigest(raw json.RawMessage) string {
	var summary struct {
		Digest string `json:"callback_sha256"`
	}
	if json.Unmarshal(raw, &summary) != nil {
		return ""
	}
	return summary.Digest
}

func (r *Receiver) markIngestionFailed(ctx context.Context, transitions runnerJobTransition, job store.RunnerJob, cause error) error {
	job.Status = store.RunnerJobStatusIngestFailed
	job.Error = RedactLogLine(cause.Error())
	end := time.Now().UTC()
	job.FinishedAt = &end
	// The request may already have been canceled; persist the failure with a
	// separate, bounded cleanup context so the job cannot look successful.
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	changed, updateErr := transitions.TransitionRunnerJobStatus(cleanupCtx, job, store.RunnerJobStatusIngesting)
	if updateErr != nil {
		return fmt.Errorf("%w (also failed to persist incomplete ingestion: %v)", cause, updateErr)
	}
	if !changed {
		return fmt.Errorf("%w (runner ingestion state changed before failure could be recorded)", cause)
	}
	return cause
}

// CheckNonce records a nonce and returns false on replay.
func (r *Receiver) CheckNonce(ctx context.Context, nonce string) error {
	ok, err := r.store.TryRecordRunnerNonce(ctx, nonce)
	if err != nil {
		return err
	}
	if !ok {
		return ErrReplayNonce
	}
	return nil
}

func policyToEffective(p analyzers.PolicySnapshot) store.EffectiveSettings {
	return store.EffectiveSettings{
		PolicyLevel:                 p.PolicyLevel,
		WorkspaceMode:               p.WorkspaceMode,
		AnalysisDepth:               p.AnalysisDepth,
		IssuePolicy:                 p.IssuePolicy,
		SeverityGate:                p.SeverityGate,
		ConfidenceGate:              p.ConfidenceGate,
		EnableHealthChecks:          p.EnableHealthChecks,
		EnableTechDebtChecks:        p.EnableTechDebtChecks,
		EnableReliabilityChecks:     p.EnableReliabilityChecks,
		EnableMaintainabilityChecks: p.EnableMaintainabilityChecks,
		EnableTestGapChecks:         p.EnableTestGapChecks,
		EnablePerformanceChecks:     p.EnablePerformanceChecks,
		EnableAIRiskChecks:          p.EnableAIRiskChecks,
		EnableCodeGraph:             p.EnableCodeGraph,
	}
}
