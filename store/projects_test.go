package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestProjectGroupCRUD(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "projects.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	repoA, err := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "a", FullName: "o/a"})
	if err != nil {
		t.Fatal(err)
	}
	repoB, err := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "b", FullName: "o/b"})
	if err != nil {
		t.Fatal(err)
	}

	created, err := s.CreateProjectGroup(ctx, store.ProjectGroup{
		Name:                "app-stack",
		Description:         "frontend + backend",
		PrimaryRepositoryID: repoA.ID,
		RepositoryIDs:       []int64{repoA.ID, repoB.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID <= 0 {
		t.Fatal("expected group id")
	}

	groups, err := s.ListProjectGroups(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatalf("groups %d", len(groups))
	}
	if len(groups[0].RepositoryIDs) != 2 {
		t.Fatalf("repo ids %v", groups[0].RepositoryIDs)
	}
}

func TestSBOMArtifactPersist(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "sbom.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	scanID := "scan-sbom-1"
	if err := s.SaveSBOMArtifact(ctx, store.SBOMArtifact{
		RepositoryID: repo.ID,
		ScanID:       scanID,
		Format:       "CycloneDX",
		PackageCount: 42,
		VulnCount:    0,
		Status:       "sbom_check_clean",
		Detail:       "clean",
	}); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetSBOMArtifactForScan(ctx, scanID)
	if err != nil {
		t.Fatal(err)
	}
	if got.PackageCount != 42 || got.Status != "sbom_check_clean" {
		t.Fatalf("artifact %+v", got)
	}
}
