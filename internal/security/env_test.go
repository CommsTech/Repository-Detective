package security

import (
	"strings"
	"testing"
)

func TestMinimalSubprocessEnvDoesNotForwardSecrets(t *testing.T) {
	t.Setenv("REPOSITORY_DETECTIVE_API_KEY", "super-secret-key")
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAEXAMPLE")
	t.Setenv("GITHUB_TOKEN", "ghp_example")
	t.Setenv("SSH_AUTH_SOCK", "/tmp/ssh-agent")

	env := MinimalSubprocessEnv()
	joined := strings.Join(env, "\n")
	for _, forbidden := range []string{"REPOSITORY_DETECTIVE_API_KEY", "AWS_ACCESS_KEY_ID", "GITHUB_TOKEN", "SSH_AUTH_SOCK"} {
		if strings.Contains(joined, forbidden+"=") {
			t.Fatalf("forbidden var %s leaked into subprocess env: %v", forbidden, env)
		}
	}
}

func TestMinimalSubprocessEnvIncludesSafeGitDefaults(t *testing.T) {
	env := MinimalSubprocessEnv()
	joined := strings.Join(env, "\n")
	for _, want := range []string{"GIT_TERMINAL_PROMPT=0", "GIT_SSH_COMMAND=disabled"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %v", want, env)
		}
	}
}

func TestSafeAttachmentFilename(t *testing.T) {
	name := SafeAttachmentFilename("../../etc/passwd", "scan-1", ".json")
	if strings.Contains(name, "..") || strings.Contains(name, "/") {
		t.Fatalf("unsafe filename: %q", name)
	}
	if !strings.HasPrefix(name, "passwd") {
		t.Fatalf("unexpected basename: %q", name)
	}
}
