package handlers

import (
	"strings"
)

// RepoAllowed returns true when a repository full name passes include/exclude filters.
func RepoAllowed(fullName string, includePatterns, excludePatterns []string) bool {
	fullName = strings.ToLower(strings.TrimSpace(fullName))
	if fullName == "" {
		return false
	}

	for _, pattern := range excludePatterns {
		if matchRepoPattern(fullName, pattern) {
			return false
		}
	}

	if len(includePatterns) == 0 {
		return true
	}

	for _, pattern := range includePatterns {
		if matchRepoPattern(fullName, pattern) {
			return true
		}
	}
	return false
}

func matchRepoPattern(fullName, pattern string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	if pattern == "" {
		return false
	}
	if pattern == fullName {
		return true
	}

	owner, repo := fullName, ""
	if parts := strings.SplitN(fullName, "/", 2); len(parts) == 2 {
		owner, repo = parts[0], parts[1]
	}

	targets := []string{fullName}
	if repo != "" {
		targets = append(targets, repo, owner+"/"+repo)
	}

	for _, target := range targets {
		if simpleGlobMatch(target, pattern) {
			return true
		}
	}
	return false
}

func simpleGlobMatch(value, pattern string) bool {
	if pattern == value {
		return true
	}
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(value, strings.TrimPrefix(pattern, "*")) {
		return true
	}
	if strings.HasSuffix(pattern, "*") && strings.HasPrefix(value, strings.TrimSuffix(pattern, "*")) {
		return true
	}
	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			return strings.HasPrefix(value, parts[0]) && strings.HasSuffix(value, parts[1])
		}
	}
	return strings.Contains(value, pattern)
}
