package containers

import (
	"regexp"
	"strings"
)

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(password|token|secret|api[_-]?key)\s*[:=]\s*\S+`),
	regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9._\-+/=]+`),
}

// RedactLogLine removes credential-like substrings from log/error text.
func RedactLogLine(s string) string {
	out := s
	for _, re := range secretPatterns {
		out = re.ReplaceAllString(out, "[REDACTED]")
	}
	out = strings.ReplaceAll(out, "REGISTRY_AUTH_FILE=", "REGISTRY_AUTH_FILE=[REDACTED]")
	return out
}
