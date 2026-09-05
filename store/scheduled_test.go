package store_test

import (
	"context"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestListScheduledRepositories(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	repo, _ := s.UpsertRepository(ctx, store.Repository{
		Owner: "o", Name: "scheduled", FullName: "o/scheduled",
		ConnectedRepo: true, DefaultBranch: "main",
	})
	enabled := true
	scheduleOn := true
	cron := "0 2 * * *"
	_ = s.SaveRepoSettings(ctx, store.RepoSettings{
		RepositoryID: repo.ID, Enabled: &enabled,
		ScheduleEnabled: &scheduleOn, ScheduleCron: &cron,
	})

	list, err := s.ListScheduledRepositories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ScheduleCron != cron {
		t.Fatalf("unexpected scheduled repos: %+v", list)
	}

	disabled := false
	_ = s.SaveRepoSettings(ctx, store.RepoSettings{
		RepositoryID: repo.ID, Enabled: &disabled,
		ScheduleEnabled: &scheduleOn, ScheduleCron: &cron,
	})
	list, _ = s.ListScheduledRepositories(ctx)
	if len(list) != 0 {
		t.Fatalf("disabled repo should not be scheduled: %+v", list)
	}
}

func TestHasRunningScanForRepository(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	_, _ = s.CreateScan(ctx, store.Scan{ID: "runscan001", RepositoryID: repo.ID, TriggerType: store.TriggerScheduled, Status: store.ScanStatusStarted})

	running, err := s.HasRunningScanForRepository(ctx, repo.ID)
	if err != nil || !running {
		t.Fatalf("expected running scan: running=%v err=%v", running, err)
	}
}

func TestGetLastScanStartedAt(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "last", FullName: "o/last"})
	older := time.Now().UTC().Add(-2 * time.Hour)
	newer := time.Now().UTC().Add(-30 * time.Minute)
	_, _ = s.CreateScan(ctx, store.Scan{
		ID: "older-scan", RepositoryID: repo.ID, TriggerType: store.TriggerManual,
		Status: store.ScanStatusCompleted, StartedAt: older, FinishedAt: &older,
	})
	_, _ = s.CreateScan(ctx, store.Scan{
		ID: "newer-scan", RepositoryID: repo.ID, TriggerType: store.TriggerManual,
		Status: store.ScanStatusCompleted, StartedAt: newer, FinishedAt: &newer,
	})

	last, err := s.GetLastScanStartedAt(ctx, repo.ID)
	if err != nil {
		t.Fatal(err)
	}
	if last == nil {
		t.Fatal("expected last scan time")
	}
	if last.Sub(newer).Abs() > 2*time.Second {
		t.Fatalf("expected newer scan start, got %v", last)
	}
}

func TestListRecentScheduledScans(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	finished := time.Now().UTC()
	_, _ = s.CreateScan(ctx, store.Scan{ID: "sched001", RepositoryID: repo.ID, TriggerType: store.TriggerScheduled, Status: store.ScanStatusCompleted, FinishedAt: &finished})

	scans, err := s.ListRecentScheduledScans(ctx, 5)
	if err != nil || len(scans) != 1 {
		t.Fatalf("recent scheduled scans: len=%d err=%v", len(scans), err)
	}
}

func TestValidateRepoSettingsSchedule(t *testing.T) {
	on := true
	cron := "0 2 * * *"
	if err := store.ValidateRepoSettings(store.RepoSettings{ScheduleEnabled: &on, ScheduleCron: &cron}); err != nil {
		t.Fatal(err)
	}
	if err := store.ValidateRepoSettings(store.RepoSettings{ScheduleEnabled: &on}); err == nil {
		t.Fatal("expected error when schedule enabled without cron")
	}
	bad := "not-a-cron"
	if err := store.ValidateRepoSettings(store.RepoSettings{ScheduleCron: &bad}); err == nil {
		t.Fatal("expected invalid cron error")
	}
}
