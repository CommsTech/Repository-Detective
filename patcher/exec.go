package patcher

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/internal/security"
)

func execFixed(argv []string, dir string, timeout time.Duration) ([]byte, error) {
	if err := validateArgv(argv); err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd, err := fixedCommand(ctx, argv)
	if err != nil {
		return nil, err
	}
	cmd.Dir = dir
	cmd.Env = security.MinimalSubprocessEnv()
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return out, fmt.Errorf("execFixed timed out after %v: %s", timeout, argv[0])
	}
	if len(out) > 256<<10 {
		out = out[:256<<10]
	}
	return out, err
}

func fixedCommand(ctx context.Context, argv []string) (*exec.Cmd, error) {
	switch argv[0] {
	case "go":
		if len(argv) < 3 {
			return nil, fmt.Errorf("invalid go command")
		}
		return exec.CommandContext(ctx, "go", argv[1], argv[2]), nil
	case "staticcheck":
		if len(argv) != 2 {
			return nil, fmt.Errorf("invalid staticcheck command")
		}
		return exec.CommandContext(ctx, "staticcheck", argv[1]), nil
	case "hadolint":
		if len(argv) != 2 {
			return nil, fmt.Errorf("invalid hadolint command")
		}
		return exec.CommandContext(ctx, "hadolint", argv[1]), nil
	default:
		return nil, fmt.Errorf("command not allowlisted")
	}
}

func redactOutput(s string) string {
	lower := strings.ToLower(s)
	for _, token := range []string{"token", "password", "secret", "api_key", "authorization"} {
		if strings.Contains(lower, token) {
			return "[redacted output]"
		}
	}
	if len(s) > 2048 {
		return s[:2048] + "...(truncated)"
	}
	return s
}
