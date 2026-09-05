package runner

import (
	"regexp"
	"strings"

	"git.commsnet.org/commstech/repository-detective/redact"
)

var logSecretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(REPOSITORY_DETECTIVE_|GITEA_)[A-Z0-9_]*TOKEN=[^\s]+`),
	regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password)\s*[:=]\s*\S+`),
}

// RedactLogLine removes secret-like substrings from operator-facing log lines.
func RedactLogLine(msg string) string {
	msg = strings.TrimSpace(redact.SecretEvidence(msg))
	for _, re := range logSecretPatterns {
		msg = re.ReplaceAllString(msg, "[REDACTED]")
	}
	if len(msg) > 500 {
		msg = msg[:500] + "…"
	}
	return msg
}
