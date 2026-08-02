package preinstall

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

const maxGitOutputBytes = 256 << 10 // 256 KiB

// CloneResult contains metadata from a shallow git clone.
type CloneResult struct {
	WorkspaceDir  string
	Cleanup       func()
	CommitSHA     string
	DefaultBranch string
	TotalBytes    int64
	FileCount     int
	SandboxID     string
	Sandbox       SandboxMeta
}

// ShallowClone clones a public repository into an isolated temp directory.
func ShallowClone(ctx context.Context, parsed ParsedRepoURL, cfg Config) (CloneResult, error) {
	if !cfg.AllowGitClone {
		return CloneResult{}, fmt.Errorf("git clone is disabled by configuration")
	}
	if err := RevalidateHost(parsed.Host, cfg.AllowPrivateNetworks); err != nil {
		return CloneResult{}, err
	}
	cloneURL := strings.TrimSpace(parsed.CloneURL)
	if cloneURL == "" {
		return CloneResult{}, fmt.Errorf("clone URL is required")
	}

	sandboxID := newSandboxID()
	parent, err := os.MkdirTemp("", "rd-preinstall-"+sandboxID+"-*")
	if err != nil {
		return CloneResult{}, fmt.Errorf("create temp dir: %w", err)
	}
	dest := filepath.Join(parent, "repo")
	cleanup := func() { _ = os.RemoveAll(parent) }
	meta := sandboxMetaFromConfig(cfg, sandboxID, dest)

	if security.SubprocessEnvExposesSecrets() {
		cleanup()
		return CloneResult{}, fmt.Errorf("internal configuration would expose operator secrets in clone environment")
	}

	cloneArgs := []string{
		"-c", "core.hooksPath=/dev/null",
		"-c", "advice.detachedHead=false",
		"clone", "--depth=1", "--single-branch", "--no-tags",
	}
	if !cfg.SandboxAllowSubmodules {
		cloneArgs = append(cloneArgs, "--no-recurse-submodules")
	}
	cloneArgs = append(cloneArgs, "--", cloneURL, dest)
	if out, err := runGit(ctx, cloneArgs); err != nil {
		cleanup()
		return CloneResult{}, fmt.Errorf("git clone failed: %w", sanitizeGitError(out, err))
	}

	revOut, err := runGit(ctx, []string{"-C", dest, "rev-parse", "HEAD"})
	if err != nil {
		cleanup()
		return CloneResult{}, fmt.Errorf("git rev-parse: %w", err)
	}
	commitSHA := strings.TrimSpace(string(revOut))

	branchOut, _ := runGit(ctx, []string{"-C", dest, "rev-parse", "--abbrev-ref", "HEAD"})
	defaultBranch := strings.TrimSpace(string(branchOut))
	if defaultBranch == "" || defaultBranch == "HEAD" {
		defaultBranch = "main"
	}

	totalBytes, fileCount, err := measureWorkspaceSandbox(dest, cfg)
	if err != nil {
		cleanup()
		return CloneResult{}, err
	}
	if cfg.SandboxReadonlyWorkspace {
		if err := makeWorkspaceReadOnly(dest); err != nil {
			cleanup()
			return CloneResult{}, fmt.Errorf("make workspace read-only: %w", err)
		}
	}

	return CloneResult{
		WorkspaceDir:  dest,
		Cleanup:       cleanup,
		CommitSHA:     commitSHA,
		DefaultBranch: defaultBranch,
		TotalBytes:    totalBytes,
		FileCount:     fileCount,
		SandboxID:     sandboxID,
		Sandbox:       meta,
	}, nil
}

func newSandboxID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())[:12]
}

func makeWorkspaceReadOnly(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		return os.Chmod(path, 0o444)
	})
}

func runGit(ctx context.Context, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = security.MinimalSubprocessEnv()
	out, err := cmd.CombinedOutput()
	if len(out) > maxGitOutputBytes {
		out = out[:maxGitOutputBytes]
	}
	return out, err
}

func sanitizeGitError(out []byte, err error) error {
	_ = out
	return fmt.Errorf("git operation failed")
}


// SensitiveEnvKeys lists env vars that must never appear in audit workspaces.
var SensitiveEnvKeys = security.SensitiveEnvKeys

// CloneEnvExposesSecrets returns true if minimal git env would leak operator secrets.
func CloneEnvExposesSecrets() bool {
	return security.SubprocessEnvExposesSecrets()
}

// GitCloneArgsForTests exposes clone argv shape for security tests.
func GitCloneArgsForTests(cloneURL, dest string) []string {
	return []string{
		"-c", "core.hooksPath=/dev/null",
		"clone", "--depth=1", "--single-branch", "--no-tags", "--no-recurse-submodules",
		"--", cloneURL, dest,
	}
}
