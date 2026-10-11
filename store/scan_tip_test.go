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
