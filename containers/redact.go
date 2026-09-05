package containers

import "git.commsnet.org/commstech/repository-detective/internal/security"

// RedactLogLine removes credential-like substrings from log/error text.
func RedactLogLine(s string) string {
	return security.SanitizeDiagnostic(s, 500)
}
