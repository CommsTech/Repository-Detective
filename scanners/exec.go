package scanners

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/internal/security"
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
	stdout, _, err := runCommandStreams(ctx, timeout, dir, name, args...)
	return stdout, err
}

// runCommandStreams executes a subprocess and keeps stdout/stderr separate.
// Parsers should use stdout (JSON tools write there). On failure, stdout still
// prefers clean parseable bytes; the error wraps merged output for diagnostics.
func runCommandStreams(ctx context.Context, timeout time.Duration, dir string, name string, args ...string) (stdoutOut, stderrOut []byte, err error) {
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, name, args...)
	cmd.Dir = dir
	cmd.Env = security.MinimalSubprocessEnv()

	stdoutBuf := &cappedBuffer{limit: maxCommandOutputBytes}
	stderrBuf := &cappedBuffer{limit: maxCommandOutputBytes}
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	runErr := cmd.Run()
	stdoutOut = stdoutBuf.Bytes()
	stderrOut = stderrBuf.Bytes()
	merged := append(append([]byte{}, stdoutOut...), stderrOut...)
	if runErr == nil {
		// Prefer stdout for JSON/NDJSON parsers; fall back to stderr only if empty.
		if len(bytes.TrimSpace(stdoutOut)) > 0 {
			return stdoutOut, stderrOut, nil
		}
		return stderrOut, stderrOut, nil
	}

	timedOut := errors.Is(cmdCtx.Err(), context.DeadlineExceeded)
	parseBytes := stdoutOut
	if len(bytes.TrimSpace(parseBytes)) == 0 {
		parseBytes = merged
	}
	return parseBytes, stderrOut, &commandExitError{err: runErr, timedOut: timedOut, output: merged}
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
