package store

import (
	"context"
	"testing"
)

func TestExpireProtectedActiveCalibrationRules(t *testing.T) {
	qs, err := Open(Config{Enabled: true, Path: t.TempDir() + "/expire-protect.db"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = qs.Close() })
	s, ok := qs.(*SQLiteStore)
	if !ok {
		t.Fatal("expected *SQLiteStore")
	}
	ctx := context.Background()

	repo, err := s.UpsertRepository(ctx, Repository{Owner: "o", Name: "r", FullName: "o/r", ConnectedRepo: true})
	if err != nil {
		t.Fatalf("repo: %v", err)
	}
	rid := repo.ID

	safe, err := s.CreateRepoCalibrationRule(ctx, RepoCalibrationRule{
		RepositoryID: &rid, Scope: "repo", Source: "static", RuleID: "QUAL-DEBUG",
		Action: "report_only", Reason: "noise", Active: true,
	})
	if err != nil {
		t.Fatalf("safe rule: %v", err)
	}

	// Bypass CreateRepoCalibrationRule protection to simulate legacy bad rows.
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO repo_calibration_rules (
			repository_id, scope, source, rule_id, action, reason, active, created_at, updated_at
		) VALUES (?, 'repo', 'gosec', 'G304', 'report_only', 'legacy bad quiet', 1, datetime('now'), datetime('now'))
	`, rid)
	if err != nil {
		t.Fatalf("insert legacy protected rule: %v", err)
	}

	n, err := s.ExpireProtectedActiveCalibrationRules(ctx)
	if err != nil {
		t.Fatalf("expire: %v", err)
	}
	if n < 1 {
		t.Fatalf("expected at least 1 protected rule expired, got %d", n)
	}

	rules, err := s.ListRepoCalibrationRules(ctx, rid, true)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, r := range rules {
		if protectedRuleIDForLearning(r.Source, r.RuleID) {
			t.Fatalf("protected rule still active: %s/%s", r.Source, r.RuleID)
		}
	}
	foundSafe := false
	for _, r := range rules {
		if r.ID == safe.ID {
			foundSafe = true
		}
	}
	if !foundSafe {
		t.Fatal("non-protected QUAL-DEBUG calibration should remain active")
	}
}

func TestCreateRepoCalibrationRuleRejectsProtected(t *testing.T) {
	s, err := Open(Config{Enabled: true, Path: t.TempDir() + "/reject-protect.db"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	repo, err := s.UpsertRepository(ctx, Repository{Owner: "o", Name: "r", FullName: "o/r2", ConnectedRepo: true})
	if err != nil {
		t.Fatalf("repo: %v", err)
	}
	rid := repo.ID
	_, err = s.CreateRepoCalibrationRule(ctx, RepoCalibrationRule{
		RepositoryID: &rid, Scope: "repo", Source: "gosec", RuleID: "G304",
		Action: "report_only", Reason: "should fail", Active: true,
	})
	if err == nil {
		t.Fatal("expected refusal for protected gosec/G304 calibration")
	}
}

func TestRejectProtectedCalibrationRecommendations(t *testing.T) {
	qs, err := Open(Config{Enabled: true, Path: t.TempDir() + "/reject-rec.db"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = qs.Close() })
	s, ok := qs.(*SQLiteStore)
	if !ok {
		t.Fatal("expected *SQLiteStore")
	}
	ctx := context.Background()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO calibration_recommendations (
			scope, recommendation_type, source, rule_id, category,
			current_action, recommended_action, reason, confidence, status, created_at, updated_at
		) VALUES
		('repo', 'report_only', 'gosec', 'G304', 'security', 'auto_issue', 'report_only', 'bad', 0.9, 'proposed', datetime('now'), datetime('now')),
		('repo', 'report_only', 'static', 'QUAL-DEBUG', 'quality', 'auto_issue', 'report_only', 'ok', 0.9, 'proposed', datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("insert recommendations: %v", err)
	}
	n, err := s.RejectProtectedCalibrationRecommendations(ctx)
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 rejected, got %d", n)
	}
	var status string
	if err := s.db.QueryRowContext(ctx, `
		SELECT status FROM calibration_recommendations WHERE rule_id = 'G304'
	`).Scan(&status); err != nil {
		t.Fatalf("scan g304: %v", err)
	}
	if status != "rejected" {
		t.Fatalf("G304 recommendation status=%s, want rejected", status)
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT status FROM calibration_recommendations WHERE rule_id = 'QUAL-DEBUG'
	`).Scan(&status); err != nil {
		t.Fatalf("scan qual: %v", err)
	}
	if status != "proposed" {
		t.Fatalf("QUAL-DEBUG should stay proposed, got %s", status)
	}
}
