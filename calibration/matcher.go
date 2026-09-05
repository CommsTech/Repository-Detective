package calibration

import (
	"context"
	"sync"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/store"
)

// SuppressionStore loads active suppression rules.
type SuppressionStore interface {
	ListActiveSuppressionsForRepository(ctx context.Context, repositoryID int64) ([]store.FindingSuppression, error)
}

// Matcher evaluates suppression policy for scan-time actions.
type Matcher struct {
	store SuppressionStore
	mu    sync.Mutex
	cache map[int64]cacheEntry
}

type cacheEntry struct {
	loadedAt    time.Time
	suppressions []store.FindingSuppression
}

// NewMatcher creates a suppression matcher.
func NewMatcher(s SuppressionStore) *Matcher {
	return &Matcher{store: s, cache: map[int64]cacheEntry{}}
}

// LoadRepository refreshes cached suppressions for a repository.
func (m *Matcher) LoadRepository(ctx context.Context, repositoryID int64) error {
	if m == nil || m.store == nil || repositoryID <= 0 {
		return nil
	}
	sups, err := m.store.ListActiveSuppressionsForRepository(ctx, repositoryID)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.cache[repositoryID] = cacheEntry{loadedAt: time.Now().UTC(), suppressions: sups}
	m.mu.Unlock()
	return nil
}

// Invalidate clears cached suppressions for a repository.
func (m *Matcher) Invalidate(repositoryID int64) {
	if m == nil {
		return
	}
	m.mu.Lock()
	delete(m.cache, repositoryID)
	m.mu.Unlock()
}

func (m *Matcher) suppressionsFor(repositoryID int64) []store.FindingSuppression {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.cache[repositoryID]; ok {
		return e.suppressions
	}
	return nil
}

// IsSuppressed reports whether a finding or issue should be excluded from gates and forge actions.
func (m *Matcher) IsSuppressed(repositoryID int64, in store.FindingMatchInput) (bool, store.FindingSuppression) {
	if m == nil {
		return false, store.FindingSuppression{}
	}
	now := time.Now().UTC()
	return store.IsSuppressedByList(m.suppressionsFor(repositoryID), in, now)
}

// FilterIssues removes suppressed issues from forge/issue/remediation pipelines.
func (m *Matcher) FilterIssues(repositoryID int64, issues []ai.CodeIssue) []ai.CodeIssue {
	if m == nil || len(issues) == 0 {
		return issues
	}
	out := make([]ai.CodeIssue, 0, len(issues))
	for _, issue := range issues {
		in := store.FindingMatchInput{
			RepositoryID: repositoryID,
			Fingerprint:  issue.Fingerprint,
			Source:       issue.Source,
			RuleID:       issue.RuleID,
			Category:     issue.Category,
			Severity:     issue.Severity,
		}
		if suppressed, _ := m.IsSuppressed(repositoryID, in); suppressed {
			continue
		}
		out = append(out, issue)
	}
	return out
}

