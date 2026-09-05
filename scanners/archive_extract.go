package scanners

import (
	"archive/zip"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// ExtractZipArchive extracts a zip into destRoot with zip-slip protection and limits.
func ExtractZipArchive(zipPath, destRoot string, maxFiles int, maxTotalBytes int64) (fileCount int, totalBytes int64, truncated bool, err error) {
	if maxFiles <= 0 {
		maxFiles = 5000
	}
	if maxTotalBytes <= 0 {
		maxTotalBytes = 500 * 1024 * 1024
	}

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return 0, 0, false, fmt.Errorf("open zip: %w", err)
	}
	defer reader.Close()

	absRoot, err := filepath.Abs(destRoot)
	if err != nil {
		return 0, 0, false, err
	}
	if err := os.MkdirAll(absRoot, 0o750); err != nil {
		return 0, 0, false, err
	}

	entries := make([]*zip.File, 0, len(reader.File))
	for _, file := range reader.File {
		if file.FileInfo().Mode()&os.ModeSymlink != 0 {
			continue
		}
		name := strings.TrimPrefix(filepath.ToSlash(file.Name), "/")
		if name == "" || strings.HasSuffix(name, "/") {
			continue
		}
		entries = append(entries, file)
	}

	stripPrefix := commonZipRootPrefix(entries)

	for _, file := range entries {
		if fileCount >= maxFiles || totalBytes >= maxTotalBytes {
			truncated = true
			break
		}

		rel := strings.TrimPrefix(filepath.ToSlash(file.Name), "/")
		if stripPrefix != "" {
			rel = strings.TrimPrefix(rel, stripPrefix)
			rel = strings.TrimPrefix(rel, "/")
		}
		if rel == "" {
			continue
		}

		safeRel, err := ValidateWorkspacePath(absRoot, rel)
		if err != nil {
			return fileCount, totalBytes, truncated, fmt.Errorf("unsafe zip entry %q: %w", file.Name, err)
		}

		target := filepath.Join(absRoot, filepath.FromSlash(safeRel))
		if !pathWithinRoot(absRoot, target) {
			return fileCount, totalBytes, truncated, fmt.Errorf("unsafe zip entry %q: path outside dest root", file.Name)
		}
		if file.FileInfo().IsDir() || strings.HasSuffix(file.Name, "/") {
			if err := os.MkdirAll(target, 0o750); err != nil {
				return fileCount, totalBytes, truncated, err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return fileCount, totalBytes, truncated, err
		}

		if uncompressedSizeWouldExceed(totalBytes, file.UncompressedSize64, maxTotalBytes) {
			truncated = true
			break
		}

		rc, err := file.Open()
		if err != nil {
			return fileCount, totalBytes, truncated, err
		}

		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.Mode().Perm()&0o644|0o600)
		if err != nil {
			rc.Close()
			return fileCount, totalBytes, truncated, err
		}

		written, copyErr := io.Copy(out, io.LimitReader(rc, maxTotalBytes-totalBytes+1))
		closeErr := out.Close()
		rc.Close()
		if copyErr != nil {
			return fileCount, totalBytes, truncated, copyErr
		}
		if closeErr != nil {
			return fileCount, totalBytes, truncated, closeErr
		}

		fileCount++
		totalBytes += written
	}

	return fileCount, totalBytes, truncated, nil
}

// uncompressedSizeWouldExceed checks byte limits without unsafe uint64→int64 casts (gosec G115).
func uncompressedSizeWouldExceed(totalBytes int64, size uint64, maxTotalBytes int64) bool {
	if maxTotalBytes <= 0 || totalBytes >= maxTotalBytes {
		return true
	}
	if size > uint64(maxTotalBytes) {
		return true
	}
	if size > uint64(math.MaxInt64) {
		return true
	}
	addend := int64(size)
	return addend > maxTotalBytes-totalBytes
}

func commonZipRootPrefix(files []*zip.File) string {
	if len(files) == 0 {
		return ""
	}

	var prefix string
	for i, file := range files {
		name := strings.TrimPrefix(filepath.ToSlash(file.Name), "/")
		parts := strings.Split(name, "/")
		if len(parts) < 2 {
			return ""
		}
		if i == 0 {
			prefix = parts[0]
			continue
		}
		if parts[0] != prefix {
			return ""
		}
	}
	if prefix == "" {
		return ""
	}
	return prefix + "/"
}
