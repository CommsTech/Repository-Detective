package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// Worker polls the core service, claims jobs, and executes them one at a time.
type Worker struct {
	cfg    WorkerConfig
	client *Client
	logger *logrus.Logger
}

// NewWorker creates a native runner worker.
func NewWorker(cfg WorkerConfig, logger *logrus.Logger) (*Worker, error) {
	cfg = cfg.Normalized()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if logger == nil {
		logger = logrus.New()
	}
	return &Worker{
		cfg:    cfg,
		client: NewClient(cfg.CoreURL, cfg.SharedSecret),
		logger: logger,
	}, nil
}

// Run blocks until context cancel or SIGTERM, processing one job at a time.
func (w *Worker) Run(ctx context.Context) error {
	if w == nil {
		return fmt.Errorf("worker is nil")
	}
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := os.MkdirAll(w.cfg.WorkspaceRoot, 0o755); err != nil {
		return err
	}

	var active sync.WaitGroup
	active.Add(1)
	go w.heartbeatLoop(ctx, &active)

	w.logger.Infof("native runner worker %s starting (core=%s)", w.cfg.RunnerID, w.cfg.CoreURL)
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("runner worker shutting down")
			active.Wait()
			return ctx.Err()
		default:
		}

		jobID, spec, err := w.client.ClaimNextJob(ctx)
		if err != nil {
			if ctx.Err() != nil {
				active.Wait()
				return ctx.Err()
			}
			time.Sleep(w.cfg.PollInterval)
			continue
		}
		if !JobTypeAllowed(w.cfg.AllowedJobTypes, spec.JobType) {
			w.logger.Warnf("skipping unsupported job type %q", spec.JobType)
			_ = w.client.SubmitResult(ctx, jobID, JobResult{
				Version: ContractVersion, JobID: jobID, ScanID: spec.ScanID,
				Status: JobStatusFailed, StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(),
				Errors: []string{"job type not allowed on this worker"},
			})
			continue
		}

		if err := w.processJob(ctx, jobID, spec); err != nil {
			w.logger.Warnf("job %s failed: %v", jobID, err)
		}
	}
}

func (w *Worker) heartbeatLoop(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	ping := func() {
		if err := w.client.Ping(ctx, w.cfg.RunnerID, w.cfg.Version, w.cfg.AllowedJobTypes); err != nil {
			if ctx.Err() == nil {
				w.logger.Warnf("runner ping failed: %v", err)
			}
		}
	}
	ping()
	ticker := time.NewTicker(w.cfg.HeartbeatEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ping()
		}
	}
}

func (w *Worker) processJob(ctx context.Context, jobID string, spec JobSpec) error {
	jobCtx, cancel := context.WithTimeout(ctx, w.cfg.JobTimeout)
	defer cancel()

	workspace := filepath.Join(w.cfg.WorkspaceRoot, jobID)
	defer os.RemoveAll(workspace)

	ref := spec.Ref
	if ref == "" {
		ref = spec.Repository.DefaultBranch
	}
	jobType := strings.TrimSpace(spec.JobType)
	if jobType == JobTypeScanFullRepoLegacy {
		jobType = JobTypeScanFullRepo
	}
	if jobType != JobTypeContainerImageScan {
		if err := CloneRepository(jobCtx, spec.Repository.CloneURL, ref, workspace, w.cfg.CloneTimeout); err != nil {
			result := JobResult{
				Version: ContractVersion, JobID: jobID, ScanID: spec.ScanID,
				Status: JobStatusFailed, StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(),
				Errors: []string{RedactLogLine(err.Error())},
			}
			return w.client.SubmitResult(jobCtx, jobID, result)
		}
	} else if err := os.MkdirAll(workspace, 0o755); err != nil {
		result := JobResult{
			Version: ContractVersion, JobID: jobID, ScanID: spec.ScanID,
			Status: JobStatusFailed, StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(),
			Errors: []string{RedactLogLine(err.Error())},
		}
		return w.client.SubmitResult(jobCtx, jobID, result)
	}

	scannerCfg, healthCfg, graphCfg := PolicyConfigsFromSpec(spec)
	result, execErr := ExecuteJob(jobCtx, spec, JobExecuteInput{
		WorkspaceDir: workspace,
		ScannerCfg:   scannerCfg,
		HealthCfg:    healthCfg,
		GraphCfg:     graphCfg,
		Logger:       w.logger,
	})
	if execErr != nil && result.Status != JobStatusFailed {
		result.Status = JobStatusFailed
		result.Errors = append(result.Errors, RedactLogLine(execErr.Error()))
	}
	for i, e := range result.Errors {
		result.Errors[i] = RedactLogLine(e)
	}
	if result.FinishedAt.IsZero() {
		result.FinishedAt = time.Now().UTC()
	}

	body, err := json.Marshal(result)
	if err != nil {
		return err
	}
	maxBytes := int64(50 * 1024 * 1024)
	if spec.Limits.MaxResultSizeMB > 0 {
		maxBytes = int64(spec.Limits.MaxResultSizeMB) * 1024 * 1024
	}
	if int64(len(body)) > maxBytes {
		result.Status = JobStatusFailed
		result.Graph = nil
		result.Findings = nil
		result.Errors = []string{fmt.Sprintf("result payload exceeds %d MB limit", spec.Limits.MaxResultSizeMB)}
		body, _ = json.Marshal(result)
	}

	w.logger.Infof("job %s finished status=%s files=%d findings=%d bytes=%d", jobID, result.Status, result.FilesAnalyzed, len(result.Findings), len(body))
	return w.client.SubmitResultBody(jobCtx, jobID, body)
}
