package profile

import (
	"path/filepath"
	"strings"
)

var ignoreDirSegments = []string{
	"vendor", "node_modules", ".git", ".venv", "venv", "__pycache__",
	"build", "dist", "target", ".terraform", "coverage", ".cache",
	".next", ".nuxt", "out", "bin", "obj",
}

var generatedPathHints = []string{
	"/generated/", "/gen/", "/.generated/", "/autogen/", "/openapi/",
	"/swagger/", "/pb/", "/proto/gen/", "/vendor/",
}

var vendorPathHints = []string{
	"/vendor/", "/node_modules/", "/third_party/",
}

var testPathHints = []string{
	"/test/", "/tests/", "/__tests__/", "/testdata/", "/fixtures/",
	"/mock/", "/mocks/", "/spec/", "/specs/",
	"/benchmark/fixture/", "/benchmark/",
}

var examplePathHints = []string{
	"/examples/", "/example/", "/samples/", "/sample/", "/demo/",
}

var docsPathHints = []string{
	"/docs/", "/doc/", "/documentation/", "/website/", "/site/",
}

var minifiedExtensions = map[string]bool{
	".min.js": true, ".min.css": true,
}

var lockFileNames = map[string]bool{
	"package-lock.json": true, "yarn.lock": true, "pnpm-lock.yaml": true,
	"poetry.lock": true, "go.sum": true, "Cargo.lock": true,
	"Gemfile.lock": true, "composer.lock": true,
}

// NormalizePath returns a stable forward-slash path without leading ./ .
func NormalizePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.TrimPrefix(path, "./")
	return filepath.ToSlash(path)
}

// IsIgnoredPath reports whether a path should be excluded from analysis.
func IsIgnoredPath(path string) bool {
	norm := "/" + NormalizePath(path) + "/"
	for _, seg := range ignoreDirSegments {
		if strings.Contains(norm, "/"+seg+"/") {
			return true
		}
	}
	return false
}

// ClassifySourceType infers how a finding path should be treated.
func ClassifySourceType(path string) string {
	norm := NormalizePath(path)
	lower := strings.ToLower("/" + norm + "/")
	base := strings.ToLower(filepath.Base(norm))

	if IsIgnoredPath(norm) {
		if strings.Contains(lower, "/vendor/") || strings.Contains(lower, "/node_modules/") {
			return SourceTypeVendor
		}
		return SourceTypeGenerated
	}

	for _, hint := range vendorPathHints {
		if strings.Contains(lower, hint) {
			return SourceTypeVendor
		}
	}
	for _, hint := range generatedPathHints {
		if strings.Contains(lower, hint) {
			return SourceTypeGenerated
		}
	}
	if strings.HasSuffix(base, ".pb.go") || strings.HasSuffix(base, "_generated.go") {
		return SourceTypeGenerated
	}
	if isMinifiedFile(base) {
		return SourceTypeGenerated
	}
	if lockFileNames[base] {
		return SourceTypeDependency
	}

	for _, hint := range testPathHints {
		if strings.Contains(lower, hint) {
			return SourceTypeTest
		}
	}
	if strings.HasSuffix(base, "_test.go") || strings.HasSuffix(base, ".test.js") ||
		strings.HasSuffix(base, ".test.ts") || strings.HasSuffix(base, "_test.py") ||
		strings.HasSuffix(base, ".spec.js") || strings.HasSuffix(base, ".spec.ts") ||
		strings.HasSuffix(base, ".go.src") {
		return SourceTypeTest
	}

	for _, hint := range examplePathHints {
		if strings.Contains(lower, hint) {
			return SourceTypeExample
		}
	}

	for _, hint := range docsPathHints {
		if strings.Contains(lower, hint) {
			return SourceTypeDocs
		}
	}
	if strings.HasSuffix(base, ".md") || strings.HasSuffix(base, ".rst") || strings.HasSuffix(base, ".adoc") {
		return SourceTypeDocs
	}

	switch base {
	case "dockerfile", "docker-compose.yml", "docker-compose.yaml",
		"makefile", "taskfile.yml", "taskfile.yaml",
		"go.mod", "go.sum", "package.json", "pyproject.toml",
		"requirements.txt", "chart.yaml", "kustomization.yaml":
		return SourceTypeConfig
	}
	ext := strings.ToLower(filepath.Ext(norm))
	switch ext {
	case ".yaml", ".yml", ".toml", ".json", ".env", ".ini", ".cfg", ".conf":
		if strings.Contains(lower, "/.github/") || strings.Contains(lower, "/.gitea/") {
			return SourceTypeConfig
		}
	}

	return SourceTypeSource
}

func isMinifiedFile(base string) bool {
	if minifiedExtensions[base] {
		return true
	}
	if strings.Contains(base, ".min.") {
		return true
	}
	return false
}

// KnownManifestBasenames lists dependency/build manifest files.
var KnownManifestBasenames = map[string]bool{
	"go.mod":              true,
	"package.json":        true,
	"pyproject.toml":      true,
	"requirements.txt":    true,
	"dockerfile":          true,
	"docker-compose.yml":  true,
	"docker-compose.yaml": true,
	"chart.yaml":          true,
	"kustomization.yaml":  true,
	"makefile":            true,
	"taskfile.yml":        true,
	"taskfile.yaml":       true,
}
