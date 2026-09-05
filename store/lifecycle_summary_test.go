package store_test

import (
	"context"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestLifecycleSummaryCounts(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	sqlite := s.(*store.SQLiteStore)

	repo, _ := sqlite.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r", ConnectedRepo: true})
	f1, _ := sqlite.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "fp1", Title: "a", Severity: "low", Source: "staticcheck", Status: store.FindingStatusOpen})
	f2, _ := sqlite.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "fp2", Title: "b", Severity: "low", Source: "staticcheck", Status: "resolved_verified"})

	_, _ = sqlite.SaveRemediationPlan(ctx, store.RemediationPlanRecord{
		PlanID: "rp-1", FindingID: &f1.ID, RepositoryID: &repo.ID, Fingerprint: "fp1",
		Source: "staticcheck", Title: "t", Status: store.RemediationStatusProposed,
	})
	_, _ = sqlite.SaveRemediationPlan(ctx, store.RemediationPlanRecord{
		PlanID: "rp-2", FindingID: &f1.ID, RepositoryID: &repo.ID, Fingerprint: "fp1",
		Source: "staticcheck", Title: "t", Status: store.RemediationStatusApproved,
	})

	prNum := 1
	_, _ = sqlite.SavePatchAttempt(ctx, store.PatchAttemptRecord{
		AttemptID: "pa-1", PlanID: "rp-2", RepositoryID: repo.ID, FindingID: &f1.ID,
		BranchName: "b", BaseRef: "main", Status: store.PatchAttemptStatusPROpened, PullRequestNumber: &prNum,
	})
	_, _ = sqlite.SavePatchAttempt(ctx, store.PatchAttemptRecord{
		AttemptID: "pa-2", PlanID: "rp-2", RepositoryID: repo.ID, FindingID: &f2.ID,
		BranchName: "b2", BaseRef: "main", Status: store.PatchAttemptStatusPRMerged,
	})

	_, _ = sqlite.SaveClosureEvidence(ctx, store.ClosureEvidenceRecord{
		FindingID: f1.ID, RepositoryID: repo.ID, Fingerprint: "fp1", Status: store.ClosureStatusPendingRescan,
	})
	_, _ = sqlite.SaveClosureEvidence(ctx, store.ClosureEvidenceRecord{
		FindingID: f2.ID, RepositoryID: repo.ID, Fingerprint: "fp2", Status: store.ClosureStatusVerified,
	})

	ls, err := sqlite.LifecycleSummary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if ls.OpenFindings < 1 || ls.PROpened < 1 || ls.PendingRescan < 1 || ls.ResolvedVerified < 1 {
		t.Fatalf("unexpected lifecycle summary: %+v", ls)
	}
}
