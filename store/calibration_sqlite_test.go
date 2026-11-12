package store

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestGetCalibrationRecommendationByIDNotLimitedByListWindow(t *testing.T) {
	qs, err := Open(Config{Enabled: true, Path: t.TempDir() + "/cal-get.db"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = qs.Close() })
	s, ok := qs.(*SQLiteStore)
	if !ok {
		t.Fatalf("expected *SQLiteStore, got %T", qs)
	}
	ctx := context.Background()
	repo, err := s.UpsertRepository(ctx, Repository{Owner: "o", Name: "get", FullName: "o/get", ConnectedRepo: true})
	if err != nil {
		t.Fatalf("repo: %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	// Insert many high-confidence rows so a list LIMIT 1000 would hide a low-confidence target.
	for i := 0; i < 1100; i++ {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO calibration_recommendations (
				scope, repository_id, recommendation_type, source, rule_id, category,
				current_action, recommended_action, reason, confidence, status, created_at, updated_at
			) VALUES ('repo', ?, 'downgrade_confidence', 'graph', ?, 'quality', 'auto_issue', 'report_only', 'noise', 0.99, 'proposed', ?, ?)
		`, repo.ID, fmt.Sprintf("GRAPH-ORPHAN-FILE-%d", i), now, now)
		if err != nil {
			t.Fatalf("seed high confidence: %v", err)
		}
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO calibration_recommendations (
			scope, repository_id, recommendation_type, source, rule_id, category,
			current_action, recommended_action, reason, confidence, status, created_at, updated_at
		) VALUES ('repo', ?, 'downgrade_confidence', 'static', 'OPT-NESTED-LOOP', 'quality', 'auto_issue', 'report_only', 'target', 0.10, 'proposed', ?, ?)
	`, repo.ID, now, now)
	if err != nil {
		t.Fatalf("seed target: %v", err)
	}
	targetID, _ := res.LastInsertId()

	listed, err := s.ListCalibrationRecommendations(ctx, "proposed", 1000)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, rec := range listed {
		if rec.ID == targetID {
			t.Fatalf("target id %d unexpectedly present in confidence-ordered top 1000", targetID)
		}
	}
	got, err := s.GetCalibrationRecommendation(ctx, targetID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.ID != targetID || got.RuleID != "OPT-NESTED-LOOP" {
		t.Fatalf("got %+v", got)
	}
}

func TestRecomputeCalibrationRuleStatsPerformance(t *testing.T) {
	s, err := Open(Config{Enabled: true, Path: t.TempDir() + "/cal.db"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, Repository{Owner: "o", Name: "c", FullName: "o/c", ConnectedRepo: true})
	now := time.Now().UTC()
	for i := 0; i < 500; i++ {
		f, _ := s.UpsertFinding(ctx, Finding{
			RepositoryID: repo.ID,
			Fingerprint:  fmt.Sprintf("fp-cal-%d", i),
			Source:       "graph",
			RuleID:       "GRAPH-ORPHAN-FILE",
			Category:     "maintainability",
			Status:       FindingStatusOpen,
			FirstSeenAt:  now,
			LastSeenAt:   now,
		})
		if i%3 == 0 {
			_, _ = s.UpsertExternalIssue(ctx, ExternalIssue{
				FindingID: f.ID, ForgeType: "gitea", IssueNumber: i + 1,
				IssueURL: "http://x", State: "open",
			})
		}
	}
	start := time.Now()
	n, err := s.RecomputeCalibrationRuleStats(ctx)
	if err != nil {
		t.Fatalf("recompute: %v", err)
	}
	if n == 0 {
		t.Fatal("expected rule stats rows")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("recompute took %s, want under 5s", elapsed)
	}
}
