package store_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestBeginScanEnforcesCommunityRepoLimit(t *testing.T) {
	s, err := store.Open(store.Config{Enabled: true, Driver: "sqlite", Path: filepath.Join(t.TempDir(), "limit.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	r := store.NewRecorder(s, nil)
	r.SetMaxConnectedRepos(1)

	_, err = r.BeginScan(ctx, store.ScanContext{
		Owner: "o", Repo: "a", ScanID: "s1", ConnectedRepo: true, ForgeType: store.ForgeTypeGitea,
	})
	if err != nil {
		t.Fatalf("first repo: %v", err)
	}
	_, err = r.BeginScan(ctx, store.ScanContext{
		Owner: "o", Repo: "b", ScanID: "s2", ConnectedRepo: true, ForgeType: store.ForgeTypeGitea,
	})
	if !errors.Is(err, store.ErrRepoLimitExceeded) {
		t.Fatalf("expected ErrRepoLimitExceeded, got %v", err)
	}
	_, err = r.BeginScan(ctx, store.ScanContext{
		Owner: "o", Repo: "a", ScanID: "s3", ConnectedRepo: true, ForgeType: store.ForgeTypeGitea,
	})
	if err != nil {
		t.Fatalf("rescan existing: %v", err)
	}
}
