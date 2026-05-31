package health

import (
	"path/filepath"
	"strings"
)

var defaultSkipDirs = []string{
	"node_modules/", "vendor/", ".git/", "dist/", "build/", "target/",
	"__pycache__/", ".venv/", "venv/",
}

// ShouldSkipPath reports whether a repo-relative path should be excluded from health checks.
func ShouldSkipPath(path string, extraPatterns []string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" {
		return true
	}
	lower := strings.ToLower(path)
	for _, dir := range defaultSkipDirs {
		if strings.Contains(lower, dir) {
			return true
		}
	}
	for _, pattern := range extraPatterns {
		if pattern != "" && strings.Contains(path, pattern) {
			return true
		}
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".pdf", ".zip", ".exe", ".dll", ".so", ".woff", ".ttf", ".min.js", ".min.css":
		return true
	}
	return false
}

func filterFiles(files []FileInput, extraPatterns []string) []FileInput {
	out := make([]FileInput, 0, len(files))
	for _, f := range files {
		if ShouldSkipPath(f.Path, extraPatterns) {
			continue
		}
		out = append(out, f)
	}
	return out
}

func filterPaths(paths []string, extraPatterns []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if !ShouldSkipPath(p, extraPatterns) {
			out = append(out, p)
		}
	}
	return out
}
