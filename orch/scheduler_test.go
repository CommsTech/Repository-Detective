package orch_test

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/limiter"
	"git.commsnet.org/commstech/repository-detective/orch"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/sirupsen/logrus"
)

func openSchedulerStore(t *testing.T) store.QueryStore {
	t.Helper()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "sched.db")})
	if err != nil {
		t.Fatal(err)
	}
	// Registered first; runs after scheduler stop during t.Cleanup (LIFO).
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("close scheduler test store: %v", err)
		}
	})
	return s
}

// startScheduler registers teardown before store close so the scheduler stops
// (and releases DB handles) before the SQLite file is closed and TempDir removed.
func startScheduler(t *testing.T, sched *orch.Scheduler) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	sched.Start(ctx)
	t.Cleanup(func() {
		cancel()
		sched.Stop()
	})
	return ctx
}

func waitPollCycles(pollInterval time.Duration, cycles int) {
	if cycles < 1 {
		cycles = 1
	}
	time.Sleep(pollInterval*time.Duration(cycles) + pollInterval/2)
}

func seedScheduledRepo(t *testing.T, s store.QueryStore, cron string) store.ScheduledRepository {
	t.Helper()
	ctx := context.Background()
	repo, err := s.UpsertRepository(ctx, store.Repository{
		Owner: "o", Name: "r", FullName: "o/r", ConnectedRepo: true, DefaultBranch: "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	on := true
	if err := s.SaveRepoSettings(ctx, store.RepoSettings{
		RepositoryID: repo.ID, Enabled: &on, ScheduleEnabled: &on, ScheduleCron: &cron,
	}); err != nil {
		t.Fatal(err)
	}
	return store.ScheduledRepository{Repository: repo, ScheduleCron: cron}
}

func TestSchedulerDisabledWhenNotEnabled(t *testing.T) {
	s := openSchedulerStore(t)
	var runs int32
	sched := orch.NewScheduler(s, func(ctx context.Context, repo store.ScheduledRepository) error {
		atomic.AddInt32(&runs, 1)
		return nil
	}, limiter.New(1), orch.Config{Enabled: false, PollInterval: 20 * time.Millisecond}, logrus.New())
	startScheduler(t, sched)
	waitPollCycles(20*time.Millisecond, 2)
	if runs != 0 {
		t.Fatalf("expected no runs when disabled, got %d", runs)
	}
}

func TestSchedulerDueRepoTriggersScan(t *testing.T) {
	s := openSchedulerStore(t)
	ctx := context.Background()
	repo := seedScheduledRepo(t, s, "0 * * * *")
	finished := time.Now().UTC().Add(-2 * time.Hour)
	_, _ = s.CreateScan(ctx, store.Scan{
		ID: "pastsched1", RepositoryID: repo.ID, TriggerType: store.TriggerScheduled,
		Status: store.ScanStatusStarted, StartedAt: finished,
	})
	_ = s.FinishScan(ctx, "pastsched1", store.ScanResult{
		Status: store.ScanStatusCompleted, FinishedAt: finished,
	})

	ran := make(chan struct{}, 1)
	var runs int32
	poll := 30 * time.Millisecond
	sched := orch.NewScheduler(s, func(ctx context.Context, repo store.ScheduledRepository) error {
		atomic.AddInt32(&runs, 1)
		select {
		case ran <- struct{}{}:
		default:
		}
		return nil
	}, limiter.New(2), orch.Config{Enabled: true, PollInterval: poll, MaxConcurrent: 1}, logrus.New())
	startScheduler(t, sched)

	select {
	case <-ran:
	case <-time.After(3 * time.Second):
		t.Fatal("expected scheduled scan to run for due repo")
	}
	if runs == 0 {
		t.Fatal("expected at least one scheduled scan run")
	}
}

func TestSchedulerIgnoresInvalidCron(t *testing.T) {
	s := openSchedulerStore(t)
	seedScheduledRepo(t, s, "not valid cron")

	list, err := s.ListScheduledRepositories(context.Background())
	if err != nil || len(list) != 1 {
		t.Fatalf("repo still listed from DB: %+v err=%v", list, err)
	}

	var runs int32
	poll := 30 * time.Millisecond
	sched := orch.NewScheduler(s, func(ctx context.Context, repo store.ScheduledRepository) error {
		atomic.AddInt32(&runs, 1)
		return nil
	}, limiter.New(1), orch.Config{Enabled: true, PollInterval: poll}, logrus.New())
	startScheduler(t, sched)
	waitPollCycles(poll, 4)
	if runs != 0 {
		t.Fatalf("invalid cron should not trigger scans, got %d", runs)
	}
}

func TestSchedulerDoesNotOverlapRunningScan(t *testing.T) {
	s := openSchedulerStore(t)
	repo := seedScheduledRepo(t, s, "* * * * *")
	ctx := context.Background()
	_, _ = s.CreateScan(ctx, store.Scan{ID: "running001", RepositoryID: repo.ID, TriggerType: store.TriggerScheduled, Status: store.ScanStatusStarted})

	var runs int32
	poll := 30 * time.Millisecond
	sched := orch.NewScheduler(s, func(ctx context.Context, repo store.ScheduledRepository) error {
		atomic.AddInt32(&runs, 1)
		return nil
	}, limiter.New(2), orch.Config{Enabled: true, PollInterval: poll}, logrus.New())
	startScheduler(t, sched)
	waitPollCycles(poll, 4)
	if runs != 0 {
		t.Fatalf("expected skip when scan running, got %d runs", runs)
	}
}

func TestSchedulerNoStartupStampede(t *testing.T) {
	s := openSchedulerStore(t)
	seedScheduledRepo(t, s, "0 0 1 1 *")

	var runs int32
	poll := 30 * time.Millisecond
	sched := orch.NewScheduler(s, func(ctx context.Context, repo store.ScheduledRepository) error {
		atomic.AddInt32(&runs, 1)
		return nil
	}, limiter.New(2), orch.Config{Enabled: true, PollInterval: poll}, logrus.New())
	startScheduler(t, sched)
	waitPollCycles(poll, 4)
	if runs != 0 {
		t.Fatalf("future cron should not stampede on startup, got %d runs", runs)
	}
}
