package store_test

import (
	"context"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/calibration"
	"git.commsnet.org/commstech/repository-detective/store"
)

func seedRepoAndFinding(t *testing.T, s store.QueryStore) (store.Repository, store.Finding) {
	t.Helper()
	ctx := context.Background()
	repo, err := s.UpsertRepository(ctx, store.Repository{
		Owner: "acme", Name: "cal", FullName: "acme/cal", ConnectedRepo: true,
	})
	if err != nil {
		t.Fatalf("repo: %v", err)
	}
	now := time.Now().UTC()
	finding, err := s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID,
		Fingerprint:  "fp-noise-001",
		Category:     "code_quality",
		Severity:     "low",
		Source:       "static",
		RuleID:       "RULE-TEST",
		Title:        "test finding",
		Status:       store.FindingStatusOpen,
		FirstSeenAt:  now,
		LastSeenAt:   now,
	})
	if err != nil {
		t.Fatalf("finding: %v", err)
	}
	return repo, finding
}

func TestFingerprintSuppressionRepoScoped(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, finding := seedRepoAndFinding(t, s)

	repoID := repo.ID
	_, err := s.CreateFindingSuppression(ctx, store.FindingSuppression{
		RepositoryID: &repoID,
		Fingerprint:  finding.Fingerprint,
		Scope:        store.SuppressionScopeRepo,
		Reason:       "noise",
		Active:       true,
	})
	if err != nil {
		t.Fatalf("create suppression: %v", err)
	}

	matcher := calibration.NewMatcher(s)
	if err := matcher.LoadRepository(ctx, repo.ID); err != nil {
		t.Fatalf("load: %v", err)
	}
	in := store.FindingMatchInput{
		RepositoryID: repo.ID,
		Fingerprint:  finding.Fingerprint,
		Source:       finding.Source,
		RuleID:       finding.RuleID,
	}
	if suppressed, _ := matcher.IsSuppressed(repo.ID, in); !suppressed {
		t.Fatal("expected fingerprint suppression to match")
	}

	issues := matcher.FilterIssues(repo.ID, []ai.CodeIssue{{
		Fingerprint: finding.Fingerprint,
		Source:      finding.Source,
		RuleID:      finding.RuleID,
		Severity:    "low",
	}})
	if len(issues) != 0 {
		t.Fatalf("expected suppressed issue filtered out, got %d", len(issues))
	}
}

func TestGraphOrphanRuleGlobalSuppression(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := seedRepoAndFinding(t, s)

	_, err := s.CreateFindingSuppression(ctx, store.FindingSuppression{
		RuleID: "GRAPH-ORPHAN-FILE",
		Source: "graph",
		Scope:  store.SuppressionScopeGlobal,
		Reason: "Beta closeout: intentional standalone/orphan graph noise",
		Active: true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	matcher := calibration.NewMatcher(s)
	_ = matcher.LoadRepository(ctx, repo.ID)
	in := store.FindingMatchInput{
		RepositoryID: repo.ID,
		Fingerprint:  "graph-orphan-fp-xyz",
		RuleID:       "GRAPH-ORPHAN-FILE",
		Source:       "graph",
		Category:     "maintainability",
		Severity:     "medium",
	}
	if suppressed, _ := matcher.IsSuppressed(repo.ID, in); !suppressed {
		t.Fatal("expected GRAPH-ORPHAN-FILE global suppression for closeout sprint")
	}
}

func TestRuleSuppressionGlobal(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, finding := seedRepoAndFinding(t, s)

	_, err := s.CreateFindingSuppression(ctx, store.FindingSuppression{
		RuleID: finding.RuleID,
		Source: finding.Source,
		Scope:  store.SuppressionScopeGlobal,
		Reason: "global rule noise",
		Active: true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	matcher := calibration.NewMatcher(s)
	_ = matcher.LoadRepository(ctx, repo.ID)
	in := store.FindingMatchInput{
		RepositoryID: repo.ID,
		Fingerprint:  "other-fingerprint",
		RuleID:       finding.RuleID,
		Source:       finding.Source,
	}
	if suppressed, _ := matcher.IsSuppressed(repo.ID, in); !suppressed {
		t.Fatal("expected global rule suppression")
	}
}

func TestExpiredSuppressionIgnored(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, finding := seedRepoAndFinding(t, s)
	repoID := repo.ID
	past := time.Now().UTC().Add(-time.Hour)
	_, err := s.CreateFindingSuppression(ctx, store.FindingSuppression{
		RepositoryID: &repoID,
		Fingerprint:  finding.Fingerprint,
		Scope:        store.SuppressionScopeRepo,
		ExpiresAt:    &past,
		Active:       true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	sups, err := s.ListActiveSuppressionsForRepository(ctx, repo.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, sup := range sups {
		if sup.Fingerprint == finding.Fingerprint {
			t.Fatal("expired suppression should not be active")
		}
	}
}

func TestDisableSuppression(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, finding := seedRepoAndFinding(t, s)
	repoID := repo.ID
	created, err := s.CreateFindingSuppression(ctx, store.FindingSuppression{
		RepositoryID: &repoID,
		Fingerprint:  finding.Fingerprint,
		Scope:        store.SuppressionScopeRepo,
		Active:       true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.DisableFindingSuppression(ctx, created.ID); err != nil {
		t.Fatalf("disable: %v", err)
	}
	sups, _ := s.ListActiveSuppressionsForRepository(ctx, repo.ID)
	for _, sup := range sups {
		if sup.ID == created.ID && sup.Active {
			t.Fatal("expected suppression disabled")
		}
	}
}

func TestListFindingsHideSuppressedByDefault(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, finding := seedRepoAndFinding(t, s)
	_ = s.UpdateFindingStatus(ctx, finding.ID, store.FindingStatusSuppressed)

	hidden, err := s.ListFindings(ctx, store.FindingFilter{RepositoryID: repo.ID, Limit: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, f := range hidden {
		if f.ID == finding.ID {
			t.Fatal("suppressed finding should be hidden by default")
		}
	}

	shown, err := s.ListFindings(ctx, store.FindingFilter{RepositoryID: repo.ID, IncludeSuppressed: true, Limit: 20})
	if err != nil {
		t.Fatalf("list shown: %v", err)
	}
	found := false
	for _, f := range shown {
		if f.ID == finding.ID {
			found = true
			if !f.Suppressed {
				t.Fatal("expected suppressed flag")
			}
		}
	}
	if !found {
		t.Fatal("expected suppressed finding when filter enabled")
	}
}

func TestScanQualityReport(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	_, _ = seedRepoAndFinding(t, s)
	report, err := s.ScanQualityReport(ctx)
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if report.TotalFindings < 1 {
		t.Fatalf("expected findings in report, got %d", report.TotalFindings)
	}
}
