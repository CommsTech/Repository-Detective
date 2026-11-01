package runner

import (
	"context"
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

// SubmitResult validates and ingests a runner job result.
func (r *Receiver) SubmitResult(ctx context.Context, jobID string, result JobResult) error {
	if r.store == nil {
		return fmt.Errorf("database disabled")
	}
	job, err := r.store.GetRunnerJob(ctx, jobID)
	if err != nil {
		return ErrUnknownJob
	}
	if job.Status == store.RunnerJobStatusCompleted || job.Status == store.RunnerJobStatusCancelled {
		return fmt.Errorf("job already finalized")
	}
	if job.ExpiresAt != nil && !job.ExpiresAt.After(time.Now().UTC()) {
		return ErrJobExpired
	}
	maxBytes := int64(r.cfg.ResultMaxSizeMB) * 1024 * 1024
	if err := ValidateResultBasic(JobView{JobID: job.JobID, ScanID: job.ScanID}, result, maxBytes); err != nil {
		return err
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

	now := time.Now().UTC()
	job.ResultSummaryJSON = result.SummaryJSON()
	job.UpdatedAt = now
	job.FinishedAt = &now

	if result.Status == JobStatusFailed || len(result.Errors) > 0 && result.Status != JobStatusCompleted {
		job.Status = store.RunnerJobStatusFailed
		if len(result.Errors) > 0 {
			job.Error = result.Errors[0]
		}
		if err := r.store.UpdateRunnerJob(ctx, job); err != nil {
			return err
		}
		if r.onResult != nil {
			return r.onResult(ctx, job, result, repo, effective)
		}
		return nil
	}

	job.Status = store.RunnerJobStatusCompleted
	if err := r.store.UpdateRunnerJob(ctx, job); err != nil {
		return err
	}

	raw, _ := json.Marshal(result)
	if int64(len(raw)) <= maxBytes {
		_ = r.store.SaveRunnerArtifact(ctx, store.RunnerArtifact{
			JobID:        job.JobID,
			ArtifactType: store.RunnerArtifactResultJSON,
			BodyJSON:     raw,
			SizeBytes:    int64(len(raw)),
			CreatedAt:    now,
		})
	}

	if r.onResult != nil {
		return r.onResult(ctx, job, result, repo, effective)
	}
	return nil
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
