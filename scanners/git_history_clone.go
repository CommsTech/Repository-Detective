package scanners

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
// token authenticates the clone; without it private repositories fail with
// "Authentication failed" and their history is never scanned for secrets.
func PrepareGitHistoryWorkspace(ctx context.Context, cloneURL, token, ref string, maxCommits int, timeoutSeconds int) (GitHistoryWorkspace, error) {
	cloneURL = strings.TrimSpace(cloneURL)
	if cloneURL == "" {
		return GitHistoryWorkspace{}, fmt.Errorf("clone URL is required for git history secret scan")
	}
	authURL, err := authenticatedCloneURL(cloneURL, token)
	if err != nil {
		return GitHistoryWorkspace{}, err
	}
	if security.SubprocessEnvExposesSecrets() {
		return GitHistoryWorkspace{}, fmt.Errorf("unsafe subprocess environment for git clone")
	}

	parent, mkErr := os.MkdirTemp("", "rd-gitleaks-history-*")
	if mkErr != nil {
		return GitHistoryWorkspace{}, fmt.Errorf("create temp dir: %w", mkErr)
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
	cloneArgs = append(cloneArgs, authURL, dest)

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

var (
	// Matches credentials embedded in a remote URL, e.g. https://user:token@host/repo.
	gitCredentialURLPattern = regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.-]*)://[^\s/@]*@`)
	// Matches local scratch workspaces that must not leak into findings or logs.
	gitTempWorkspacePattern = regexp.MustCompile(`(?:/[^\s:'"]*)?/(?:rd-gitleaks-history|rd-archive|rd-scan|bugbot-archive)-[A-Za-z0-9_-]+\S*`)
)

// sanitizeHistoryGitError keeps git's own message so operators can tell auth failures
// from network or disk failures, while stripping credentials and local scratch paths.
func sanitizeHistoryGitError(out []byte) error {
	msg := redactScannerDetail(string(out))
	msg = gitCredentialURLPattern.ReplaceAllString(msg, "$1://***@")
	msg = gitTempWorkspacePattern.ReplaceAllString(msg, "<workspace>")
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return fmt.Errorf("git operation failed")
	}
	const maxMsg = 300
	if len(msg) > maxMsg {
		msg = msg[:maxMsg-3] + "..."
	}
	return fmt.Errorf("git operation failed: %s", msg)
}

// authenticatedCloneURL embeds token as HTTP basic credentials, matching how the
// remediation patcher clones. The token never reaches argv logs because clone
// output is redacted by sanitizeHistoryGitError before it is surfaced.
func authenticatedCloneURL(cloneURL, token string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return cloneURL, nil
	}
	// Only HTTP(S) remotes take basic credentials. scp-style remotes such as
	// git@host:owner/repo.git are not parseable as URLs and rely on SSH keys, so
	// they are handed back untouched rather than failing the scan.
	parsed, err := url.Parse(cloneURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return cloneURL, nil
	}
	parsed.User = url.UserPassword("oauth2", token)
	return parsed.String(), nil
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
