package analyzers

import "strings"

// ClassifyScannerRunStatus maps raw scanner exit/output into operational classes.
func ClassifyScannerRunStatus(exitCode int, stdout, stderr string, timedOut bool) string {
	if timedOut {
		return "timed_out"
	}
	if exitCode != 0 && !looksLikeJSON(stdout) {
		if strings.Contains(strings.ToLower(stderr), "not found") ||
			strings.Contains(strings.ToLower(stderr), "no such file") {
			return "scanner_unavailable"
		}
		return "failed"
	}
	if !looksLikeJSON(stdout) && strings.TrimSpace(stdout) != "" {
		return "parse_failed"
	}
	if exitCode != 0 {
		return "failed"
	}
	return "success"
}

func looksLikeJSON(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")
}
