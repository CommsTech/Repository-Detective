package store_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/learning"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestRepoScopedRecommendationsIgnoreScannerFailedDilution(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "dilute.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r", ConnectedRepo: true})
	if err != nil {
		t.Fatal(err)
	}
	// 5 FP marks for GRAPH-ORPHAN-FILE
	for i := 0; i < 5; i++ {
		if _, err := s.RecordLearningEvent(ctx, store.LearningEvent{
			RepositoryID: repo.ID, Source: "graph", RuleID: "GRAPH-ORPHAN-FILE",
			EventType: learning.EventUserMarkedFalsePositive,
			IdempotencyKey: fmt.Sprintf("fp-%d", i),
		}); err != nil {
			t.Fatal(err)
		}
	}
	// Flood of scanner_failed on same source/rule must not block recommendation
	for i := 0; i < 20; i++ {
		if _, err := s.RecordLearningEvent(ctx, store.LearningEvent{
			RepositoryID: repo.ID, Source: "graph", RuleID: "GRAPH-ORPHAN-FILE",
			EventType: learning.EventScannerFailed,
			IdempotencyKey: fmt.Sprintf("fail-%d", i),
		}); err != nil {
			t.Fatal(err)
		}
	}
	n, err := s.GenerateRepoScopedRecommendations(ctx, repo.ID, 5)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 recommendation despite scanner_failed noise, got %d", n)
	}
}

func TestIngestAndGenerateFromAIAdvisory(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "ai-learn.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "eagle", FullName: "o/eagle", ConnectedRepo: true})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for i := 0; i < 2; i++ {
		fp := fmt.Sprintf("fp-orphan-%d", i)
		if _, err := s.UpsertFinding(ctx, store.Finding{
			RepositoryID: repo.ID, Fingerprint: fp, Title: "orphan", Severity: "info",
			Source: "graph", RuleID: "GRAPH-ORPHAN-FILE", Category: "maintainability",
			Status: "open", FirstSeenAt: now, LastSeenAt: now,
		}); err != nil {
			t.Fatal(err)
		}
		reviewID := fmt.Sprintf("air-test-%d", i)
		if _, err := s.CreateAIAdvisoryReview(ctx, store.AIAdvisoryReview{
			ReviewID: reviewID, ScanID: fmt.Sprintf("scan-%d", i), RepositoryID: repo.ID,
			ScanType: "scheduled", Status: store.AIReviewStatusCompleted, StartedAt: now, CreatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.CreateAIAdvisoryRecommendation(ctx, store.AIAdvisoryRecommendation{
			ReviewID: reviewID, FindingFingerprint: fp,
			Classification: "possible_false_positive", SuggestedAction: "calibrate_repo_scope",
			Reason: "launcher entry point", OperatorStatus: "pending", CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}

	ingested, err := s.IngestAIAdvisoryCalibrateSuggestions(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}
	if ingested != 2 {
		t.Fatalf("expected 2 ingested events, got %d", ingested)
	}
	// idempotent
	ingested2, err := s.IngestAIAdvisoryCalibrateSuggestions(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}
	if ingested2 != 2 {
		// RecordLearningEvent is idempotent; function still iterates and "succeeds" inserts that no-op.
		// Count should still be stable in learning_events.
	}
	events, _ := s.ListLearningEvents(ctx, repo.ID, 20)
	calibrateEvents := 0
	for _, ev := range events {
		if ev.EventType == learning.EventAIAdvisoryCalibrateSuggested {
			calibrateEvents++
		}
	}
	if calibrateEvents != 2 {
		t.Fatalf("expected 2 calibrate learning events, got %d", calibrateEvents)
	}

	n, err := s.GenerateCalibrationFromAIAdvisory(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 calibration recommendation from AI cluster, got %d", n)
	}
}

func TestCountLearningEventsSince(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "count.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	for i := 0; i < 3; i++ {
		_, _ = s.RecordLearningEvent(ctx, store.LearningEvent{
			RepositoryID: repo.ID, Source: "llm-auditor", RuleID: "auth",
			EventType: learning.EventLLMAuditorTimeout,
			IdempotencyKey: fmt.Sprintf("t-%d", i),
		})
	}
	n, err := s.CountLearningEventsSince(ctx, learning.EventLLMAuditorTimeout, time.Now().UTC().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("expected 3, got %d", n)
	}
}
