package preinstall

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"git.commsnet.org/commstech/bugbot/internal/security"
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

	parent, err := os.MkdirTemp("", "bugbot-preinstall-*")
	if err != nil {
		return CloneResult{}, fmt.Errorf("create temp dir: %w", err)
	}
	dest := filepath.Join(parent, "repo")
	cleanup := func() { _ = os.RemoveAll(parent) }

	if security.SubprocessEnvExposesSecrets() {
		cleanup()
		return CloneResult{}, fmt.Errorf("internal configuration would expose operator secrets in clone environment")
	}

	cloneArgs := []string{
		"clone", "--depth=1", "--single-branch", "--no-tags",
		"--", cloneURL, dest,
	}
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

	totalBytes, fileCount, err := measureWorkspace(dest, cfg.maxRepoSizeBytes(), cfg.MaxFiles)
	if err != nil {
		cleanup()
		return CloneResult{}, err
	}

	return CloneResult{
		WorkspaceDir:  dest,
		Cleanup:       cleanup,
		CommitSHA:     commitSHA,
		DefaultBranch: defaultBranch,
		TotalBytes:    totalBytes,
		FileCount:     fileCount,
	}, nil
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

func measureWorkspace(root string, maxBytes int64, maxFiles int) (int64, int, error) {
	if maxFiles <= 0 {
		maxFiles = 5000
	}
	var total int64
	count := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Type()&os.ModeSymlink != 0 {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		count++
		if count > maxFiles {
			return fmt.Errorf("repository exceeds max file count (%d)", maxFiles)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		if total > maxBytes {
			return fmt.Errorf("repository exceeds max size (%d MB)", maxBytes/(1024*1024))
		}
		return nil
	})
	return total, count, err
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
		"clone", "--depth=1", "--single-branch", "--no-tags",
		"--", cloneURL, dest,
	}
}
