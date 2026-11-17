package runner

import (
	"git.commsnet.org/commstech/repository-detective/internal/security"
)

// RedactLogLine removes secret-like substrings from operator-facing log lines.
func RedactLogLine(msg string) string {
	return security.SanitizeDiagnostic(msg, 500)
}
