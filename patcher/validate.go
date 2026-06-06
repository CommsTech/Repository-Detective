package patcher

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"git.commsnet.org/commstech/bugbot/internal/security"
)

var forbiddenSubstrings = []string{
	"|", "&", ";", "`", "$", "(", ")", "<", ">", "\n", "\r",
	"npm install", "pip install", "curl", "wget", "bash -c", "sh -c",
	"make", "terraform apply", "docker build", "kubectl",
}

// ParseAllowedCommand splits a validation command into safe fixed argv.
func ParseAllowedCommand(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty command")
	}
	lower := strings.ToLower(raw)
	for _, bad := range forbiddenSubstrings {
		if strings.Contains(lower, bad) {
			return nil, fmt.Errorf("forbidden command fragment")
		}
	}

	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	switch parts[0] {
	case "go":
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid go command")
		}
		switch parts[1] {
		case "test", "vet":
			if len(parts) != 3 || !isSafeGoPackagePattern(parts[2]) {
				return nil, fmt.Errorf("only go test or go vet with safe package pattern allowed")
			}
			return parts, nil
		default:
			return nil, fmt.Errorf("unsupported go subcommand")
		}
	case "staticcheck":
		if len(parts) != 2 || !isSafeGoPackagePattern(parts[1]) {
			return nil, fmt.Errorf("only staticcheck with safe package pattern allowed")
		}
		return parts, nil
	case "hadolint":
		if len(parts) != 2 {
			return nil, fmt.Errorf("hadolint requires exactly one file path")
		}
		path := parts[1]
		if !isSafeRelativePath(path) {
			return nil, fmt.Errorf("unsafe hadolint path")
		}
		return parts, nil
	default:
		return nil, fmt.Errorf("command not allowlisted")
	}
}

func isSafeRelativePath(p string) bool {
	p = strings.TrimSpace(p)
	if p == "" || filepath.IsAbs(p) {
		return false
	}
	clean := filepath.Clean(p)
	if strings.HasPrefix(clean, "..") || strings.Contains(clean, "..") {
		return false
	}
	return true
}

func isSafeGoPackagePattern(pattern string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "./..." {
		return true
	}
	if !strings.HasPrefix(pattern, "./") || !strings.HasSuffix(pattern, "/...") {
		return false
	}
	base := strings.TrimSuffix(strings.TrimPrefix(pattern, "./"), "/...")
	if base == "" || strings.Contains(base, "..") {
		return false
	}
	for _, seg := range strings.Split(base, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
		for _, r := range seg {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
				continue
			}
			return false
		}
	}
	return true
}

// RunAllowedCommand executes an allowlisted command in workspaceDir.
func RunAllowedCommand(raw string, workspaceDir string, timeout time.Duration) TestResult {
	argv, err := ParseAllowedCommand(raw)
	if err != nil {
		return TestResult{Command: raw, Status: "rejected", Detail: err.Error()}
	}
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	out, runErr := runFixedCommand(argv, workspaceDir, timeout)
	detail := redactOutput(string(out))
	if runErr != nil {
		return TestResult{Command: raw, Status: "failed", Detail: detail}
	}
	return TestResult{Command: raw, Status: "passed", Detail: detail}
}

func runFixedCommand(argv []string, dir string, timeout time.Duration) ([]byte, error) {
	if security.SubprocessEnvExposesSecrets() {
		return nil, fmt.Errorf("unsafe subprocess environment")
	}
	return execFixedRunner(argv, dir, timeout)
}

var execFixedRunner = execFixed
