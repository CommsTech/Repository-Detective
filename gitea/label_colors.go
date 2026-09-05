package gitea

import "strings"

// DefaultLabelColor returns the Repository Detective palette color (hex, no #) for a label name.
func DefaultLabelColor(name string) string {
	key := strings.ToLower(strings.TrimSpace(name))
	switch key {
	case "repository-detective":
		return "0d1b2a"
	case "automated-review":
		return "374151"
	case "repository-detective/security":
		return "991b1b"
	case "repository-detective/secret":
		return "7c3aed"
	case "repository-detective/dependency":
		return "2563eb"
	case "repository-detective/code-quality":
		return "0ea5a4"
	case "repository-detective/reliability":
		return "0891b2"
	case "repository-detective/maintainability":
		return "059669"
	case "repository-detective/performance":
		return "0284c7"
	case "repository-detective/test-gap":
		return "6366f1"
	case "repository-detective/tech-debt":
		return "a16207"
	case "repository-detective/architecture":
		return "4338ca"
	case "repository-detective/ai-generated-risk":
		return "c026d3"
	case "repository-detective/open":
		return "0ea5a4"
	case "repository-detective/still-present":
		return "f59e0b"
	case "repository-detective/needs-human-review":
		return "d97706"
	case "repository-detective/not-reproduced":
		return "6b7280"
	case "repository-detective/resolved-verified":
		return "16a34a"
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
		if strings.HasPrefix(key, "severity/") {
			return "6b7280"
		}
		if strings.Contains(key, "security") || strings.Contains(key, "secret") {
			return "991b1b"
		}
		return "1e3a8a"
	}
}
