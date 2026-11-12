package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"git.commsnet.org/commstech/repository-detective/remediation"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestRemediationPlanCRUD(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Driver: "sqlite", Path: filepath.Join(dir, "rem.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r", ConnectedRepo: true})
	finding, _ := s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "fp-1", Title: "test", Severity: "high",
		Confidence: 0.9, Category: "code_quality", Source: "staticcheck",
	})

	fid := finding.ID
	rid := repo.ID
	plan := remediation.Plan{
		ID: "rp-test-1", FindingID: fid, RepositoryID: rid, Fingerprint: "fp-1",
		Category: "code_quality", Severity: "high", Source: "staticcheck",
		Title: "test", Summary: "summary", FixStrategy: "fix it",
		RegressionRisk: remediation.RiskLow, FixComplexity: remediation.ComplexitySmall,
		Status: remediation.StatusProposed,
	}
	rec, err := s.SaveRemediationPlan(ctx, store.RemediationPlanFromDomain(plan))
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetRemediationPlanByPlanID(ctx, rec.PlanID)
	if err != nil {
		t.Fatal(err)
	}
	if got.FixStrategy != "fix it" {
		t.Fatalf("unexpected strategy %q", got.FixStrategy)
	}
	latest, err := s.GetLatestRemediationPlanByFindingID(ctx, fid)
	if err != nil {
		t.Fatal(err)
	}
	if latest.PlanID != rec.PlanID {
		t.Fatal("latest mismatch")
	}
	if err := s.UpdateRemediationPlanStatus(ctx, rec.PlanID, store.RemediationStatusApproved); err != nil {
		t.Fatal(err)
	}
	_ = s.SupersedeRemediationPlansForFinding(ctx, fid)
}
