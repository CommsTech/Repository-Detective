package gitea

import "strings"

// DefaultLabelColor returns the Repository Detective palette color (hex, no #) for a label name.
func DefaultLabelColor(name string) string {
	key := strings.ToLower(strings.TrimSpace(name))
	switch key {
	case "source/repository-detective", "repository-detective":
		return "0d1b2a"
	case "automated-review":
		return "374151"
	case "category/security", "repository-detective/security":
		return "991b1b"
	case "category/secret", "repository-detective/secret":
		return "7c3aed"
	case "category/dependency", "repository-detective/dependency":
		return "2563eb"
	case "category/container":
		return "0ea5a4"
	case "category/reliability", "repository-detective/reliability":
		return "0891b2"
	case "category/maintainability", "repository-detective/maintainability", "repository-detective/code-quality":
		return "059669"
	case "triage/needs-review", "repository-detective/needs-human-review":
		return "d97706"
	case "repository-detective/open":
		return "0ea5a4"
	case "triage/confirmed":
		return "16a34a"
	case "triage/false-positive", "repository-detective/false-positive":
		return "6b7280"
	case "triage/accepted-risk", "repository-detective/suppressed":
		return "a16207"
	case "remediation/auto-pr":
		return "2563eb"
	case "remediation/manual":
		return "0ea5a4"
	case "remediation/blocked":
		return "dc2626"
	case "remediation/unknown":
		return "6b7280"
	case "severity/critical":
		return "dc2626"
	case "severity/high":
		return "ea580c"
	case "severity/medium":
		return "f59e0b"
	case "severity/low":
		return "3b82f6"
	case "severity/info":
		return "6b7280"
	default:
		if strings.HasPrefix(key, "scanner/") {
			return "4338ca"
		}
		if strings.HasPrefix(key, "severity/") {
			return "6b7280"
		}
		if strings.HasPrefix(key, "category/") {
			return "1e3a8a"
		}
		if strings.Contains(key, "security") || strings.Contains(key, "secret") {
			return "991b1b"
		}
		return "1e3a8a"
	}
}
