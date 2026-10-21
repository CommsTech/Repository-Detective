package scanners

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultGoScannerMaxFindings = 100

// HasGoModule reports whether the workspace root contains a go.mod file.
func HasGoModule(dir string) bool {
	if dir == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil && !info.IsDir()
}

// HasGoFiles reports whether any workspace entry is a Go source file.
func HasGoFiles(entries []FileEntry) bool {
	for _, entry := range entries {
		if strings.HasSuffix(strings.ToLower(entry.Path), ".go") {
			return true
		}
	}
	return false
}

// WorkspaceHasGo reports whether govulncheck/staticcheck/gosec should attempt a scan.
func WorkspaceHasGo(dir string, entries []FileEntry) bool {
	return HasGoModule(dir) || HasGoFiles(entries)
}

func goScannerMaxFindings(cfg Config) int {
	if cfg.GoScannerMaxFindings > 0 {
		return cfg.GoScannerMaxFindings
	}
	return defaultGoScannerMaxFindings
}

type cappedFindings struct {
	Findings  []Finding
	Total     int
	Truncated bool
}

func capFindings(findings []Finding, max int) cappedFindings {
	if max <= 0 {
		max = defaultGoScannerMaxFindings
	}
	total := len(findings)
	if total <= max {
		return cappedFindings{Findings: findings, Total: total}
	}
	return cappedFindings{
		Findings:  append([]Finding(nil), findings[:max]...),
		Total:     total,
		Truncated: true,
	}
}

func truncateDetailForScanner(scanner string, max, total int) string {
	if total <= max {
		return ""
	}
	if scanner != "" {
		return truncateDetailLabel(scanner, max, total)
	}
	return truncateDetailLabel("findings", max, total)
}

func truncateDetailLabel(label string, max, total int) string {
	return "truncated to " + itoa(max) + " " + label + " (" + itoa(total) + " total)"
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	n := v
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func scannerTimeoutSeconds(specific, fallback int) time.Duration {
	seconds := specific
	if seconds <= 0 {
		seconds = fallback
	}
	if seconds <= 0 {
		seconds = 120
	}
	return time.Duration(seconds) * time.Second
}
