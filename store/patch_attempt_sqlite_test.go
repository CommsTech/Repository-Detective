package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"git.commsnet.org/commstech/repository-detective/patcher"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestPatchAttemptCRUD(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Driver: "sqlite", Path: filepath.Join(dir, "pa.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r", ConnectedRepo: true})
	attempt := patcher.PatchAttempt{
		ID: "pa-test-1", PlanID: "rp-1", RepositoryID: repo.ID, BranchName: "repository-detective/fix/x",
		BaseRef: "main", Status: patcher.StatusFailed, Error: "test",
	}
	rec, err := s.SavePatchAttempt(ctx, store.PatchAttemptFromDomain(attempt))
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetPatchAttemptByAttemptID(ctx, rec.AttemptID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Error != "test" {
		t.Fatalf("unexpected error %q", got.Error)
	}
	list, err := s.ListPatchAttemptsByPlanID(ctx, "rp-1")
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %v", list, err)
	}
}
