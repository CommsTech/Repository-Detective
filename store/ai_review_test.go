package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestAIAdvisoryReviewCRUD(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "ai_review.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	now := time.Now().UTC()
	rec, err := s.CreateAIAdvisoryReview(ctx, store.AIAdvisoryReview{
		ReviewID: "air-test1", ScanID: "scan-1", RepositoryID: repo.ID,
		ScanType: "repo", Status: store.AIReviewStatusRunning,
		FindingsSent: 2, RedactionCount: 1, StartedAt: now, CreatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	rec.Status = store.AIReviewStatusCompleted
	rec.RecommendationsCount = 1
	rec.OverallAssessment = "ok"
	fin := now.Add(time.Minute)
	rec.FinishedAt = &fin
	if err := s.UpdateAIAdvisoryReview(ctx, rec); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetAIAdvisoryReviewByScanID(ctx, "scan-1")
	if err != nil || got.Status != store.AIReviewStatusCompleted {
		t.Fatalf("got %+v err=%v", got, err)
	}
	_, err = s.CreateAIAdvisoryRecommendation(ctx, store.AIAdvisoryRecommendation{
		ReviewID: "air-test1", FindingFingerprint: "fp1",
		Classification: "possible_false_positive", SuggestedAction: "calibrate_repo_scope",
		Reason: "test", OperatorStatus: "pending", CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	recs, err := s.ListAIAdvisoryRecommendations(ctx, "air-test1")
	if err != nil || len(recs) != 1 {
		t.Fatalf("recs=%d err=%v", len(recs), err)
	}
}
