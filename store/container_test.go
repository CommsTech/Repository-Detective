package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestContainerImageReferenceCRUD(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "container.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	ref, err := s.UpsertContainerImageReference(ctx, store.ContainerImageReference{
		RepositoryID: repo.ID, Image: "alpine:3.20", Tag: "3.20",
		TargetType: "registry_image", FilePath: "Dockerfile", Line: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	refs, _ := s.ListContainerImageReferences(ctx, repo.ID)
	if len(refs) != 1 || refs[0].ID != ref.ID {
		t.Fatalf("refs %+v", refs)
	}
}

func TestContainerImageScanRecord(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "cscan.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	sc, err := s.CreateContainerImageScan(ctx, store.ContainerImageScan{
		RepositoryID: repo.ID, ScanID: "cis-1", RunnerJobID: "rj-1",
		Image: "alpine:3.20", Status: store.ContainerScanStatusQueued,
		StartedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateContainerImageScan(ctx, sc.ID, store.ContainerScanStatusCompleted, "sha256:abc", 0, []byte(`{}`), []byte(`[]`), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	scans, _ := s.ListContainerImageScans(ctx, repo.ID, 10)
	if len(scans) != 1 || scans[0].Status != store.ContainerScanStatusCompleted {
		t.Fatalf("scans %+v", scans)
	}
}
