package notify

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const maxMessageLen = 3500
const maxTitleLen = 200
const maxSummaryLen = 500

var (
	secretPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password|bearer)\s*[:=]\s*\S+`),
		regexp.MustCompile(`(?i)-----BEGIN [A-Z ]+-----`),
		regexp.MustCompile(`(?i)(ghp_|gho_|glpat-|xox[baprs]-)[A-Za-z0-9\-_]+`),
		regexp.MustCompile(`(?i)https?://[^\s]*@(api\.telegram\.org|hooks\.slack\.com|discord\.com/api/webhooks)[^\s]*`),
	}
)

// SanitizeText removes secrets and truncates user-facing notification text.
func SanitizeText(s string) string {
	s = strings.TrimSpace(s)
	for _, re := range secretPatterns {
		s = re.ReplaceAllString(s, "[redacted]")
	}
	s = strings.ReplaceAll(s, "\x00", "")
	if utf8.RuneCountInString(s) > maxMessageLen {
		s = truncateRunes(s, maxMessageLen) + "…"
	}
	return s
}

// SanitizeTitle sanitizes a short title line.
func SanitizeTitle(s string) string {
	s = SanitizeText(s)
	if utf8.RuneCountInString(s) > maxTitleLen {
		s = truncateRunes(s, maxTitleLen) + "…"
	}
	return s
}

// SanitizeSummary sanitizes a summary line.
func SanitizeSummary(s string) string {
	s = SanitizeText(s)
	if utf8.RuneCountInString(s) > maxSummaryLen {
		s = truncateRunes(s, maxSummaryLen) + "…"
	}
	return s
}

// FormatMessage builds a concise plain-text notification body.
func FormatMessage(ev Event) Message {
	title := SanitizeTitle(ev.Title)
	summary := SanitizeSummary(ev.Summary)
	repo := SanitizeText(ev.Repository)

	var b strings.Builder
	b.WriteString("Repository Detective\n")
	if ev.Type != "" {
		b.WriteString("Event: ")
		b.WriteString(ev.Type)
		b.WriteByte('\n')
	}
	if repo != "" {
		b.WriteString("Repo: ")
		b.WriteString(repo)
		b.WriteByte('\n')
	}
	if ev.ScanID != "" {
		b.WriteString("Scan: ")
		b.WriteString(SanitizeText(ev.ScanID))
		b.WriteByte('\n')
	}
	if ev.Severity != "" {
		b.WriteString("Severity: ")
		b.WriteString(strings.ToLower(ev.Severity))
		b.WriteByte('\n')
	}
	if title != "" {
		b.WriteString("Title: ")
		b.WriteString(title)
		b.WriteByte('\n')
	}
	if summary != "" {
		b.WriteString("Summary: ")
		b.WriteString(summary)
		b.WriteByte('\n')
	}
	if len(ev.Counts) > 0 {
		b.WriteString("Counts: ")
		first := true
		for k, v := range ev.Counts {
			if !first {
				b.WriteString(", ")
			}
			first = false
			fmt.Fprintf(&b, "%s=%d", SanitizeText(k), v)
		}
		b.WriteByte('\n')
	}
	if ev.URL != "" {
		b.WriteString("Link: ")
		b.WriteString(SanitizeText(ev.URL))
		b.WriteByte('\n')
	}
	return Message{Text: strings.TrimSpace(b.String()), Event: ev}
}

// SafeRepoLabel returns owner/name from a full name; host-only for audit URLs when needed.
func SafeRepoLabel(fullName string) string {
	return SanitizeText(fullName)
}

// SafeAuditHost extracts host from HTTPS audit URL without path secrets.
func SafeAuditHost(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	rawURL = strings.TrimPrefix(rawURL, "https://")
	rawURL = strings.TrimPrefix(rawURL, "http://")
	if idx := strings.Index(rawURL, "/"); idx >= 0 {
		rawURL = rawURL[:idx]
	}
	if idx := strings.Index(rawURL, "@"); idx >= 0 {
		rawURL = rawURL[idx+1:]
	}
	return SanitizeText(rawURL)
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	n := 0
	for i := range s {
		if n == max {
			return s[:i]
		}
		n++
	}
	return s
}

// SeverityRank orders severities for threshold comparison.
func SeverityRank(severity string) int {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical", "crit":
		return 5
	case "high", "error":
		return 4
	case "medium", "warning", "warn":
		return 3
	case "low", "info", "note":
		return 2
	default:
		return 0
	}
}

// PassesSeverityThreshold reports whether event severity meets the configured minimum.
func PassesSeverityThreshold(severity, minSeverity string) bool {
	if minSeverity == "" {
		minSeverity = "high"
	}
	if severity == "" {
		return true
	}
	return SeverityRank(severity) >= SeverityRank(minSeverity)
}

// FindingSeverityWording returns safe wording for security findings.
func FindingSeverityWording(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "Critical severity finding detected"
	case "high":
		return "High severity finding detected"
	case "medium":
		return "Medium severity finding detected"
	default:
		return "Finding detected"
	}
}
