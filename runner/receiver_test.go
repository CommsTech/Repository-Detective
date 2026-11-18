package runner_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/runner"
	"git.commsnet.org/commstech/repository-detective/store"
)

func openReceiverStore(t *testing.T) store.QueryStore {
	t.Helper()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "runner.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("close receiver test store: %v", err)
		}
	})
	return s
}

func seedRunnerJob(t *testing.T, s store.QueryStore, jobID, scanID string, expires time.Time) store.RunnerJob {
	t.Helper()
	ctx := context.Background()
	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	if err != nil {
		t.Fatal(err)
	}
	if scanID != "" {
		_, err = s.CreateScan(ctx, store.Scan{ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerScheduled, Status: store.ScanStatusStarted})
		if err != nil {
			t.Fatal(err)
		}
	}
	spec, _ := json.Marshal(runner.BuildJobSpec(runner.Config{}, jobID, repo, scanID, "main", "", analyzers.PolicySnapshot{}))
	policy, _ := json.Marshal(analyzers.PolicySnapshot{})
	job, err := s.CreateRunnerJob(ctx, store.RunnerJob{
		JobID: jobID, RepositoryID: repo.ID, ScanID: scanID,
		JobType: store.RunnerJobTypeScanFullRepo, Status: store.RunnerJobStatusQueued,
		RunnerMode: runner.ModeGiteaActions, Ref: "main",
		JobSpecJSON: spec, PolicySnapshotJSON: policy, ExpiresAt: &expires,
	})
	if err != nil {
		t.Fatal(err)
	}
	return job
}

func newReceiver(t *testing.T, s store.QueryStore) *runner.Receiver {
	t.Helper()
	cfg := runner.Config{ResultMaxSizeMB: 1}
	return runner.NewReceiver(s, cfg, nil,
		func(context.Context, store.RunnerJob, runner.JobResult, store.Repository, store.EffectiveSettings) error {
			return nil
		})
}

func completedResult(jobID, scanID string) runner.JobResult {
	return runner.JobResult{
		Version: runner.ContractVersion,
		JobID:   jobID, ScanID: scanID, Status: runner.JobStatusCompleted,
		StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(),
	}
}

// Results are accepted only for work that a runner actually claimed.
func activateRunnerJob(t *testing.T, s store.QueryStore, jobID string) {
	t.Helper()
	job, err := s.GetRunnerJob(context.Background(), jobID)
	if err != nil {
		t.Fatal(err)
	}
	job.Status = store.RunnerJobStatusRunning
	if err := s.UpdateRunnerJob(context.Background(), job); err != nil {
		t.Fatal(err)
	}
}

func TestReceiverSubmitResultUnknownJob(t *testing.T) {
	s := openReceiverStore(t)
	r := newReceiver(t, s)
	err := r.SubmitResult(context.Background(), "missing-job", completedResult("missing-job", "scan-x"))
	if err == nil {
		t.Fatal("expected unknown job error")
	}
	if !errors.Is(err, runner.ErrUnknownJob) {
		t.Fatalf("expected ErrUnknownJob, got %v", err)
	}
}

func TestReceiverSubmitResultExpiredJob(t *testing.T) {
	s := openReceiverStore(t)
	jobID := "rj-expired1"
	scanID := "scan-expired1"
	seedRunnerJob(t, s, jobID, scanID, time.Now().UTC().Add(-time.Minute))
	r := newReceiver(t, s)
	err := r.SubmitResult(context.Background(), jobID, completedResult(jobID, scanID))
	if err == nil {
		t.Fatal("expected expired job error")
	}
	if !errors.Is(err, runner.ErrJobExpired) {
		t.Fatalf("expected ErrJobExpired, got %v", err)
	}
}

func TestReceiverSubmitResultCancelledJob(t *testing.T) {
	s := openReceiverStore(t)
	jobID := "rj-cancel01"
	scanID := "scan-cancel01"
	expires := time.Now().UTC().Add(time.Hour)
	seedRunnerJob(t, s, jobID, scanID, expires)
	if err := s.CancelRunnerJob(context.Background(), jobID); err != nil {
		t.Fatal(err)
	}
	r := newReceiver(t, s)
	err := r.SubmitResult(context.Background(), jobID, completedResult(jobID, scanID))
	if err == nil {
		t.Fatal("expected cancelled job rejection")
	}
	if err.Error() != "job already finalized" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReceiverSubmitResultScanIDMismatch(t *testing.T) {
	s := openReceiverStore(t)
	jobID := "rj-mismatch"
	scanID := "scan-real"
	expires := time.Now().UTC().Add(time.Hour)
	seedRunnerJob(t, s, jobID, scanID, expires)
	r := newReceiver(t, s)
	err := r.SubmitResult(context.Background(), jobID, completedResult(jobID, "scan-other"))
	if err == nil {
		t.Fatal("expected scan id mismatch")
	}
	if !errors.Is(err, runner.ErrScanIDMismatch) {
		t.Fatalf("expected ErrScanIDMismatch, got %v", err)
	}
}

func TestReceiverSubmitResultForbiddenAction(t *testing.T) {
	s := openReceiverStore(t)
	jobID := "rj-forbidden"
	scanID := "scan-forbidden"
	expires := time.Now().UTC().Add(time.Hour)
	seedRunnerJob(t, s, jobID, scanID, expires)
	r := newReceiver(t, s)
	result := completedResult(jobID, scanID)
	result.ForbiddenAction = "issue_create"
	err := r.SubmitResult(context.Background(), jobID, result)
	if err == nil {
		t.Fatal("expected forbidden action rejection")
	}
	if !errors.Is(err, runner.ErrForbiddenAction) {
		t.Fatalf("expected ErrForbiddenAction, got %v", err)
	}
}

func TestReceiverCheckNonceReplay(t *testing.T) {
	s := openReceiverStore(t)
	r := newReceiver(t, s)
	ctx := context.Background()
	if err := r.CheckNonce(ctx, "nonce-replay-1"); err != nil {
		t.Fatalf("first nonce: %v", err)
	}
	err := r.CheckNonce(ctx, "nonce-replay-1")
	if err == nil {
		t.Fatal("expected replay rejection")
	}
	if !errors.Is(err, runner.ErrReplayNonce) {
		t.Fatalf("expected ErrReplayNonce, got %v", err)
	}
}

func TestReceiverGetJobSpecExpired(t *testing.T) {
	s := openReceiverStore(t)
	jobID := "rj-spec-exp"
	scanID := "scan-spec-exp"
	seedRunnerJob(t, s, jobID, scanID, time.Now().UTC().Add(-time.Minute))
	r := newReceiver(t, s)
	_, _, err := r.GetJobSpec(context.Background(), jobID)
	if err == nil {
		t.Fatal("expected expired job on spec fetch")
	}
	if !errors.Is(err, runner.ErrJobExpired) {
		t.Fatalf("expected ErrJobExpired, got %v", err)
	}
}

func TestReceiverSubmitResultAcceptsValidJob(t *testing.T) {
	s := openReceiverStore(t)
	jobID := "rj-valid01"
	scanID := "scan-valid01"
	expires := time.Now().UTC().Add(time.Hour)
	seedRunnerJob(t, s, jobID, scanID, expires)
	activateRunnerJob(t, s, jobID)
	r := newReceiver(t, s)
	if err := r.SubmitResult(context.Background(), jobID, completedResult(jobID, scanID)); err != nil {
		t.Fatalf("submit valid result: %v", err)
	}
	job, err := s.GetRunnerJob(context.Background(), jobID)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != store.RunnerJobStatusCompleted {
		t.Fatalf("expected completed job, got %s", job.Status)
	}
}

func TestReceiverJobsExpiredCallback(t *testing.T) {
	s := openReceiverStore(t)
	jobID := "rj-stale-exp"
	scanID := "scan-stale-exp"
	seedRunnerJob(t, s, jobID, scanID, time.Now().UTC().Add(-time.Minute))
	r := newReceiver(t, s)
	var expired int64
	r.SetJobsExpiredHandler(func(ctx context.Context, count int64) {
		expired = count
	})
	_, _, err := r.ClaimNextJob(context.Background())
	_ = err // no queued jobs after expiry is expected
	if expired <= 0 {
		t.Fatalf("expected expired callback count > 0, got %d", expired)
	}
}

// Inconsistent worker callbacks must not be reported to operators as successes.
func TestReceiverCompletedStatusWithErrorsFailsClosed(t *testing.T) {
	s := openReceiverStore(t)
	jobID, scanID := "rj-errorstatus", "scan-errorstatus"
	seedRunnerJob(t, s, jobID, scanID, time.Now().UTC().Add(time.Hour))
	activateRunnerJob(t, s, jobID)
	r := newReceiver(t, s)
	result := completedResult(jobID, scanID)
	result.Errors = []string{"scanner timed out"}
	if err := r.SubmitResult(context.Background(), jobID, result); err != nil {
		t.Fatalf("submit result: %v", err)
	}
	job, err := s.GetRunnerJob(context.Background(), jobID)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != store.RunnerJobStatusFailed {
		t.Fatalf("completed-with-errors incorrectly succeeded: %s", job.Status)
	}
	if job.Error != "scanner timed out" {
		t.Fatalf("error not preserved: %q", job.Error)
	}
	var summary map[string]any
	if err := json.Unmarshal(job.ResultSummaryJSON, &summary); err != nil {
		t.Fatal(err)
	}
	if summary["status"] != runner.JobStatusFailed {
		t.Fatalf("summary falsely reports success: %v", summary)
	}
}

func TestReceiverSubmissionFailsClosedForUnclaimedJob(t *testing.T) {
	s := openReceiverStore(t)
	seedRunnerJob(t, s, "rj-unclaimed", "scan-unclaimed", time.Now().UTC().Add(time.Hour))
	r := newReceiver(t, s)
	if err := r.SubmitResult(context.Background(), "rj-unclaimed", completedResult("rj-unclaimed", "scan-unclaimed")); err == nil {
		t.Fatal("unclaimed job accepted a worker result")
	}
	job, err := s.GetRunnerJob(context.Background(), "rj-unclaimed")
	if err != nil || job.Status != store.RunnerJobStatusQueued {
		t.Fatalf("unclaimed job mutated: status=%q err=%v", job.Status, err)
	}
}

func TestReceiverCallbackIdempotencyAndConflict(t *testing.T) {
	s := openReceiverStore(t)
	const jobID, scanID = "rj-idempotent", "scan-idempotent"
	seedRunnerJob(t, s, jobID, scanID, time.Now().UTC().Add(time.Hour))
	activateRunnerJob(t, s, jobID)
	calls := 0
	r := runner.NewReceiver(s, runner.Config{ResultMaxSizeMB: 1}, nil,
		func(ctx context.Context, job store.RunnerJob, res runner.JobResult, repo store.Repository, eff store.EffectiveSettings) error {
			calls++
			return nil
		})
	result := completedResult(jobID, scanID)
	if err := r.SubmitResult(context.Background(), jobID, result); err != nil {
		t.Fatal(err)
	}
	if err := r.SubmitResult(context.Background(), jobID, result); err != nil {
		t.Fatalf("same callback must be idempotent: %v", err)
	}
	if calls != 1 {
		t.Fatalf("duplicate callback was ingested %d times", calls)
	}
	result.FilesAnalyzed = 12
	if err := r.SubmitResult(context.Background(), jobID, result); err == nil {
		t.Fatal("conflicting callback was accepted")
	}
}

func TestReceiverIngestionErrorDoesNotCompleteJob(t *testing.T) {
	s := openReceiverStore(t)
	const jobID, scanID = "rj-ingestion-fail", "scan-ingestion-fail"
	seedRunnerJob(t, s, jobID, scanID, time.Now().UTC().Add(time.Hour))
	activateRunnerJob(t, s, jobID)
	r := runner.NewReceiver(s, runner.Config{ResultMaxSizeMB: 1}, nil,
		func(context.Context, store.RunnerJob, runner.JobResult, store.Repository, store.EffectiveSettings) error {
			return errors.New("simulated scan persistence failure")
		})
	result := completedResult(jobID, scanID)
	if err := r.SubmitResult(context.Background(), jobID, result); err == nil {
		t.Fatal("ingestion failure falsely returned success")
	}
	job, err := s.GetRunnerJob(context.Background(), jobID)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != store.RunnerJobStatusIngestFailed || job.Error == "" {
		t.Fatalf("ingestion failure was hidden: status=%q err=%q", job.Status, job.Error)
	}
	if err := r.SubmitResult(context.Background(), jobID, result); err == nil {
		t.Fatal("partial ingestion replay accepted without reconciliation")
	}
}

func TestReceiverDuplicateDuringActiveIngestionCannotRunTwice(t *testing.T) {
	s := openReceiverStore(t)
	const jobID, scanID = "rj-concurrent", "scan-concurrent"
	seedRunnerJob(t, s, jobID, scanID, time.Now().UTC().Add(time.Hour))
	activateRunnerJob(t, s, jobID)
	entered, release := make(chan struct{}), make(chan struct{})
	r := runner.NewReceiver(s, runner.Config{ResultMaxSizeMB: 1}, nil,
		func(context.Context, store.RunnerJob, runner.JobResult, store.Repository, store.EffectiveSettings) error {
			close(entered)
			<-release
			return nil
		})
	result := completedResult(jobID, scanID)
	done := make(chan error, 1)
	go func() { done <- r.SubmitResult(context.Background(), jobID, result) }()
	<-entered
	if err := r.SubmitResult(context.Background(), jobID, result); err == nil {
		t.Error("second callback accepted while ingestion still in progress")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("original ingestion did not finish: %v", err)
	}
}

func TestReceiverRequiresIngestionHandler(t *testing.T) {
	s := openReceiverStore(t)
	const jobID, scanID = "rj-no-handler", "scan-no-handler"
	seedRunnerJob(t, s, jobID, scanID, time.Now().UTC().Add(time.Hour))
	activateRunnerJob(t, s, jobID)
	r := runner.NewReceiver(s, runner.Config{ResultMaxSizeMB: 1}, nil, nil)
	if err := r.SubmitResult(context.Background(), jobID, completedResult(jobID, scanID)); err == nil {
		t.Fatal("accepted a result without an ingestion handler")
	}
	job, err := s.GetRunnerJob(context.Background(), jobID)
	if err != nil || job.Status != store.RunnerJobStatusRunning {
		t.Fatalf("job falsely finalized without handler: %+v err=%v", job, err)
	}
}
