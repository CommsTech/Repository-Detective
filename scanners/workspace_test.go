package scanners_test

import (
	"errors"
	"testing"

	"git.commsnet.org/commstech/repository-detective/scanners"
)

func TestValidateWorkspacePathRejectsTraversal(t *testing.T) {
	root := t.TempDir()

	cases := []string{
		"../outside.txt",
		"foo/../../etc/passwd",
		"/etc/passwd",
		`C:\Windows\System32\cmd.exe`,
		`\\server\share\file.txt`,
	}

	for _, path := range cases {
		if _, err := scanners.ValidateWorkspacePath(root, path); err == nil {
			t.Fatalf("expected unsafe path %q to be rejected", path)
		} else if !errors.Is(err, scanners.ErrUnsafeWorkspacePath) {
			t.Fatalf("path %q: expected ErrUnsafeWorkspacePath, got %v", path, err)
		}
	}
}

func TestValidateWorkspacePathAllowsNormalPaths(t *testing.T) {
	root := t.TempDir()

	clean, err := scanners.ValidateWorkspacePath(root, "src/main.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clean != "src/main.go" {
		t.Fatalf("expected cleaned path src/main.go, got %q", clean)
	}
}

func TestCreateWorkspaceRejectsMaliciousPath(t *testing.T) {
	_, cleanup, err := scanners.CreateWorkspace([]scanners.FileEntry{
		{Path: "../escape.go", Content: "package escape"},
	})
	if cleanup != nil {
		cleanup()
	}
	if err == nil {
		t.Fatal("expected malicious path to fail workspace creation")
	}
}

func TestCreateWorkspaceWritesNormalPaths(t *testing.T) {
	dir, cleanup, err := scanners.CreateWorkspace([]scanners.FileEntry{
		{Path: "src/main.go", Content: "package main\n"},
		{Path: "go.mod", Content: "module example.com/test\n\ngo 1.21\n"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	if dir == "" {
		t.Fatal("expected workspace dir")
	}
}
