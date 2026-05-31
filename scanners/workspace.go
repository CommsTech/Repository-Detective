package scanners

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

// CreateWorkspace writes files into a temporary directory tree for scanner tools.
func CreateWorkspace(files []FileEntry) (dir string, cleanup func(), err error) {
	dir, err = os.MkdirTemp("", "bugbot-scan-*")
	if err != nil {
		return "", nil, err
	}

	cleanup = func() {
		_ = os.RemoveAll(dir)
	}

	written := make(map[string]bool)
	for _, file := range files {
		path := strings.TrimPrefix(filepath.ToSlash(file.Path), "/")
		if path == "" || file.Content == "" {
			continue
		}
		target := filepath.Join(dir, path)
		if err := writeFile(target, file.Content); err != nil {
			cleanup()
			return "", nil, err
		}
		written[path] = true
	}

	if len(written) == 0 {
		cleanup()
		return "", nil, fmt.Errorf("no files to scan")
	}

	return dir, cleanup, nil
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
