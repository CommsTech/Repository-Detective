package security

import (
	"regexp"
	"strings"
)

// RedactSecrets applies heuristic secret redaction for logs, UI-adjacent text, and exports.
// This is not a guarantee of compliance — administrators must still control access and retention.
var redactPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(password|api[_-]?key|secret|token|auth)\s*[:=]\s*["'][^"']{4,}["']`),
	regexp.MustCompile(`(?i)[?&]api_key=[^&\s"']+`),
	regexp.MustCompile(`(?i)X-Repository-Detective-API-Key:\s*\S+`),
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9\-._~+/]+=*`),
	regexp.MustCompile(`(?i)(ghp_|gho_|glpat-|xox[baprs]-)[A-Za-z0-9\-_]+`),
}

// RedactSecrets masks likely secret material in arbitrary text.
func RedactSecrets(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, pattern := range redactPatterns {
		value = pattern.ReplaceAllString(value, "[REDACTED]")
	}
	return value
}

// RedactLogField redacts and truncates a value safe for info/warn logs.
func RedactLogField(value string, maxLen int) string {
	value = RedactSecrets(value)
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.TrimSpace(value)
	if maxLen <= 0 {
		maxLen = 200
	}
	if len(value) > maxLen {
		return value[:maxLen] + "…"
	}
	return value
}

// RedactAccessLogLine masks credential material in HTTP access log fields.
func RedactAccessLogLine(value string) string {
	return RedactSecrets(value)
}
