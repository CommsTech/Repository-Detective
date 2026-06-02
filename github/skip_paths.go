package github

import "strings"

func repositoryPathSkipped(path string) bool {
	p := strings.TrimSpace(path)
	if p == "" {
		return false
	}
	switch p {
	case "vendor", "node_modules", ".git":
		return true
	}
	if strings.HasPrefix(p, "vendor/") || strings.HasPrefix(p, "node_modules/") {
		return true
	}
	return false
}
