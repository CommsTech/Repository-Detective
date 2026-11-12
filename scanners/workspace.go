package scanners

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ErrUnsafeWorkspacePath is returned when a repo-relative path cannot be written safely.
var ErrUnsafeWorkspacePath = fmt.Errorf("unsafe workspace path")

var windowsDrivePath = regexp.MustCompile(`^[a-zA-Z]:`)

// FileEntry is a file written into a scan workspace.
type FileEntry struct {
	Path    string
	Content string
}

// dependencyManifests are always fetched when present so CVE scanners work on scoped pushes.
var dependencyManifests = []string{
	"go.mod",
	"go.sum",
	"package.json",
	"package-lock.json",
	"yarn.lock",
	"pnpm-lock.yaml",
	"requirements.txt",
	"requirements-dev.txt",
	"Pipfile",
	"Pipfile.lock",
	"poetry.lock",
	"pyproject.toml",
	"Cargo.toml",
	"Cargo.lock",
	"Gemfile",
	"Gemfile.lock",
	"composer.json",
	"composer.lock",
	"pom.xml",
	"build.gradle",
	"build.gradle.kts",
	"Dockerfile",
	"docker-compose.yml",
	"docker-compose.yaml",
}

// ManifestPaths returns standard dependency/config paths to fetch alongside changed files.
func ManifestPaths() []string {
	out := make([]string, len(dependencyManifests))
	copy(out, dependencyManifests)
	return out
}

// ValidateWorkspacePath ensures relPath stays inside workspaceRoot after cleaning.
func ValidateWorkspacePath(workspaceRoot, relPath string) (string, error) {
	if strings.TrimSpace(relPath) == "" {
		return "", fmt.Errorf("%w: empty path", ErrUnsafeWorkspacePath)
	}

	normalized := filepath.ToSlash(strings.TrimSpace(relPath))
	normalized = strings.ReplaceAll(normalized, "\\", "/")
	if filepath.IsAbs(normalized) || strings.HasPrefix(normalized, "/") {
		return "", fmt.Errorf("%w: absolute path %q", ErrUnsafeWorkspacePath, relPath)
	}
	if windowsDrivePath.MatchString(normalized) {
		return "", fmt.Errorf("%w: drive path %q", ErrUnsafeWorkspacePath, relPath)
	}
	if strings.Contains(normalized, "..") {
		return "", fmt.Errorf("%w: parent segment in %q", ErrUnsafeWorkspacePath, relPath)
	}

	cleaned := pathClean(normalized)
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("%w: invalid path %q", ErrUnsafeWorkspacePath, relPath)
	}

	absRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return "", fmt.Errorf("workspace root: %w", err)
	}
	target := filepath.Join(absRoot, filepath.FromSlash(cleaned))
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("workspace target: %w", err)
	}

	rootPrefix := absRoot + string(os.PathSeparator)
	if absTarget != absRoot && !strings.HasPrefix(absTarget, rootPrefix) {
		return "", fmt.Errorf("%w: resolves outside workspace: %q", ErrUnsafeWorkspacePath, relPath)
	}

	return cleaned, nil
}

// pathWithinRoot reports whether target resolves inside workspaceRoot (zip-slip guard after join).
func pathWithinRoot(workspaceRoot, target string) bool {
	absRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rootPrefix := absRoot + string(os.PathSeparator)
	return absTarget == absRoot || strings.HasPrefix(absTarget, rootPrefix)
}

func pathClean(path string) string {
	parts := strings.Split(path, "/")
	var clean []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return ""
		}
		clean = append(clean, part)
	}
	return strings.Join(clean, "/")
}

// ListWorkspaceFiles enumerates relative file paths under a workspace root for scanning.
func ListWorkspaceFiles(root string, maxFiles int) ([]FileEntry, error) {
	return listWorkspaceEntries(root, maxFiles)
}

// CreateWorkspace writes files into a temporary directory tree for scanner tools.
func CreateWorkspace(files []FileEntry) (dir string, cleanup func(), err error) {
	dir, err = os.MkdirTemp("", "rd-scan-*")
	if err != nil {
		return "", nil, err
	}

	cleanup = func() {
		_ = os.RemoveAll(dir)
	}

	written := make(map[string]bool)
	for _, file := range files {
		if file.Content == "" {
			continue
		}
		safePath, err := ValidateWorkspacePath(dir, file.Path)
		if err != nil {
			cleanup()
			return "", nil, err
		}
		target := filepath.Join(dir, filepath.FromSlash(safePath))
		if err := writeFile(target, file.Content); err != nil {
			cleanup()
			return "", nil, err
		}
		written[safePath] = true
	}

	if len(written) == 0 {
		cleanup()
		return "", nil, fmt.Errorf("no files to scan")
	}

	return dir, cleanup, nil
}

// WriteWorkspaceBytes writes content to relPath inside workspaceRoot after containment checks.
func WriteWorkspaceBytes(workspaceRoot, relPath string, content []byte, perm os.FileMode) error {
	safeRel, err := ValidateWorkspacePath(workspaceRoot, relPath)
	if err != nil {
		return err
	}
	absRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return fmt.Errorf("workspace root: %w", err)
	}
	target := filepath.Join(absRoot, filepath.FromSlash(safeRel))
	if !pathWithinRoot(absRoot, target) {
		return fmt.Errorf("%w: path escape %q", ErrUnsafeWorkspacePath, relPath)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return err
	}
	return os.WriteFile(target, content, perm)
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}
