package preinstall

import (
	"path/filepath"
	"strings"
)

// skipPreinstallPath excludes benchmark fixtures, vendor/minified, and placeholder paths from install-blocking findings.
func skipPreinstallPath(path string) bool {
	p := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(path), "\\", "/"))
	base := strings.ToLower(filepath.Base(p))
	switch {
	case strings.Contains(p, "/benchmark/fixture/"), strings.HasPrefix(p, "benchmark/fixture/"):
		return true
	case strings.HasSuffix(base, ".go.src"), strings.HasSuffix(base, "_test.go"), strings.Contains(p, "/testdata/"):
		return true
	case strings.Contains(p, "/vendor/"), strings.HasPrefix(p, "vendor/"), strings.Contains(p, "node_modules/"), strings.Contains(base, ".min."):
		return true
	case strings.Contains(p, "/fixtures/"), strings.Contains(p, "/examples/"), strings.Contains(p, "/example/"):
		return true
	}
	return false
}

// isEnvFallbackPattern reports env-var fallback templates that are not hardcoded secrets.
func isEnvFallbackPattern(title, evidence string) bool {
	combined := strings.ToLower(title + " " + evidence)
	if strings.Contains(combined, "getenv") || strings.Contains(combined, "os.environ") {
		return true
	}
	if strings.Contains(combined, "${") && strings.Contains(combined, ":-}") {
		return true
	}
	return false
}
