package runner

import (
	"bytes"
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

// CloneRepository performs a shallow single-branch clone into dest.
func CloneRepository(ctx context.Context, cloneURL, ref, dest string, timeout time.Duration) error {
	return CloneRepositoryAtCommit(ctx, cloneURL, ref, "", dest, timeout)
}

// CloneRepositoryAtCommit clones a branch and refuses to scan a moving ref if
// it did not resolve to the exact commit approved by the core scheduler.
// The supported transport set excludes git's command-executing ext protocol.
var pinnedCommitRE = regexp.MustCompile(`(?i)^[a-f0-9]{40}([a-f0-9]{24})?$`)

func CloneRepositoryAtCommit(ctx context.Context, cloneURL, ref, commitSHA, dest string, timeout time.Duration) error {
	cloneURL = strings.TrimSpace(cloneURL)
	if err := validateCloneTarget(cloneURL); err != nil {
		return err
	}
	if commitSHA != "" && !pinnedCommitRE.MatchString(commitSHA) {
		return fmt.Errorf("invalid pinned commit SHA")
	}
	if strings.HasPrefix(ref, "-") || strings.ContainsAny(ref, "\x00\n\r") {
		return fmt.Errorf("unsafe branch ref")
	}
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		return err
	}
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{
		"clone", "--depth", "1", "--single-branch", "--no-tags",
	}
	if branch := strings.TrimSpace(ref); branch != "" && branch != "HEAD" {
		args = append(args, "--branch", branch)
	}
	args = append(args, "--", cloneURL, dest)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(security.MinimalSubprocessEnv(), "GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_NO_REPLACE_OBJECTS=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %s", RedactLogLine(string(out)))
	}
	if commitSHA != "" {
		verify := exec.CommandContext(ctx, "git", "-C", dest, "rev-parse", "HEAD")
		verify.Env = cmd.Env
		actual, err := verify.Output()
		if err != nil || !bytes.EqualFold(bytes.TrimSpace(actual), []byte(commitSHA)) {
			return fmt.Errorf("scan aborted: checkout does not match pinned commit (ref moved or stale): expected %s", commitSHA)
		}
	}
	return nil
}

func validateCloneTarget(raw string) error {
	if raw == "" || strings.ContainsAny(raw, "\x00\n\r") || strings.HasPrefix(raw, "-") {
		return fmt.Errorf("unsafe clone target")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("unsafe clone URL: %w", err)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "ssh" {
		return fmt.Errorf("clone URL requires https or ssh, got %q", parsed.Scheme)
	}
	if parsed.Hostname() == "" || parsed.User != nil && parsed.User.Username() == "" {
		return fmt.Errorf("clone URL must have a host")
	}
	if parsed.User != nil {
		if _, hasPassword := parsed.User.Password(); hasPassword {
			return fmt.Errorf("embedded clone credentials are not permitted")
		}
	}
	return nil
}
