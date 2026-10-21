package store

import (
	"context"
	"testing"
	"time"
)

func TestTryInsertOperatorEvidenceClaimsOnce(t *testing.T) {
	s, err := Open(Config{Enabled: true, Path: t.TempDir() + "/claim.db"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	key := EvidenceAIFirstRunRuleTuneKey(7)
	first, err := s.TryInsertOperatorEvidence(ctx, key, map[string]any{"status": "in_progress", "claimed_at": time.Now().UTC().Format(time.RFC3339)})
	if err != nil || !first {
		t.Fatalf("first claim ok=%v err=%v", first, err)
	}
	second, err := s.TryInsertOperatorEvidence(ctx, key, map[string]any{"status": "in_progress"})
	if err != nil || second {
		t.Fatalf("second claim must lose, ok=%v err=%v", second, err)
	}
	if err := s.DeleteOperatorEvidenceIfStatus(ctx, key, "in_progress"); err != nil {
		t.Fatal(err)
	}
	raw, err := s.GetOperatorEvidenceJSON(ctx, key)
	if err != nil || raw != "" {
		t.Fatalf("claim should be released, raw=%q err=%v", raw, err)
	}
}

func TestCreateRepoCalibrationRuleIsIdempotent(t *testing.T) {
	s, err := Open(Config{Enabled: true, Path: t.TempDir() + "/cal.db"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	repo, err := s.UpsertRepository(ctx, Repository{Owner: "o", Name: "r", FullName: "o/cal", ConnectedRepo: true})
	if err != nil {
		t.Fatal(err)
	}
	rule := RepoCalibrationRule{
		RepositoryID: &repo.ID, Scope: "repo", Source: "static", RuleID: "GOV-PIPELINE-SECRET-ECHO",
		Action: "report_only", Reason: "test", Active: true,
	}
	first, err := s.CreateRepoCalibrationRule(ctx, rule)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.CreateRepoCalibrationRule(ctx, rule)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("duplicate insert should return the existing rule, got %d and %d", first.ID, second.ID)
	}
	rules, err := s.ListRepoCalibrationRules(ctx, repo.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 {
		t.Fatalf("want 1 active rule, got %d", len(rules))
	}
}
