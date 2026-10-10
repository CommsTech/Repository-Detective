package preinstall_test

import (
	"os"
	"path/filepath"
	"testing"

	"git.commsnet.org/commstech/repository-detective/preinstall"
)

func TestValidateWorkspacePathBlocksTraversal(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "..", "outside.txt")
	if err := preinstall.ValidateWorkspacePathForTest(root, outside); err == nil {
		t.Fatal("expected path escape to be blocked")
	}
}

func TestMeasureWorkspaceSandboxFileCountLimit(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(filepath.Join(root, "f"+string(rune('a'+i))+".txt"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg := preinstall.DefaultConfig()
	cfg.MaxFiles = 3
	if _, _, err := preinstall.MeasureWorkspaceSandboxForTest(root, cfg); err == nil {
		t.Fatal("expected file count limit error")
	}
}

func TestGitCloneArgsDisableSubmodulesAndHooks(t *testing.T) {
	args := preinstall.GitCloneArgsForTests("https://github.com/o/r.git", "/tmp/repo")
	joined := stringsJoin(args)
	if !containsAll(joined, "--no-recurse-submodules", "core.hooksPath=/dev/null") {
		t.Fatalf("unexpected clone args: %v", args)
	}
}

func stringsJoin(parts []string) string {
	out := ""
	for _, p := range parts {
		out += p + " "
	}
	return out
}

func containsAll(hay string, parts ...string) bool {
	for _, p := range parts {
		if !stringsContains(hay, p) {
			return false
		}
	}
	return true
}

func stringsContains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
