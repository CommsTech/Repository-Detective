package graph

import (
	"os"
	"path/filepath"

	"git.commsnet.org/commstech/bugbot/scanners"
)

// LoadWorkspaceFiles reads text files from a workspace for graph analysis.
func LoadWorkspaceFiles(root string, entries []scanners.FileEntry, maxFileBytes int64, skipPatterns []string) []FileInput {
	if maxFileBytes <= 0 {
		maxFileBytes = 512 * 1024
	}
	var out []FileInput
	for _, entry := range entries {
		if ShouldSkipPath(entry.Path, skipPatterns) {
			continue
		}
		if entry.Content != "" {
			out = append(out, FileInput{Path: entry.Path, Content: entry.Content, Language: detectLanguage(entry.Path)})
			continue
		}
		abs := filepath.Join(root, filepath.FromSlash(entry.Path))
		info, err := os.Stat(abs)
		if err != nil || info.IsDir() || info.Size() > maxFileBytes {
			continue
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		out = append(out, FileInput{
			Path:     entry.Path,
			Content:  string(data),
			Language: detectLanguage(entry.Path),
		})
	}
	return out
}

// FindingsFromCandidates converts candidate overlays from scan findings.
func FindingsFromCandidates(items []FindingOverlay) []FindingOverlay {
	return items
}
