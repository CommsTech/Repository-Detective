package scanners_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"git.commsnet.org/commstech/repository-detective/scanners"
)

type stubArchiveDownloader struct {
	path    string
	cleanup func()
	err     error
}

func (s stubArchiveDownloader) DownloadRepositoryArchive(ctx context.Context, owner, repo, ref string, maxBytes int64) (string, func(), int64, error) {
	if s.err != nil {
		return "", nil, 0, s.err
	}
	return s.path, s.cleanup, 0, nil
}

func TestPrepareWorkspaceAPIMode(t *testing.T) {
	cfg := scanners.DefaultWorkspaceConfig()
	cfg.Mode = scanners.WorkspaceModeAPI

	prepared, err := scanners.PrepareWorkspace(context.Background(), cfg, nil, "o", "r", "abc1234", true, []scanners.FileEntry{
		{Path: "main.go", Content: "package main\n"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer prepared.Cleanup()

	if prepared.Meta.ModeUsed != scanners.WorkspaceModeAPI {
		t.Fatalf("expected api mode, got %s", prepared.Meta.ModeUsed)
	}
	if prepared.Meta.FallbackUsed {
		t.Fatal("expected no fallback")
	}
	if prepared.Meta.FileCount != 1 {
		t.Fatalf("expected 1 file, got %d", prepared.Meta.FileCount)
	}
}

func TestPrepareWorkspaceAutoFallsBackToAPI(t *testing.T) {
	cfg := scanners.DefaultWorkspaceConfig()
	cfg.Mode = scanners.WorkspaceModeAuto

	prepared, err := scanners.PrepareWorkspace(context.Background(), cfg, stubArchiveDownloader{err: fmt.Errorf("archive unavailable")}, "o", "r", "main", false, []scanners.FileEntry{
		{Path: "go.mod", Content: "module x\n\ngo 1.21\n"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer prepared.Cleanup()

	if prepared.Meta.ModeUsed != scanners.WorkspaceModeAPI {
		t.Fatalf("expected api fallback, got %s", prepared.Meta.ModeUsed)
	}
	if !prepared.Meta.FallbackUsed {
		t.Fatal("expected fallback_used=true")
	}
	if prepared.Meta.WorkspaceError == "" {
		t.Fatal("expected workspace_error to be recorded")
	}
}

func TestPrepareWorkspaceArchiveModeUsesZip(t *testing.T) {
	zipPath := writeTestZip(t, map[string]string{
		"repo-main/go.mod": "module example.com/test\n\ngo 1.21\n",
	})
	cleanup := func() { _ = os.Remove(zipPath) }

	cfg := scanners.DefaultWorkspaceConfig()
	cfg.Mode = scanners.WorkspaceModeArchive

	prepared, err := scanners.PrepareWorkspace(context.Background(), cfg, stubArchiveDownloader{path: zipPath, cleanup: cleanup}, "o", "r", "deadbeef", true, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer prepared.Cleanup()

	if prepared.Meta.ModeUsed != scanners.WorkspaceModeArchive {
		t.Fatalf("expected archive mode, got %s", prepared.Meta.ModeUsed)
	}
	if !prepared.Meta.CommitPinned {
		t.Fatal("expected commit_pinned=true")
	}
	if prepared.Meta.FileCount == 0 {
		t.Fatal("expected extracted files")
	}
}
