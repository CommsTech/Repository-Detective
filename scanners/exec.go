package scanners

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"git.commsnet.org/commstech/bugbot/internal/security"
)

// commandExitError wraps subprocess failures with timeout and output context.
type commandExitError struct {
	err     error
	timedOut bool
	output  []byte
}

func (e *commandExitError) Error() string {
	if e.timedOut {
		return fmt.Sprintf("command timed out: %v", e.err)
	}
	return e.err.Error()
}

func (e *commandExitError) Unwrap() error {
	return e.err
}

func runCommand(ctx context.Context, timeout time.Duration, dir string, name string, args ...string) ([]byte, error) {
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, name, args...)
	cmd.Dir = dir
	cmd.Env = security.MinimalSubprocessEnv()

	stdout := &cappedBuffer{limit: maxCommandOutputBytes}
	stderr := &cappedBuffer{limit: maxCommandOutputBytes}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	output := append(stdout.Bytes(), stderr.Bytes()...)
	if err == nil {
		return output, nil
	}

	timedOut := errors.Is(cmdCtx.Err(), context.DeadlineExceeded)
	return output, &commandExitError{err: err, timedOut: timedOut, output: output}
}

func commandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func normalizeSeverity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical", "crit":
		return "critical"
	case "high", "error":
		return "high"
	case "medium", "warning", "warn":
		return "medium"
	case "low", "info", "note":
		return "low"
	default:
		return "medium"
	}
}

func severityRank(severity string) int {
	switch normalizeSeverity(severity) {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	default:
		return 1
	}
}

func meetsMinSeverity(severity, minimum string) bool {
	return severityRank(severity) >= severityRank(minimum)
}
