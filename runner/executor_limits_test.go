package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceExceedingFileLimitIsNotClean(t *testing.T) {
	root := t.TempDir()
	for _, f := range []string{"a.go", "b.go"} {
		if err := os.WriteFile(filepath.Join(root, f), []byte("package x\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := collectWorkspaceFiles(root, nil, 1); err == nil || !strings.Contains(err.Error(), "coverage incomplete") {
		t.Fatalf("truncated scan did not fail: %v", err)
	}
}
func TestWorkspaceOversizedFileIsNotClean(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "oversize.go"), make([]byte, 2*1024*1024+1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := collectWorkspaceFiles(root, nil, 50); err == nil || !strings.Contains(err.Error(), "coverage incomplete") {
		t.Fatalf("oversized workspace source did not fail: %v", err)
	}
}
func TestWorkspaceSymlinkIsNotTrusted(t *testing.T) {
	root := t.TempDir()
	if err := os.Symlink("/etc/passwd", filepath.Join(root, "passwd.go")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, _, err := collectWorkspaceFiles(root, nil, 50); err == nil {
		t.Fatal("worker silently skipped symlink")
	}
}
func TestWorkspaceBinaryMediaExcluded(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "logo.png"), make([]byte, 2*1024*1024+1), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "readme.go"), []byte("package x"), 0600); err != nil {
		t.Fatal(err)
	}
	sources, _, err := collectWorkspaceFiles(root, nil, 1)
	if err != nil || len(sources) != 1 || sources[0].Path != "readme.go" {
		t.Fatalf("sources=%v err=%v", sources, err)
	}
}
