package health

import (
	"os"
	"path/filepath"
	"strings"

	"git.commsnet.org/commstech/repository-detective/scanners"
)

// Run executes enabled health checks against a workspace.
func Run(input RunInput, cfg Config, skipPatterns []string) []Finding {
	if !cfg.Enabled {
		return nil
	}
	cfg = cfg.normalized()
	files := filterFiles(input.Files, skipPatterns)
	paths := filterPaths(input.AllPaths, skipPatterns)
	if len(paths) == 0 {
		for _, f := range files {
			paths = append(paths, f.Path)
		}
	}

	// enrich test-gap detection from package.json
	for _, f := range files {
		if strings.EqualFold(filepath.Base(f.Path), "package.json") && ScanPackageJSONForTestScript(f.Content) {
			paths = append(paths, "tests/.keep")
		}
	}

	var findings []Finding
	if cfg.EnableTechDebt {
		findings = append(findings, runTechDebtChecks(files)...)
	}
	if cfg.EnableReliability {
		findings = append(findings, runReliabilityChecks(files)...)
	}
	if cfg.EnableMaintainability {
		findings = append(findings, runMaintainabilityChecks(files, cfg)...)
	}
	if cfg.EnableTestGap {
		findings = append(findings, runTestGapChecks(paths, files)...)
	}
	if cfg.EnablePerformance {
		findings = append(findings, runPerformanceChecks(files)...)
	}
	if cfg.EnableAIRisk {
		findings = append(findings, runAIRiskChecks(files)...)
	}

	if len(findings) > cfg.MaxFindings {
		findings = findings[:cfg.MaxFindings]
	}
	return findings
}

// LoadWorkspaceFiles reads text files from a workspace for health analysis.
func LoadWorkspaceFiles(root string, entries []scanners.FileEntry, maxFileBytes int64) []FileInput {
	if maxFileBytes <= 0 {
		maxFileBytes = 512 * 1024
	}
	var out []FileInput
	for _, entry := range entries {
		if ShouldSkipPath(entry.Path, nil) {
			continue
		}
		if entry.Content != "" {
			out = append(out, FileInput{Path: entry.Path, Content: entry.Content, Language: detectLang(entry.Path)})
			continue
		}
		safePath, err := scanners.ValidateWorkspacePath(root, entry.Path)
		if err != nil {
			continue
		}
		abs := filepath.Join(root, filepath.FromSlash(safePath))
		info, err := os.Stat(abs)
		if err != nil || info.IsDir() || info.Size() > maxFileBytes {
			continue
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		if !isTextContent(data) {
			continue
		}
		out = append(out, FileInput{
			Path:     entry.Path,
			Content:  string(data),
			Language: detectLang(entry.Path),
		})
	}
	return out
}

func isTextContent(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	nonText := 0
	limit := len(data)
	if limit > 8000 {
		limit = 8000
	}
	for _, b := range data[:limit] {
		if b == 0 {
			return false
		}
		if b < 9 || (b > 13 && b < 32) {
			nonText++
		}
	}
	return nonText*20 < limit
}
