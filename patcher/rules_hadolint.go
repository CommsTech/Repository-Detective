package patcher

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"git.commsnet.org/commstech/bugbot/remediation"
)

var apkAddLinePattern = regexp.MustCompile(`(?i)(apk\s+add(?:\s+--[^\s\\&]+)*\s+)([^\\&]+)`)

func applyHadolintPatch(plan remediation.Plan, workspaceDir string, maxFiles, maxLines int) (PatchResult, error) {
	rule := strings.ToUpper(strings.TrimSpace(plan.RuleID))
	switch {
	case rule == "DL3018" || strings.HasPrefix(rule, "DL3018"):
		return applyHadolintApkPinPatch(plan, workspaceDir, maxFiles, maxLines)
	default:
		return applyHadolintAptPatch(plan, workspaceDir, maxFiles, maxLines)
	}
}

func applyHadolintAptPatch(plan remediation.Plan, workspaceDir string, maxFiles, maxLines int) (PatchResult, error) {
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

func applyHadolintApkPinPatch(plan remediation.Plan, workspaceDir string, maxFiles, maxLines int) (PatchResult, error) {
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
		if !strings.Contains(strings.ToLower(line), "apk add") {
			continue
		}
		pinned, ok := pinApkAddPackages(line)
		if ok && pinned != line {
			lines[i] = pinned
			changed = true
		}
	}
	if !changed {
		return PatchResult{}, fmt.Errorf("no applicable hadolint DL3018 patch")
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
		Summary:      "hadolint DL3018: pin apk packages with =* version placeholders",
	}, nil
}

func pinApkAddPackages(line string) (string, bool) {
	m := apkAddLinePattern.FindStringSubmatch(line)
	if len(m) < 3 {
		return line, false
	}
	prefix, pkgPart := m[1], strings.TrimSpace(m[2])
	pkgPart = strings.TrimSuffix(pkgPart, `\`)
	pkgPart = strings.TrimSpace(strings.TrimSuffix(pkgPart, "&&"))
	if pkgPart == "" {
		return line, false
	}
	tokens := strings.Fields(pkgPart)
	changed := false
	for i, tok := range tokens {
		if strings.Contains(tok, "=") {
			continue
		}
		tokens[i] = tok + "=*"
		changed = true
	}
	if !changed {
		return line, false
	}
	suffix := strings.TrimPrefix(line, prefix+strings.TrimSpace(m[2]))
	rebuilt := prefix + strings.Join(tokens, " ") + suffix
	return rebuilt, true
}
