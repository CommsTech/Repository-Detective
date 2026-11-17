package runner

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Input rejection happens before any network or filesystem mutations.
func TestCloneRepositoryRejectsUnsafeTargets(t *testing.T) {
	tests := []struct{ name, url, ref, sha string }{
		{"file protocol", "file:///etc/passwd", "main", ""},
		{"git ext command", "ext::sh -c id", "main", ""},
		{"embedded HTTP token", "https://user:secret@git.example/r.git", "main", ""},
		{"missing host", "https:///r.git", "main", ""},
		{"option-like branch", "https://git.example/r.git", "--upload-pack=evil", ""},
		{"newline branch", "https://git.example/r.git", "main\nother", ""},
		{"bad sha", "https://git.example/r.git", "main", "abc123"},
		{"bad sha64", "https://git.example/r.git", "main", strings.Repeat("z", 64)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := CloneRepositoryAtCommit(context.Background(), test.url, test.ref, test.sha, t.TempDir()+"/scan", 2*time.Second); err == nil {
				t.Fatal("unsafe clone request accepted")
			}
		})
	}
}

func TestWorkspaceIdentifierSafety(t *testing.T) {
	for _, name := range []string{"", "../escape", "rj-../../escape", "/tmp/escape", "rj-a/b", "rj-🦄", "j1"} {
		if safeRunnerJobID.MatchString(name) {
			t.Fatalf("unsafe job ID allowed: %q", name)
		}
	}
	if !safeRunnerJobID.MatchString("rj-00ffaab112233445") {
		t.Fatal("normal generated job id rejected")
	}
}
