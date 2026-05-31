package patcher

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"git.commsnet.org/commstech/bugbot/remediation"
)

var fmtSprintfLiteral = regexp.MustCompile(`fmt\.Sprintf\("([^"\\]|\\.)*"\)`)

func applyStaticcheckPatch(plan remediation.Plan, workspaceDir string, maxFiles, maxLines int) (PatchResult, error) {
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
	original := string(data)
	updated := original
	rule := strings.ToLower(plan.RuleID)

	switch {
	case rule == "s1039" || strings.HasPrefix(rule, "s1039"):
		updated = fmtSprintfLiteral.ReplaceAllStringFunc(updated, func(match string) string {
			inner := strings.TrimPrefix(match, `fmt.Sprintf(`)
			inner = strings.TrimSuffix(inner, `)`)
			if strings.HasPrefix(inner, `"`) && strings.HasSuffix(inner, `"`) {
				return inner
			}
			return match
		})
	default:
		return PatchResult{}, fmt.Errorf("staticcheck rule %s not patchable", plan.RuleID)
	}

	if updated == original {
		return PatchResult{}, fmt.Errorf("no applicable staticcheck patch for rule %s", plan.RuleID)
	}
	diffLines := countChangedLines(original, updated)
	if diffLines > maxLines {
		return PatchResult{}, fmt.Errorf("patch exceeds max diff lines (%d)", maxLines)
	}
	if err := os.WriteFile(full, []byte(updated), 0o644); err != nil {
		return PatchResult{}, fmt.Errorf("write file: %w", err)
	}
	return PatchResult{
		FilesChanged: []string{path},
		DiffLines:    diffLines,
		Summary:      fmt.Sprintf("staticcheck patch (%s): %d lines changed", plan.RuleID, diffLines),
	}, nil
}

func countChangedLines(before, after string) int {
	b := strings.Split(before, "\n")
	a := strings.Split(after, "\n")
	n := 0
	max := len(b)
	if len(a) > max {
		max = len(a)
	}
	for i := 0; i < max; i++ {
		var bl, al string
		if i < len(b) {
			bl = b[i]
		}
		if i < len(a) {
			al = a[i]
		}
		if bl != al {
			n++
		}
	}
	if n == 0 {
		n = 1
	}
	return n
}
