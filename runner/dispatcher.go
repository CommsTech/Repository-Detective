package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/sirupsen/logrus"
)

// Dispatcher creates runner jobs on the core service.
type Dispatcher struct {
	store  store.QueryStore
	cfg    Config
	logger *logrus.Logger
}

// NewDispatcher creates a job dispatcher.
func NewDispatcher(s store.QueryStore, cfg Config, logger *logrus.Logger) *Dispatcher {
	return &Dispatcher{store: s, cfg: cfg.Normalized(), logger: logger}
}

// CreateScanJob enqueues a runner job for a started scan.
func (d *Dispatcher) CreateScanJob(ctx context.Context, repo store.Repository, scanID, ref, commitSHA string, policy analyzers.PolicySnapshot) (store.RunnerJob, error) {
	if d.store == nil {
		return store.RunnerJob{}, fmt.Errorf("database disabled")
	}
	running, err := d.store.CountRunningRunnerJobs(ctx)
	if err != nil {
		return store.RunnerJob{}, err
	}
	if running >= d.cfg.MaxConcurrentJobs {
		return store.RunnerJob{}, fmt.Errorf("runner job capacity reached")
	}

	jobID, err := newJobID()
	if err != nil {
		return store.RunnerJob{}, err
	}
	spec := BuildJobSpec(d.cfg, jobID, repo, scanID, ref, commitSHA, policy)
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return store.RunnerJob{}, err
	}
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return store.RunnerJob{}, err
	}

	now := time.Now().UTC()
	expires := now.Add(time.Duration(d.cfg.JobTimeoutSeconds) * time.Second)
	job := store.RunnerJob{
		JobID:              jobID,
		RepositoryID:       repo.ID,
		ScanID:             scanID,
		JobType:            store.RunnerJobTypeScanFullRepo,
		Status:             store.RunnerJobStatusQueued,
		RunnerMode:         selectRunnerMode(d.cfg, spec.JobType),
		Ref:                ref,
		CommitSHA:          commitSHA,
		PolicySnapshotJSON: policyJSON,
		JobSpecJSON:        specJSON,
		ResultSummaryJSON:  json.RawMessage(`{}`),
		CreatedAt:          now,
		ExpiresAt:          &expires,
	}
	created, err := d.store.CreateRunnerJob(ctx, job)
	if err != nil {
		return store.RunnerJob{}, err
	}
	if d.logger != nil {
		d.logger.WithFields(logrus.Fields{
			"job_id": jobID, "scan_id": scanID, "repo": repo.FullName,
		}).Info("Runner job queued")
	}
	return created, nil
}

// CreateTypedJob enqueues a delegated job of a specific type (graph, sbom, remediation_verify).
func (d *Dispatcher) CreateTypedJob(ctx context.Context, jobType string, repo store.Repository, scanID, ref, commitSHA string, policy analyzers.PolicySnapshot) (store.RunnerJob, error) {
	if d.store == nil {
		return store.RunnerJob{}, fmt.Errorf("database disabled")
	}
	jobType = strings.TrimSpace(jobType)
	if jobType == "" {
		return store.RunnerJob{}, fmt.Errorf("job type required")
	}
	if !jobTypeAllowed(d.cfg.AllowedJobTypes, jobType) {
		return store.RunnerJob{}, fmt.Errorf("job type %q not allowed by server config", jobType)
	}
	running, err := d.store.CountRunningRunnerJobs(ctx)
	if err != nil {
		return store.RunnerJob{}, err
	}
	if running >= d.cfg.MaxConcurrentJobs {
		return store.RunnerJob{}, fmt.Errorf("runner job capacity reached")
	}

	jobID, err := newJobID()
	if err != nil {
		return store.RunnerJob{}, err
	}
	spec := BuildJobSpecForType(d.cfg, jobID, jobType, repo, scanID, ref, commitSHA, policy, nil)
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return store.RunnerJob{}, err
	}
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return store.RunnerJob{}, err
	}

	now := time.Now().UTC()
	expires := now.Add(time.Duration(d.cfg.JobTimeoutSeconds) * time.Second)
	job := store.RunnerJob{
		JobID:              jobID,
		RepositoryID:       repo.ID,
		ScanID:             scanID,
		JobType:            jobType,
		Status:             store.RunnerJobStatusQueued,
		RunnerMode:         selectRunnerMode(d.cfg, jobType),
		Ref:                ref,
		CommitSHA:          commitSHA,
		PolicySnapshotJSON: policyJSON,
		JobSpecJSON:        specJSON,
		ResultSummaryJSON:  json.RawMessage(`{}`),
		CreatedAt:          now,
		ExpiresAt:          &expires,
	}
	created, err := d.store.CreateRunnerJob(ctx, job)
	if err != nil {
		return store.RunnerJob{}, err
	}
	if d.logger != nil {
		d.logger.WithFields(logrus.Fields{
			"job_id": jobID, "job_type": jobType, "scan_id": scanID, "repo": repo.FullName,
		}).Info("Runner typed job queued")
	}
	return created, nil
}

// CreateContainerImageScanJob enqueues a runner job to scan a container image.
func (d *Dispatcher) CreateContainerImageScanJob(ctx context.Context, repo store.Repository, scanID string, payload ContainerScanPayload, policy analyzers.PolicySnapshot) (store.RunnerJob, error) {
	if d.store == nil {
		return store.RunnerJob{}, fmt.Errorf("database disabled")
	}
	if !jobTypeAllowed(d.cfg.AllowedJobTypes, JobTypeContainerImageScan) {
		return store.RunnerJob{}, fmt.Errorf("job type %q not allowed by server config", JobTypeContainerImageScan)
	}
	jobID, err := newJobID()
	if err != nil {
		return store.RunnerJob{}, err
	}
	spec := BuildJobSpecForType(d.cfg, jobID, JobTypeContainerImageScan, repo, scanID, repo.DefaultBranch, "", policy, ContainerScanTasks)
	spec.ContainerScan = &payload
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return store.RunnerJob{}, err
	}
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return store.RunnerJob{}, err
	}
	now := time.Now().UTC()
	expires := now.Add(time.Duration(d.cfg.JobTimeoutSeconds) * time.Second)
	job := store.RunnerJob{
		JobID: jobID, RepositoryID: repo.ID, ScanID: scanID,
		JobType: store.RunnerJobTypeContainerImageScan,
		Status:  store.RunnerJobStatusQueued, RunnerMode: selectRunnerMode(d.cfg, JobTypeContainerImageScan),
		Ref: repo.DefaultBranch, PolicySnapshotJSON: policyJSON, JobSpecJSON: specJSON,
		ResultSummaryJSON: json.RawMessage(`{}`), CreatedAt: now, ExpiresAt: &expires,
	}
	return d.store.CreateRunnerJob(ctx, job)
}

func jobTypeAllowed(allowed []string, jobType string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, t := range allowed {
		if strings.TrimSpace(t) == strings.TrimSpace(jobType) {
			return true
		}
	}
	return false
}

func selectRunnerMode(cfg Config, jobType string) string {
	return SelectRunnerMode(cfg, jobType)
}

// SelectRunnerMode picks the execution backend for a job type.
func SelectRunnerMode(cfg Config, jobType string) string {
	cfg = cfg.Normalized()
	switch cfg.Mode {
	case ModeNative, ModeGiteaActions, ModeCore:
		return cfg.Mode
	case ModeAuto:
		if jobType == JobTypeRemediationVerify {
			return ModeGiteaActions
		}
		return ModeNative
	default:
		return ModeNative
	}
}

func newJobID() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "rj-" + hex.EncodeToString(buf), nil
}
