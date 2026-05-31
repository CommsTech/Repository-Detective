package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/bugbot/store"
)

func TestClosureEvidenceCRUD(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Driver: "sqlite", Path: filepath.Join(dir, "closure.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r", ConnectedRepo: true})
	finding, _ := s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "fp-cl", Title: "t", Severity: "medium",
		Source: "staticcheck", Status: store.FindingStatusOpen,
	})

	rec, err := s.SaveClosureEvidence(ctx, store.ClosureEvidenceRecord{
		FindingID: finding.ID, RepositoryID: repo.ID, Fingerprint: "fp-cl",
		OriginalSource: "staticcheck", Status: store.ClosureStatusPendingRescan,
		Reason: "waiting", MergeCommitSHA: "abc123",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetLatestClosureEvidenceByFindingID(ctx, finding.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.MergeCommitSHA != "abc123" {
		t.Fatalf("unexpected merge sha %q", got.MergeCommitSHA)
	}
	got.VerificationScanID = "scan-1"
	got.Status = store.ClosureStatusVerified
	if err := s.UpdateClosureEvidence(ctx, got); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdatePatchAttemptMerged(ctx, "pa-x", "sha", time.Now()); err == nil {
		t.Fatal("expected error for missing attempt")
	}
	_ = rec
}
