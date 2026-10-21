package graph

import (
	"path/filepath"
	"strings"
)

var defaultSkipDirs = []string{
	"node_modules/", "vendor/", ".git/", "dist/", "build/", "target/",
	"__pycache__/", ".venv/", "venv/", "examples/", "example/", "docs/", "doc/",
}

// ShouldSkipPath reports whether a repo-relative path should be excluded from the graph.
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
	base := strings.ToLower(filepath.Base(path))
	if strings.HasSuffix(base, "_gen.go") || strings.Contains(lower, "/generated/") {
		return true
	}
	for _, pattern := range extraPatterns {
		if pattern != "" && strings.Contains(path, pattern) {
			return true
		}
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".pdf", ".zip", ".exe", ".dll", ".so", ".woff", ".ttf", ".min.js", ".min.css", ".lock", ".sum":
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
