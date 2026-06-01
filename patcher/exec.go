package patcher

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"git.commsnet.org/commstech/bugbot/internal/security"
)

func execFixed(argv []string, dir string, timeout time.Duration) ([]byte, error) {
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
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
