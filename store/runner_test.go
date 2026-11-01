package store_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestRunnerJobLifecycle(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	scanID := "runner-scan-001"
	_, _ = s.CreateScan(ctx, store.Scan{ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerScheduled, Status: store.ScanStatusStarted})

	spec, _ := json.Marshal(map[string]any{"version": 1})
	policy, _ := json.Marshal(analyzers.PolicySnapshot{AnalysisDepth: 2})
	expires := time.Now().UTC().Add(time.Hour)
	job, err := s.CreateRunnerJob(ctx, store.RunnerJob{
		JobID: "rj-test0001", RepositoryID: repo.ID, ScanID: scanID,
		JobType: store.RunnerJobTypeScanFullRepo, Status: store.RunnerJobStatusQueued,
		RunnerMode: "gitea_actions", Ref: "main", JobSpecJSON: spec, PolicySnapshotJSON: policy,
		ExpiresAt: &expires,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	claimed, err := s.ClaimNextRunnerJob(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if claimed.JobID != job.JobID || claimed.Status != store.RunnerJobStatusRunning {
		t.Fatalf("unexpected claimed job: %+v", claimed)
	}

	claimed.Status = store.RunnerJobStatusCompleted
	claimed.ResultSummaryJSON = json.RawMessage(`{"status":"completed"}`)
	if err := s.UpdateRunnerJob(ctx, claimed); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := s.GetRunnerJobByScanID(ctx, scanID)
	if err != nil || got.Status != store.RunnerJobStatusCompleted {
		t.Fatalf("get by scan: %v status=%s", err, got.Status)
	}
}

func TestRunnerNonceReplay(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	ok, err := s.TryRecordRunnerNonce(ctx, "nonce-1")
	if err != nil || !ok {
		t.Fatalf("first nonce: ok=%v err=%v", ok, err)
	}
	ok, err = s.TryRecordRunnerNonce(ctx, "nonce-1")
	if err != nil || ok {
		t.Fatalf("replay should fail: ok=%v err=%v", ok, err)
	}
}

func TestCancelRunnerJob(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "c", FullName: "o/c"})
	_, err := s.CreateRunnerJob(ctx, store.RunnerJob{
		JobID: "rj-cancel01", RepositoryID: repo.ID, JobType: store.RunnerJobTypeScanFullRepo,
		Status: store.RunnerJobStatusQueued, RunnerMode: "gitea_actions",
		JobSpecJSON: json.RawMessage(`{}`), PolicySnapshotJSON: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CancelRunnerJob(ctx, "rj-cancel01"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
}

func TestExpireStaleRunnerJobsMarksRunningExpired(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "stale", FullName: "o/stale"})
	past := time.Now().UTC().Add(-time.Minute)
	started := past.Add(-20 * time.Minute)
	_, err := s.CreateRunnerJob(ctx, store.RunnerJob{
		JobID: "rj-stale-run", RepositoryID: repo.ID, JobType: store.RunnerJobTypeScanFullRepo,
		Status: store.RunnerJobStatusRunning, RunnerMode: "native", Ref: "main",
		JobSpecJSON: json.RawMessage(`{}`), PolicySnapshotJSON: json.RawMessage(`{}`),
		StartedAt: &started, ExpiresAt: &past,
	})
	if err != nil {
		t.Fatal(err)
	}
	n, err := s.ExpireStaleRunnerJobs(ctx, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 expired job, got %d", n)
	}
	got, err := s.GetRunnerJob(ctx, "rj-stale-run")
	if err != nil || got.Status != store.RunnerJobStatusExpired {
		t.Fatalf("status=%s err=%v", got.Status, err)
	}
}
