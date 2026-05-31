package scanners

import (
	"os"
	"path/filepath"
	"strings"
)

const defaultIACScannerMaxFindings = 100

func iacScannerMaxFindings(cfg Config) int {
	if cfg.IACScannerMaxFindings > 0 {
		return cfg.IACScannerMaxFindings
	}
	return defaultIACScannerMaxFindings
}

// IsDockerfilePath reports whether a workspace-relative path looks like a Dockerfile.
func IsDockerfilePath(path string) bool {
	path = filepath.ToSlash(strings.ToLower(strings.TrimSpace(path)))
	if path == "" {
		return false
	}
	base := filepath.Base(path)
	if base == "dockerfile" {
		return true
	}
	if strings.HasSuffix(base, ".dockerfile") {
		return true
	}
	return strings.Contains(path, "/dockerfile")
}

// CollectDockerfilePaths returns absolute Dockerfile paths present in the workspace.
func CollectDockerfilePaths(dir string, entries []FileEntry) []string {
	seen := make(map[string]struct{})
	var paths []string
	add := func(rel string) {
		rel = filepath.ToSlash(strings.TrimSpace(rel))
		if rel == "" || !IsDockerfilePath(rel) {
			return
		}
		abs := filepath.Join(dir, filepath.FromSlash(rel))
		if _, ok := seen[abs]; ok {
			return
		}
		info, err := os.Stat(abs)
		if err != nil || info.IsDir() {
			return
		}
		seen[abs] = struct{}{}
		paths = append(paths, abs)
	}
	for _, entry := range entries {
		add(entry.Path)
	}
	if dir != "" {
		add("Dockerfile")
		add("docker/Dockerfile")
	}
	return paths
}

// WorkspaceHasDockerfiles reports whether hadolint should attempt a scan.
func WorkspaceHasDockerfiles(dir string, entries []FileEntry) bool {
	return len(CollectDockerfilePaths(dir, entries)) > 0
}

func isYAMLPath(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}

func isWorkflowPath(path string) bool {
	path = filepath.ToSlash(strings.ToLower(path))
	return strings.Contains(path, ".github/workflows/") ||
		strings.Contains(path, ".gitea/workflows/") ||
		strings.Contains(path, ".gitlab/ci/") ||
		strings.HasSuffix(path, ".gitlab-ci.yml")
}

func entryLooksLikeKubernetes(entry FileEntry) bool {
	if !isYAMLPath(entry.Path) {
		return false
	}
	path := filepath.ToSlash(strings.ToLower(entry.Path))
	if strings.Contains(path, "/k8s/") ||
		strings.Contains(path, "/kubernetes/") ||
		strings.Contains(path, "/manifests/") ||
		strings.Contains(path, "/deploy/") ||
		strings.Contains(path, "/helm/") ||
		strings.Contains(path, "/charts/") {
		return true
	}
	content := strings.ToLower(entry.Content)
	return strings.Contains(content, "apiversion:") && strings.Contains(content, "kind:")
}

func entryLooksLikeCloudFormation(entry FileEntry) bool {
	path := filepath.ToSlash(strings.ToLower(entry.Path))
	if strings.HasSuffix(path, ".template") || strings.Contains(path, "cloudformation") {
		return true
	}
	content := strings.ToLower(entry.Content)
	return strings.Contains(content, "awstemplateformatversion") ||
		strings.Contains(content, `"awstemplateformatversion"`)
}

func isCheckovTargetPath(path string, content string) bool {
	path = filepath.ToSlash(strings.ToLower(strings.TrimSpace(path)))
	if path == "" {
		return false
	}
	switch {
	case strings.HasSuffix(path, ".tf"):
	case strings.HasSuffix(path, ".tf.json"):
	case strings.HasSuffix(path, "chart.yaml"):
	case strings.HasSuffix(path, "docker-compose.yml"):
	case strings.HasSuffix(path, "docker-compose.yaml"):
	case strings.HasSuffix(path, "compose.yml"):
	case strings.HasSuffix(path, "compose.yaml"):
	case IsDockerfilePath(path):
	case isWorkflowPath(path):
	default:
		entry := FileEntry{Path: path, Content: content}
		if entryLooksLikeKubernetes(entry) {
			return true
		}
		if entryLooksLikeCloudFormation(entry) {
			return true
		}
		if strings.Contains(path, "/templates/") && isYAMLPath(path) {
			return true
		}
		return false
	}
	return true
}

// WorkspaceHasIaC reports whether checkov should attempt a scan.
func WorkspaceHasIaC(entries []FileEntry) bool {
	for _, entry := range entries {
		if isCheckovTargetPath(entry.Path, entry.Content) {
			return true
		}
	}
	return false
}
