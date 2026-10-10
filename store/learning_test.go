package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/findinglearn"
	"git.commsnet.org/commstech/repository-detective/learning"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestLearningEventIdempotent(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "learn.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	ev := store.LearningEvent{
		RepositoryID: repo.ID, EventType: learning.EventResolvedVerified,
		Source: "graph", RuleID: "GRAPH-ORPHAN", IdempotencyKey: "test:1",
	}
	if _, err := s.RecordLearningEvent(ctx, ev); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RecordLearningEvent(ctx, ev); err != nil {
		t.Fatal(err)
	}
	events, _ := s.ListLearningEvents(ctx, repo.ID, 10)
	if len(events) != 1 {
		t.Fatalf("expected 1 event got %d", len(events))
	}
}

func TestRepoScopedRecommendationRequiresEvidence(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "rec.db")})
	defer s.Close()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "a", FullName: "o/a"})
	n, err := s.GenerateRepoScopedRecommendations(ctx, repo.ID, 5)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("expected 0 recommendations without events, got %d", n)
	}
}

func TestStructuralGroupAssign(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "struct.db")})
	defer s.Close()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	now := time.Now().UTC()
	f1, _ := s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "fp1", Title: "t", Severity: "medium",
		Source: "semgrep", Status: "open", FirstSeenAt: now, LastSeenAt: now,
	})
	hash := findinglearn.StructuralHash("R1", "security", "eval(x)")
	if err := s.AssignStructuralGroup(ctx, repo.ID, hash, f1.ID); err != nil {
		t.Fatal(err)
	}
}

func TestRepoCalibrationRuleCRUD(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "rule.db")})
	defer s.Close()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	rid := repo.ID
	rule, err := s.CreateRepoCalibrationRule(ctx, store.RepoCalibrationRule{
		RepositoryID: &rid, Scope: "repo", Source: "ruff", RuleID: "I001",
		Action: "downgrade_to_info", Reason: "test", Active: true, EvidenceCount: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	rules, _ := s.ListRepoCalibrationRules(ctx, repo.ID, true)
	if len(rules) != 1 || rules[0].ID != rule.ID {
		t.Fatalf("rules %+v", rules)
	}
}
