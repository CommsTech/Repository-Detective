package runner

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

// CloneRepository performs a shallow single-branch clone into dest.
func CloneRepository(ctx context.Context, cloneURL, ref, dest string, timeout time.Duration) error {
	cloneURL = strings.TrimSpace(cloneURL)
	if cloneURL == "" {
		return fmt.Errorf("clone URL required")
	}
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
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
	args = append(args, cloneURL, dest)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = security.MinimalSubprocessEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %s", RedactLogLine(string(out)))
	}
	return nil
}
