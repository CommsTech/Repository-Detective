package sbom

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PersistArtifact copies a workspace SBOM file into durable storage under
// <dataRoot>/sbom/<repoID>/<scanID>/ so downloads survive tmp workspace cleanup.
func PersistArtifact(dataRoot string, repositoryID int64, scanID, src string) (string, error) {
	src = strings.TrimSpace(src)
	if src == "" {
		return "", fmt.Errorf("empty SBOM artifact path")
	}
	info, err := os.Stat(src)
	if err != nil {
		return "", fmt.Errorf("stat source SBOM: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("SBOM artifact is not a regular file: %s", src)
	}

	root := strings.TrimSpace(dataRoot)
	if root == "" || root == "." {
		root = "/app/data"
	}
	root = filepath.Clean(root)
	destDir := filepath.Join(root, "sbom", fmt.Sprintf("%d", repositoryID), sanitizePathSegment(scanID))
	// 0750: durable SBOM dirs stay group-readable for operators without world access (gosec G301).
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return "", fmt.Errorf("mkdir durable SBOM dir: %w", err)
	}

	base := filepath.Base(src)
	if base == "." || base == string(filepath.Separator) || base == "" {
		base = "sbom.cdx.json"
	}
	dest := filepath.Join(destDir, base)
	if err := copyFileAtomic(src, dest, 0o600); err != nil {
		return "", err
	}
	return dest, nil
}

// DataRootFromDatabasePath returns the durable data directory from a SQLite path.
func DataRootFromDatabasePath(databasePath string) string {
	root := filepath.Clean(filepath.Dir(strings.TrimSpace(databasePath)))
	if root == "" || root == "." {
		return "/app/data"
	}
	return root
}

// ArtifactFileAvailable reports whether a stored artifact path still exists on disk.
func ArtifactFileAvailable(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func sanitizePathSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" || out == "." || out == ".." {
		return "unknown"
	}
	return out
}

func copyFileAtomic(src, dest string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source SBOM: %w", err)
	}
	defer in.Close()

	tmp := dest + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("create durable SBOM temp: %w", err)
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("copy SBOM bytes: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close durable SBOM temp: %w", closeErr)
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename durable SBOM: %w", err)
	}
	return nil
}
