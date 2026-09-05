package patcher

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

const maxGitOutputBytes = 256 << 10

var branchSafeChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// Workspace holds a temporary git checkout.
type Workspace struct {
	Dir     string
	Cleanup func()
	BaseSHA string
}

// PrepareWorkspace shallow-clones cloneURL into a temp directory.
func PrepareWorkspace(ctx context.Context, cloneURL, token, baseRef string) (Workspace, error) {
	if security.SubprocessEnvExposesSecrets() {
		return Workspace{}, fmt.Errorf("unsafe subprocess environment")
	}
	parent, err := os.MkdirTemp("", "repository-detective-pr-*")
	if err != nil {
		return Workspace{}, fmt.Errorf("create temp dir: %w", err)
	}
	dest := filepath.Join(parent, "repo")
	cleanup := func() { _ = os.RemoveAll(parent) }

	authURL, err := tokenizedCloneURL(cloneURL, token)
	if err != nil {
		cleanup()
		return Workspace{}, err
	}

	args := []string{"clone", "--depth=1", "--single-branch", "--no-tags", "--", authURL, dest}
	if out, err := runGit(ctx, args); err != nil {
		cleanup()
		return Workspace{}, fmt.Errorf("git clone failed: %w", sanitizeGitError(string(out), err))
	}

	if baseRef != "" && baseRef != "HEAD" {
		if out, err := runGit(ctx, []string{"-C", dest, "fetch", "origin", baseRef+":"+baseRef, "--depth=1"}); err != nil {
			_ = out
			if out2, err2 := runGit(ctx, []string{"-C", dest, "checkout", baseRef}); err2 != nil {
				cleanup()
				return Workspace{}, fmt.Errorf("checkout base ref: %w", sanitizeGitError(string(out2), err2))
			}
		} else {
			if out2, err2 := runGit(ctx, []string{"-C", dest, "checkout", baseRef}); err2 != nil {
				cleanup()
				return Workspace{}, fmt.Errorf("checkout base ref: %w", sanitizeGitError(string(out2), err2))
			}
		}
	}

	revOut, err := runGit(ctx, []string{"-C", dest, "rev-parse", "HEAD"})
	if err != nil {
		cleanup()
		return Workspace{}, fmt.Errorf("git rev-parse: %w", err)
	}
	return Workspace{
		Dir:     dest,
		Cleanup: cleanup,
		BaseSHA: strings.TrimSpace(string(revOut)),
	}, nil
}

// BranchName builds a safe remediation branch name.
func BranchName(prefix, fingerprint string) string {
	short := fingerprint
	if len(short) > 12 {
		short = short[:12]
	}
	short = branchSafeChars.ReplaceAllString(short, "")
	if short == "" {
		short = "fix"
	}
	prefix = strings.Trim(prefix, "/")
	return prefix + "/" + short
}

// CreateBranch creates and checks out branchName in workspace.
func CreateBranch(ctx context.Context, workspaceDir, branchName string) error {
	if branchName == "" {
		return fmt.Errorf("branch name required")
	}
	if out, err := runGit(ctx, []string{"-C", workspaceDir, "checkout", "-b", branchName}); err != nil {
		return fmt.Errorf("create branch: %w", sanitizeGitError(string(out), err))
	}
	return nil
}

// CommitAll stages all changes and commits with message.
func CommitAll(ctx context.Context, workspaceDir, message string) (string, error) {
	if err := ensureLocalGitIdentity(ctx, workspaceDir); err != nil {
		return "", err
	}
	if out, err := runGit(ctx, []string{"-C", workspaceDir, "add", "-A"}); err != nil {
		return "", fmt.Errorf("git add: %w", sanitizeGitError(string(out), err))
	}
	if out, err := runGit(ctx, []string{"-C", workspaceDir, "commit", "-m", message}); err != nil {
		return "", fmt.Errorf("git commit: %w", sanitizeGitError(string(out), err))
	}
	out, err := runGit(ctx, []string{"-C", workspaceDir, "rev-parse", "HEAD"})
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// PushBranch pushes branchName to origin using token auth.
func PushBranch(ctx context.Context, cloneURL, token, workspaceDir, branchName string) error {
	authURL, err := tokenizedCloneURL(cloneURL, token)
	if err != nil {
		return err
	}
	if out, err := runGit(ctx, []string{"-C", workspaceDir, "push", authURL, branchName+":"+branchName}); err != nil {
		return fmt.Errorf("git push: %w", sanitizeGitError(string(out), err))
	}
	return nil
}

func ensureLocalGitIdentity(ctx context.Context, workspaceDir string) error {
	for _, args := range [][]string{
		{"-C", workspaceDir, "config", "user.name", "Repository Detective"},
		{"-C", workspaceDir, "config", "user.email", "repository-detective@noreply.invalid"},
	} {
		if out, err := runGit(ctx, args); err != nil {
			return fmt.Errorf("git config: %w", sanitizeGitError(string(out), err))
		}
	}
	return nil
}

func tokenizedCloneURL(cloneURL, token string) (string, error) {
	cloneURL = strings.TrimSpace(cloneURL)
	if cloneURL == "" {
		return "", fmt.Errorf("clone URL required")
	}
	u, err := url.Parse(cloneURL)
	if err != nil {
		return "", fmt.Errorf("parse clone URL: %w", err)
	}
	if token != "" {
		u.User = url.UserPassword("oauth2", token)
	}
	return u.String(), nil
}

func runGit(ctx context.Context, args []string) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = security.MinimalSubprocessEnv()
	out, err := cmd.CombinedOutput()
	if len(out) > maxGitOutputBytes {
		out = out[:maxGitOutputBytes]
	}
	return out, err
}

func sanitizeGitError(out string, err error) error {
	if err == nil {
		return nil
	}
	out = redactTokenFromGitOutput(out)
	if out == "" {
		return err
	}
	return fmt.Errorf("%v: %s", err, redactTokenFromGitOutput(out))
}

func redactTokenFromGitOutput(s string) string {
	s = regexp.MustCompile(`oauth2:[^@\s]+@`).ReplaceAllString(s, "oauth2:***@")
	s = regexp.MustCompile(`://[^@\s]+@`).ReplaceAllString(s, "://***@")
	return redactOutput(s)
}

// EnsureDefaultBranchNotTarget rejects attempts to push directly to default branch.
func EnsureDefaultBranchNotTarget(branchName, defaultBranch string) error {
	if branchName == "" || defaultBranch == "" {
		return nil
	}
	if branchName == defaultBranch {
		return fmt.Errorf("refusing to push to default branch")
	}
	return nil
}

// GitTimeout returns a conservative timeout for git operations.
func GitTimeout() time.Duration {
	return 5 * time.Minute
}
