package store

import (
	"context"
	"fmt"
	"testing"
	"time"
)

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
