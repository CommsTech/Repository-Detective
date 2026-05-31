package patcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"git.commsnet.org/commstech/bugbot/remediation"
)

func applyHadolintPatch(plan remediation.Plan, workspaceDir string, maxFiles, maxLines int) (PatchResult, error) {
	if len(plan.AffectedFiles) == 0 {
		return PatchResult{}, fmt.Errorf("no affected file for patch")
	}
	if len(plan.AffectedFiles) > maxFiles {
		return PatchResult{}, fmt.Errorf("too many files to patch")
	}
	path := plan.AffectedFiles[0]
	if !isSafeRelativePath(path) {
		return PatchResult{}, fmt.Errorf("unsafe file path")
	}
	full := filepath.Join(workspaceDir, filepath.FromSlash(path))
	data, err := os.ReadFile(full)
	if err != nil {
		return PatchResult{}, fmt.Errorf("read file: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	changed := false
	for i, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "apt-get install") && !strings.Contains(lower, "--no-install-recommends") {
			lines[i] = strings.Replace(line, "apt-get install", "apt-get install --no-install-recommends", 1)
			changed = true
		}
	}
	if !changed {
		return PatchResult{}, fmt.Errorf("no applicable hadolint patch")
	}
	updated := strings.Join(lines, "\n")
	diffLines := countChangedLines(string(data), updated)
	if diffLines > maxLines {
		return PatchResult{}, fmt.Errorf("patch exceeds max diff lines")
	}
	if err := os.WriteFile(full, []byte(updated), 0o644); err != nil {
		return PatchResult{}, fmt.Errorf("write file: %w", err)
	}
	return PatchResult{
		FilesChanged: []string{path},
		DiffLines:    diffLines,
		Summary:      "hadolint patch: add --no-install-recommends to apt-get install",
	}, nil
}
