package scanners_test

import (
	"errors"
	"os"
	"path/filepath"
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

func TestWriteWorkspaceBytesRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if err := scanners.WriteWorkspaceBytes(root, "../escape.txt", []byte("x"), 0o600); err == nil {
		t.Fatal("expected traversal write to be rejected")
	} else if !errors.Is(err, scanners.ErrUnsafeWorkspacePath) {
		t.Fatalf("expected ErrUnsafeWorkspacePath, got %v", err)
	}
}

func TestWriteWorkspaceBytesWritesInsideRoot(t *testing.T) {
	root := t.TempDir()
	content := []byte("package main\n")
	if err := scanners.WriteWorkspaceBytes(root, "src/main.go", content, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "src", "main.go"))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(data) != string(content) {
		t.Fatalf("content mismatch: %q", data)
	}
}
