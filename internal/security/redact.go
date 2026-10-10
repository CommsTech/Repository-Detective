package security

import (
	"regexp"
	"strings"
)

// RedactSecrets applies heuristic secret redaction for logs, UI-adjacent text, and exports.
// This is not a guarantee of compliance — administrators must still control access and retention.
//
// Credential/secret redaction is intentional. Privacy-sensitive path minimization is separate
// (see MinimizeSensitivePath) and is not applied by default to operational diagnostics.
var redactPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(password|api[_-]?key|secret|token|auth)\s*[:=]\s*["'][^"']{4,}["']`),
	regexp.MustCompile(`(?i)(password|api[_-]?key|secret|token|auth)\s*[:=]\s*[^\s"'\\]{8,}`),
	regexp.MustCompile(`(?i)[?&](api[_-]?key|access_token|token|secret)=[^&\s"']+`),
	regexp.MustCompile(`(?i)X-Repository-Detective-API-Key:\s*\S+`),
	regexp.MustCompile(`(?i)Authorization:\s*Bearer\s+\S+`),
	regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9\-._~+/]+=*`),
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	regexp.MustCompile(`(?i)(ghp_|gho_|ghu_|ghs_|ghr_|glpat-|xox[baprs]-)[A-Za-z0-9\-_]+`),
	regexp.MustCompile(`(?i)sk-[A-Za-z0-9]{20,}`), // OpenAI-like
	regexp.MustCompile(`(?i)(REPOSITORY_DETECTIVE_|GITEA_|GITHUB_|OPENAI_|ANTHROPIC_)[A-Z0-9_]*?(TOKEN|KEY|SECRET)=[^\s]+`),
	regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----[\s\S]*?-----END (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`),
	// Credentials embedded in URLs: https://user:pass@host
	regexp.MustCompile(`(?i)(https?://)([^/\s:@]+):([^/\s@]+)@`),
}

// RedactSecrets masks likely secret material in arbitrary text.
func RedactSecrets(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, pattern := range redactPatterns {
		if strings.Contains(pattern.String(), "https?://") {
			value = pattern.ReplaceAllString(value, "${1}[REDACTED]:[REDACTED]@")
			continue
		}
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

// SanitizeDiagnostic is the central path for durable diagnostics (scanner stderr, Doctor
// details, support bundles, API error surfaces). It redacts credentials and truncates.
func SanitizeDiagnostic(value string, maxLen int) string {
	value = RedactSecrets(value)
	// Avoid recording environment dumps / shell export blocks wholesale.
	if strings.Contains(strings.ToUpper(value), "PRINTENV") ||
		strings.Contains(value, "declare -x ") {
		value = "[REDACTED environment dump]"
	}
	if maxLen <= 0 {
		maxLen = 4000
	}
	if len(value) > maxLen {
		return value[:maxLen] + "…[truncated]"
	}
	return value
}

// MinimizeSensitivePath optionally shortens home/user path prefixes for privacy-sensitive
// exports. Operational logs should prefer RedactLogField / SanitizeDiagnostic instead.
func MinimizeSensitivePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	// Windows user profile
	if re := regexp.MustCompile(`(?i)^[A-Z]:\\Users\\[^\\]+\\`); re.MatchString(path) {
		return re.ReplaceAllString(path, `C:\Users\[user]\`)
	}
	// Unix home
	if re := regexp.MustCompile(`(?i)^/home/[^/]+/`); re.MatchString(path) {
		return re.ReplaceAllString(path, `/home/[user]/`)
	}
	if strings.HasPrefix(path, "/Users/") {
		parts := strings.SplitN(path, "/", 4)
		if len(parts) >= 3 {
			rest := ""
			if len(parts) == 4 {
				rest = parts[3]
			}
			return "/Users/[user]/" + rest
		}
	}
	return path
}
