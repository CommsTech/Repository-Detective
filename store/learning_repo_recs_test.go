package store

import (
	"context"
	"fmt"
	"testing"
)

func TestGenerateRepoScopedRecommendationsFromFalsePositives(t *testing.T) {
	s, err := Open(Config{Enabled: true, Path: t.TempDir() + "/learn.db"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	repo, err := s.UpsertRepository(ctx, Repository{Owner: "o", Name: "r", FullName: "o/r", ConnectedRepo: true})
	if err != nil {
		t.Fatalf("repo: %v", err)
	}
	for i := 0; i < 5; i++ {
		_, err := s.RecordLearningEvent(ctx, LearningEvent{
			RepositoryID:   repo.ID,
			ScanID:         "scan-1",
			Source:         "graph",
			RuleID:         "GRAPH-ORPHAN-FILE",
			EventType:      "user_marked_false_positive",
			IdempotencyKey: fmt.Sprintf("fp-%d", i),
		})
		if err != nil {
			t.Fatalf("event %d: %v", i, err)
		}
	}
	n, err := s.GenerateRepoScopedRecommendations(ctx, repo.ID, 5)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 recommendation, got %d", n)
	}
	recs, err := s.ListCalibrationRecommendations(ctx, "proposed", 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("expected 1 proposed rec, got %d", len(recs))
	}
	if recs[0].Scope != "repo" || recs[0].RuleID != "GRAPH-ORPHAN-FILE" {
		t.Fatalf("unexpected rec: %+v", recs[0])
	}
	if recs[0].RepositoryID == nil || *recs[0].RepositoryID != repo.ID {
		t.Fatalf("expected repo id %d, got %+v", repo.ID, recs[0].RepositoryID)
	}
	// Second generate should be idempotent while proposed exists.
	n2, err := s.GenerateRepoScopedRecommendations(ctx, repo.ID, 5)
	if err != nil {
		t.Fatalf("generate2: %v", err)
	}
	if n2 != 0 {
		t.Fatalf("expected 0 duplicate recommendations, got %d", n2)
	}
}
