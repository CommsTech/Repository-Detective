package calibration

import (
	"context"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/store"
)

type stubSuppressionStore struct {
	byRepo map[int64][]store.FindingSuppression
	err    error
}

func (s *stubSuppressionStore) ListActiveSuppressionsForRepository(_ context.Context, repositoryID int64) ([]store.FindingSuppression, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.byRepo[repositoryID], nil
}

func repoID(id int64) *int64 { return &id }

func TestMatcherIsSuppressedAfterLoad(t *testing.T) {
	now := time.Now().UTC()
	stub := &stubSuppressionStore{
		byRepo: map[int64][]store.FindingSuppression{
			1: {{
				RepositoryID: repoID(1),
				Scope:        store.SuppressionScopeRepo,
				Source:       "static",
				RuleID:       "QUAL-DEBUG",
				Active:       true,
				CreatedAt:    now.Add(-time.Hour),
			}},
			2: {{
				RepositoryID: repoID(2),
				Scope:        store.SuppressionScopeRepo,
				Source:       "graph",
				RuleID:       "GRAPH-ORPHAN-FILE",
				Active:       true,
				CreatedAt:    now.Add(-time.Hour),
			}},
		},
	}
	m := NewMatcher(stub)
	ctx := context.Background()

	if err := m.LoadRepository(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if err := m.LoadRepository(ctx, 2); err != nil {
		t.Fatal(err)
	}

	suppressed, _ := m.IsSuppressed(1, store.FindingMatchInput{
		RepositoryID: 1,
		Source:       "static",
		RuleID:       "QUAL-DEBUG",
	})
	if !suppressed {
		t.Fatal("repo 1 QUAL-DEBUG should be suppressed")
	}

	suppressed, _ = m.IsSuppressed(1, store.FindingMatchInput{
		RepositoryID: 1,
		Source:       "static",
		RuleID:       "SEC-EVAL",
	})
	if suppressed {
		t.Fatal("unrelated rule must not be suppressed")
	}

	suppressed, _ = m.IsSuppressed(1, store.FindingMatchInput{
		RepositoryID: 1,
		Source:       "graph",
		RuleID:       "GRAPH-ORPHAN-FILE",
	})
	if suppressed {
		t.Fatal("repo 2 suppression must not apply to repo 1")
	}
}

func TestMatcherFilterIssues(t *testing.T) {
	now := time.Now().UTC()
	stub := &stubSuppressionStore{
		byRepo: map[int64][]store.FindingSuppression{
			3: {{
				RepositoryID: repoID(3),
				Scope:        store.SuppressionScopeRepo,
				Source:       "ruff",
				RuleID:       "LINT-RUFF-E902",
				Active:       true,
				CreatedAt:    now.Add(-time.Hour),
			}},
		},
	}
	m := NewMatcher(stub)
	if err := m.LoadRepository(context.Background(), 3); err != nil {
		t.Fatal(err)
	}

	issues := []ai.CodeIssue{
		{Source: "ruff", RuleID: "LINT-RUFF-E902", Title: "noise"},
		{Source: "static", RuleID: "SEC-EVAL", Title: "real"},
	}
	filtered := m.FilterIssues(3, issues)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 issue after filter, got %d", len(filtered))
	}
	if filtered[0].RuleID != "SEC-EVAL" {
		t.Fatalf("wrong issue kept: %s", filtered[0].RuleID)
	}
}

func TestMatcherInvalidate(t *testing.T) {
	now := time.Now().UTC()
	stub := &stubSuppressionStore{
		byRepo: map[int64][]store.FindingSuppression{
			4: {{
				RepositoryID: repoID(4),
				Scope:        store.SuppressionScopeRepo,
				Source:       "static",
				RuleID:       "QUAL-DEBUG",
				Active:       true,
				CreatedAt:    now.Add(-time.Hour),
			}},
		},
	}
	m := NewMatcher(stub)
	ctx := context.Background()
	if err := m.LoadRepository(ctx, 4); err != nil {
		t.Fatal(err)
	}
	m.Invalidate(4)
	suppressed, _ := m.IsSuppressed(4, store.FindingMatchInput{
		RepositoryID: 4,
		Source:       "static",
		RuleID:       "QUAL-DEBUG",
	})
	if suppressed {
		t.Fatal("cache should be empty after invalidate")
	}
}
