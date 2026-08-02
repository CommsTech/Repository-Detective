package scanners

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/internal/security"
)

// GitHistoryWorkspace is a temporary git clone for history secret scanning.
type GitHistoryWorkspace struct {
	Dir     string
	Cleanup func()
}

// PrepareGitHistoryWorkspace shallow- or full-clones cloneURL for gitleaks detect mode.
// maxCommits 0 means full history (no --depth limit on clone).
func PrepareGitHistoryWorkspace(ctx context.Context, cloneURL, ref string, maxCommits int, timeoutSeconds int) (GitHistoryWorkspace, error) {
	cloneURL = strings.TrimSpace(cloneURL)
	if cloneURL == "" {
		return GitHistoryWorkspace{}, fmt.Errorf("clone URL is required for git history secret scan")
	}
	if security.SubprocessEnvExposesSecrets() {
		return GitHistoryWorkspace{}, fmt.Errorf("unsafe subprocess environment for git clone")
	}

	parent, err := os.MkdirTemp("", "rd-gitleaks-history-*")
	if err != nil {
		return GitHistoryWorkspace{}, fmt.Errorf("create temp dir: %w", err)
	}
	dest := filepath.Join(parent, "repo")
	cleanup := func() { _ = os.RemoveAll(parent) }

	cloneCtx := ctx
	if timeoutSeconds > 0 {
		var cancel context.CancelFunc
		cloneCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
		defer cancel()
	}

	cloneArgs := []string{"clone", "--single-branch", "--no-tags", "--"}
	if maxCommits > 0 {
		cloneArgs = append(cloneArgs, fmt.Sprintf("--depth=%d", maxCommits))
	}
	cloneArgs = append(cloneArgs, cloneURL, dest)

	if out, err := runGitHistory(cloneCtx, cloneArgs); err != nil {
		cleanup()
		return GitHistoryWorkspace{}, fmt.Errorf("git clone for history scan failed: %w", sanitizeHistoryGitError(out))
	}

	ref = strings.TrimSpace(ref)
	if ref != "" && ref != "HEAD" && !looksLikeCommitSHA(ref) {
		if out, err := runGitHistory(cloneCtx, []string{"-C", dest, "fetch", "origin", ref+":"+ref, "--depth=1"}); err == nil {
			_, _ = out, err
			if out2, err2 := runGitHistory(cloneCtx, []string{"-C", dest, "checkout", ref}); err2 != nil {
				cleanup()
				return GitHistoryWorkspace{}, fmt.Errorf("checkout ref %q: %w", ref, sanitizeHistoryGitError(out2))
			}
		} else if out2, err2 := runGitHistory(cloneCtx, []string{"-C", dest, "checkout", ref}); err2 != nil {
			cleanup()
			return GitHistoryWorkspace{}, fmt.Errorf("checkout ref %q: %w", ref, sanitizeHistoryGitError(out2))
		}
	}

	if _, err := os.Stat(filepath.Join(dest, ".git")); err != nil {
		cleanup()
		return GitHistoryWorkspace{}, fmt.Errorf("cloned workspace is not a git repository")
	}

	return GitHistoryWorkspace{Dir: dest, Cleanup: cleanup}, nil
}

func runGitHistory(ctx context.Context, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = security.MinimalSubprocessEnv()
	out, err := cmd.CombinedOutput()
	const maxOut = 256 << 10
	if len(out) > maxOut {
		out = out[:maxOut]
	}
	return out, err
}

func sanitizeHistoryGitError(out []byte) error {
	_ = out
	return fmt.Errorf("git operation failed")
}

func looksLikeCommitSHA(ref string) bool {
	if len(ref) < 7 || len(ref) > 40 {
		return false
	}
	for _, c := range ref {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}
