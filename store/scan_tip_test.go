package store

import (
	"context"
	"testing"
	"time"
)

func TestCountCompletedScansForRepository(t *testing.T) {
	s, err := Open(Config{Enabled: true, Path: t.TempDir() + "/count.db"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	repo, err := s.UpsertRepository(ctx, Repository{
		ForgeType: ForgeTypeGitea, Owner: "o", Name: "count-scans", FullName: "o/count-scans", ConnectedRepo: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	n, err := s.CountCompletedScansForRepository(ctx, repo.ID)
	if err != nil || n != 0 {
		t.Fatalf("empty repo count=%d err=%v", n, err)
	}
	_, err = s.CreateScan(ctx, Scan{
		ID: "count-scan-1", RepositoryID: repo.ID, TriggerType: TriggerManual,
		Status: ScanStatusCompleted, CommitSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateScan(ctx, Scan{
		ID: "count-scan-2", RepositoryID: repo.ID, TriggerType: TriggerManual,
		Status: ScanStatusStarted,
	})
	if err != nil {
		t.Fatal(err)
	}
	n, err = s.CountCompletedScansForRepository(ctx, repo.ID)
	if err != nil || n != 1 {
		t.Fatalf("want 1 completed scan, got %d err=%v", n, err)
	}
}

func TestLatestCompletedScanCommitSHA(t *testing.T) {
	s, err := Open(Config{Enabled: true, Path: t.TempDir() + "/tip.db"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	repo, err := s.UpsertRepository(ctx, Repository{Owner: "o", Name: "r", FullName: "o/r", ConnectedRepo: true})
	if err != nil {
		t.Fatalf("repo: %v", err)
	}
	got, err := s.LatestCompletedScanCommitSHA(ctx, repo.ID)
	if err != nil || got != "" {
		t.Fatalf("expected empty, got %q err=%v", got, err)
	}
	finished := time.Now().UTC()
	_, err = s.CreateScan(ctx, Scan{
		ID: "s1", RepositoryID: repo.ID, TriggerType: TriggerScheduled,
		Status: ScanStatusCompleted, CommitSHA: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		FinishedAt: &finished,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err = s.LatestCompletedScanCommitSHA(ctx, repo.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("got %q", got)
	}
}

func TestLatestCompletedScanCommitSHATieBreaksOnRowID(t *testing.T) {
	s, err := Open(Config{Enabled: true, Path: t.TempDir() + "/tip-tie.db"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	repo, err := s.UpsertRepository(ctx, Repository{Owner: "o", Name: "r", FullName: "o/tie", ConnectedRepo: true})
	if err != nil {
		t.Fatal(err)
	}
	same := time.Date(2026, 10, 16, 19, 27, 59, 0, time.UTC)
	older := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	newer := "cccccccccccccccccccccccccccccccccccccccc"
	for _, spec := range []Scan{
		{ID: "older", RepositoryID: repo.ID, TriggerType: TriggerScheduled, Status: ScanStatusCompleted, CommitSHA: older, StartedAt: same},
		{ID: "newer", RepositoryID: repo.ID, TriggerType: TriggerScheduled, Status: ScanStatusCompleted, CommitSHA: newer, StartedAt: same},
	} {
		if _, err := s.CreateScan(ctx, spec); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.LatestCompletedScanCommitSHA(ctx, repo.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != newer {
		t.Fatalf("same-timestamp scans must keep the later row, got %q", got)
	}
}
