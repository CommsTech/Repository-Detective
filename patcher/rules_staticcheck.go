package patcher

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"git.commsnet.org/commstech/repository-detective/remediation"
	"git.commsnet.org/commstech/repository-detective/scanners"
)

var fmtSprintfLiteral = regexp.MustCompile(`fmt\.Sprintf\("([^"\\]|\\.)*"\)`)
var fmtPackageUse = regexp.MustCompile(`\bfmt\.`)

func applyStaticcheckPatch(plan remediation.Plan, workspaceDir string, maxFiles, maxLines int) (PatchResult, error) {
	if len(plan.AffectedFiles) == 0 {
		return PatchResult{}, fmt.Errorf("no affected file for patch")
	}
	if len(plan.AffectedFiles) > maxFiles {
		return PatchResult{}, fmt.Errorf("too many files to patch")
	}
	path := plan.AffectedFiles[0]
	full, err := patchWorkspaceFile(workspaceDir, path)
	if err != nil {
		return PatchResult{}, err
	}
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
		updated = removeUnusedFmtImport(updated)
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
	if err := scanners.WriteWorkspaceBytes(workspaceDir, path, []byte(updated), 0o600); err != nil {
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

func removeUnusedFmtImport(content string) string {
	if fmtPackageUse.MatchString(stripGoComments(content)) {
		return content
	}
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	inImportBlock := false
	blockStart := -1
	blockLines := make([]string, 0, 4)

	flushBlock := func() {
		if blockStart < 0 {
			return
		}
		kept := make([]string, 0, len(blockLines))
		for _, bl := range blockLines {
			trimmed := strings.TrimSpace(bl)
			if trimmed == `"fmt"` || strings.HasPrefix(trimmed, `"fmt" `) {
				continue
			}
			kept = append(kept, bl)
		}
		// Drop empty import blocks when fmt was the only import.
		if len(kept) <= 2 {
			blockStart = -1
			blockLines = blockLines[:0]
			inImportBlock = false
			return
		}
		out = append(out, kept...)
		blockStart = -1
		blockLines = blockLines[:0]
		inImportBlock = false
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == `import "fmt"` {
			continue
		}
		if strings.HasPrefix(trimmed, "import (") {
			inImportBlock = true
			blockStart = len(out)
			blockLines = append(blockLines, line)
			continue
		}
		if inImportBlock {
			blockLines = append(blockLines, line)
			if trimmed == ")" {
				flushBlock()
			}
			continue
		}
		out = append(out, line)
	}
	if inImportBlock {
		out = append(out, blockLines...)
	}
	return strings.Join(out, "\n")
}

func stripGoComments(content string) string {
	var b strings.Builder
	inBlock := false
	for _, line := range strings.Split(content, "\n") {
		if inBlock {
			if idx := strings.Index(line, "*/"); idx >= 0 {
				inBlock = false
				line = line[idx+2:]
			} else {
				continue
			}
		}
		if idx := strings.Index(line, "/*"); idx >= 0 {
			before := line[:idx]
			after := line[idx+2:]
			if end := strings.Index(after, "*/"); end >= 0 {
				line = before + after[end+2:]
			} else {
				inBlock = true
				line = before
			}
		}
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
