package patcher

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"git.commsnet.org/commstech/bugbot/internal/security"
)

func execFixed(argv []string, dir string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = security.MinimalSubprocessEnv()
	out, err := cmd.CombinedOutput()
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
