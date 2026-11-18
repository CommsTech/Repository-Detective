package store_test

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"git.commsnet.org/commstech/repository-detective/remediation"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestRemediationReviewIsOneWayAndAtomic(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "review.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
	})
	for _, decision := range []string{store.RemediationStatusApproved, store.RemediationStatusRejected} {
		plan, err := s.SaveRemediationPlan(ctx, store.RemediationPlanFromDomain(remediation.Plan{ID: "review-" + decision, Status: remediation.StatusProposed}))
		if err != nil {
			t.Fatal(err)
		}
		if err := s.UpdateRemediationPlanStatus(ctx, plan.PlanID, decision); err != nil {
			t.Fatal(err)
		}
		reverse := store.RemediationStatusApproved
		if decision == reverse {
			reverse = store.RemediationStatusRejected
		}
		if err := s.UpdateRemediationPlanStatus(ctx, plan.PlanID, reverse); err == nil {
			t.Fatal("terminal review decision was reversed")
		}
		if err := s.UpdateRemediationPlanStatus(ctx, plan.PlanID, decision); err == nil {
			t.Fatal("duplicate review unexpectedly accepted")
		}
		if err := s.UpdateRemediationPlanStatus(ctx, plan.PlanID, store.RemediationStatusSuperseded); err == nil {
			t.Fatal("non-review target unexpectedly accepted")
		}
		got, err := s.GetRemediationPlanByPlanID(ctx, plan.PlanID)
		if err != nil || got.Status != decision {
			t.Fatalf("status=%q err=%v", got.Status, err)
		}
	}
	plan, err := s.SaveRemediationPlan(ctx, store.RemediationPlanFromDomain(remediation.Plan{ID: "review-concurrent", Status: remediation.StatusProposed}))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	ch := make(chan error, 2)
	for _, status := range []string{store.RemediationStatusApproved, store.RemediationStatusRejected} {
		wg.Add(1)
		go func(decision string) {
			defer wg.Done()
			ch <- s.UpdateRemediationPlanStatus(ctx, plan.PlanID, decision)
		}(status)
	}
	wg.Wait()
	close(ch)
	accepted := 0
	for e := range ch {
		if e == nil {
			accepted++
		} else if !strings.Contains(e.Error(), "no longer awaiting review") {
			t.Errorf("unexpected error: %v", e)
		}
	}
	if accepted != 1 {
		t.Fatalf("wanted exactly one accepted review, got %d", accepted)
	}
}

func TestRemediationSupersededCannotBeApproved(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "superseded.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	if err != nil {
		t.Fatal(err)
	}
	finding, err := s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "review-fp", Title: "test", Severity: "medium"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := s.SaveRemediationPlan(ctx, store.RemediationPlanFromDomain(remediation.Plan{ID: "supersede-1", FindingID: finding.ID, RepositoryID: repo.ID, Status: remediation.StatusProposed}))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SupersedeRemediationPlansForFinding(ctx, finding.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateRemediationPlanStatus(ctx, plan.PlanID, store.RemediationStatusApproved); err == nil {
		t.Fatal("superseded plan approved")
	}
}
