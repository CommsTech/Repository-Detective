package store_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

// Priority must be selected in SQL before LIMIT, not by sorting a newest-first
// subset. This is essential for operators with large low-severity backlogs.
func TestFindingPriorityQueueDoesNotHideOlderCritical(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "team", Name: "app", FullName: "team/app"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	critical, err := s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "old-critical", Title: "Old critical", Severity: "critical", Status: "open", Category: "dependency", Source: "trivy", FirstSeenAt: now.Add(-48 * time.Hour), LastSeenAt: now.Add(-48 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	high, err := s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "old-high", Title: "Old high", Severity: "high", Status: "open", Category: "dependency", Source: "trivy", FirstSeenAt: now.Add(-47 * time.Hour), LastSeenAt: now.Add(-47 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 215; i++ {
		_, err := s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: fmt.Sprintf("new-medium-%03d", i), Title: "Medium noise", Severity: "medium", Status: "open", Category: "quality", Source: "ruff", FirstSeenAt: now.Add(time.Duration(i) * time.Second), LastSeenAt: now.Add(time.Duration(i) * time.Second)})
		if err != nil {
			t.Fatal(err)
		}
	}
	focus := store.FindingFilter{PriorityOnly: true, Prioritize: true, Status: "open", Limit: 1}
	count, err := s.CountFindings(ctx, focus)
	if err != nil || count != 2 {
		t.Fatalf("priority count=%d err=%v, want 2", count, err)
	}
	first, err := s.ListFindings(ctx, focus)
	if err != nil || len(first) != 1 || first[0].ID != critical.ID {
		t.Fatalf("first priority=%+v err=%v", first, err)
	}
	focus.Offset = 1
	second, err := s.ListFindings(ctx, focus)
	if err != nil || len(second) != 1 || second[0].ID != high.ID {
		t.Fatalf("second priority=%+v err=%v", second, err)
	}
	all, err := s.ListFindings(ctx, store.FindingFilter{Status: "open", Prioritize: true, Limit: 2})
	if err != nil || len(all) != 2 || all[0].ID != critical.ID || all[1].ID != high.ID {
		t.Fatalf("severity-first order=%+v err=%v", all, err)
	}
}
