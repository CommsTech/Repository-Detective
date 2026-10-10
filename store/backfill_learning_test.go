package store

import (
	"context"
	"testing"
)

func TestBackfillFalsePositiveLearningEvents(t *testing.T) {
	s, err := Open(Config{Enabled: true, Path: t.TempDir() + "/backfill.db"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()

	repo, err := s.UpsertRepository(ctx, Repository{Owner: "o", Name: "r", FullName: "o/r", ConnectedRepo: true})
	if err != nil {
		t.Fatalf("repo: %v", err)
	}
	scan, err := s.CreateScan(ctx, Scan{ID: "scan-backfill-1", RepositoryID: repo.ID, Status: "completed"})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	finding, err := s.UpsertFinding(ctx, Finding{
		RepositoryID:    repo.ID,
		Fingerprint:     "fp-backfill-test",
		Source:          "golangci-lint",
		RuleID:          "LINT-GO-typecheck",
		Status:          "suppressed",
		LastSeenScanID:  scan.ID,
		FirstSeenScanID: scan.ID,
	})
	if err != nil {
		t.Fatalf("finding: %v", err)
	}

	n, err := s.BackfillFalsePositiveLearningEvents(ctx, 10)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 backfill event, got %d", n)
	}
	n2, err := s.BackfillFalsePositiveLearningEvents(ctx, 10)
	if err != nil {
		t.Fatalf("backfill2: %v", err)
	}
	if n2 != 0 {
		t.Fatalf("expected idempotent backfill, got %d", n2)
	}

	recs, err := s.GenerateRepoScopedRecommendations(ctx, repo.ID, 5)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if recs != 0 {
		t.Fatalf("expected 0 recommendations from single event, got %d", recs)
	}
	_ = finding
}
