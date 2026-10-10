package scanners

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Paths under these prefixes are excluded from golangci-lint findings (fixtures, vendored code).
var linterSkipPathPrefixes = []string{
	"testdata/",
	"vendor/",
	"benchmark/",
	".cache/",
}

type linterSpec struct {
	name     string
	lang     string
	quality  bool
	matchExt map[string]bool
	run      func(ctx context.Context, logger *logrus.Logger, dir string, files []string, cfg Config) RunResult
}

// RunLinters executes language-specific linters against files in the workspace.
func RunLinters(ctx context.Context, logger *logrus.Logger, dir string, entries []FileEntry, enableSecurity, enableQuality bool, cfg Config) []RunResult {
	byExt := groupFilesByExtension(entries)
	if len(byExt) == 0 {
		return nil
	}

	specs := []linterSpec{
		{
			name:     "golangci-lint",
			lang:     "go",
			matchExt: map[string]bool{".go": true},
			run:      runGolangciLint,
		},
		{
			name:     "ruff",
			lang:     "python",
			matchExt: map[string]bool{".py": true},
			run:      runRuff,
		},
		{
			name:     "shellcheck",
			lang:     "shell",
			matchExt: map[string]bool{".sh": true, ".bash": true},
			run:      runShellcheck,
		},
	}

	type job struct {
		spec    linterSpec
		matched []string
	}
	var jobs []job
	for _, spec := range specs {
		var matched []string
		for ext, paths := range byExt {
			if spec.matchExt[ext] {
				matched = append(matched, paths...)
			}
		}
		if len(matched) == 0 {
			continue
		}
		if spec.quality && !enableQuality {
			jobs = append(jobs, job{spec: spec})
			continue
		}
		if !spec.quality && !enableSecurity {
			jobs = append(jobs, job{spec: spec})
			continue
		}
		jobs = append(jobs, job{spec: spec, matched: matched})
	}
	if len(jobs) == 0 {
		return nil
	}

	results := make([]RunResult, len(jobs))
	var wg sync.WaitGroup
	for i, j := range jobs {
		if j.matched == nil {
			detail := "security analysis disabled"
			if j.spec.quality {
				detail = "quality analysis disabled"
			}
			results[i] = RunResult{Scanner: j.spec.name, Status: StatusDisabled, Detail: detail}
			continue
		}
		wg.Add(1)
		go func(idx int, spec linterSpec, matched []string) {
			defer wg.Done()
			results[idx] = spec.run(ctx, logger, dir, matched, cfg)
		}(i, j.spec, j.matched)
	}
	wg.Wait()
	return results
}

func groupFilesByExtension(entries []FileEntry) map[string][]string {
	grouped := make(map[string][]string)
	for _, entry := range entries {
		ext := strings.ToLower(filepath.Ext(entry.Path))
		if ext == "" {
			continue
		}
		grouped[ext] = append(grouped[ext], entry.Path)
	}
	return grouped
}

type golangciIssue struct {
	Text       string `json:"Text"`
	FromLinter string `json:"FromLinter"`
	Severity   string `json:"Severity"`
	Pos        struct {
		Filename string `json:"Filename"`
		Line     int    `json:"Line"`
	} `json:"Pos"`
}

type golangciReport struct {
	Issues []golangciIssue `json:"Issues"`
	Error  string          `json:"Error"`
}

func runGolangciLint(ctx context.Context, logger *logrus.Logger, dir string, files []string, cfg Config) RunResult {
	result := RunResult{Scanner: "golangci-lint"}
	if !commandAvailable("golangci-lint") {
		logger.Warn("[SCANNER:golangci-lint] binary not found")
		result.Status = StatusBinaryMissing
		return result
	}
	if !WorkspaceHasGo(dir, fileEntriesFromPaths(files)) {
		result.Status = StatusClean
		result.Detail = "no Go module or files"
		return result
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	args := []string{
		"run",
		"./...",
		"--out-format", "json",
		"--issues-exit-code=0",
	}
	output, err := runCommand(ctx, timeout, dir, "golangci-lint", args...)
	if err != nil && len(output) == 0 {
		result.Status = classifyCommandError(err)
		result.Detail = err.Error()
		return result
	}

	findings, parseErr := parseGolangciOutput(output, dir, "", cfg)
	if parseErr != nil && len(findings) == 0 {
		result.Status = StatusParseFailed
		result.Detail = parseErr.Error()
		return result
	}

	result = resultWithFindings("golangci-lint", findings)
	logger.Infof("[SCANNER:golangci-lint] status=%s findings=%d", result.Status, len(findings))
	return result
}

func fileEntriesFromPaths(paths []string) []FileEntry {
	out := make([]FileEntry, 0, len(paths))
	for _, p := range paths {
		out = append(out, FileEntry{Path: p})
	}
	return out
}

func shouldSkipLinterPath(path string) bool {
	path = strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(path)), "/")
	for _, prefix := range linterSkipPathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func parseGolangciOutput(output []byte, dir, relPath string, cfg Config) ([]Finding, error) {
	payload := output
	if raw, err := extractJSONObject(output); err == nil {
		payload = raw
	}
	var report golangciReport
	if err := json.Unmarshal(payload, &report); err != nil {
		return nil, err
	}
	if len(report.Issues) == 0 && strings.TrimSpace(report.Error) != "" {
		return nil, fmt.Errorf("golangci-lint: %s", strings.TrimSpace(report.Error))
	}

	var findings []Finding
	for _, issue := range report.Issues {
		file := strings.TrimPrefix(issue.Pos.Filename, dir)
		file = strings.TrimPrefix(file, string(filepath.Separator))
		file = firstNonEmpty(file, relPath)
		if shouldSkipLinterPath(file) {
			continue
		}
		severity := normalizeSeverity(issue.Severity)
		if severity == "medium" && issue.FromLinter != "" {
			severity = linterSeverityFromName(issue.FromLinter)
		}
		if !meetsMinSeverity(severity, cfg.LinterMinSeverity) {
			continue
		}

		findings = append(findings, Finding{
			ID:          stableLinterRuleID("LINT-GO", issue.FromLinter),
			Source:      "golangci-lint",
			Category:    "lint",
			Severity:    severity,
			Title:       issue.Text,
			Description: fmt.Sprintf("%s reported by golangci-lint (%s)", issue.Text, issue.FromLinter),
			File:        file,
			Line:        issue.Pos.Line,
			Confidence:  0.9,
			Reference:   issue.FromLinter,
			Code:        issue.Text,
		})
	}
	return findings, nil
}

type ruffIssue struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Filename string `json:"filename"`
	Location struct {
		Row int `json:"row"`
	} `json:"location"`
}

func runRuff(ctx context.Context, logger *logrus.Logger, dir string, files []string, cfg Config) RunResult {
	result := RunResult{Scanner: "ruff"}
	if !commandAvailable("ruff") {
		logger.Warn("[SCANNER:ruff] binary not found")
		result.Status = StatusBinaryMissing
		return result
	}

	var targets []string
	for _, relPath := range files {
		target := filepath.Join(dir, filepath.FromSlash(relPath))
		if _, err := os.Stat(target); err == nil {
			targets = append(targets, target)
		}
	}
	if len(targets) == 0 {
		result.Status = StatusClean
		return result
	}

	args := append([]string{"check", "--output-format", "json"}, targets...)
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	output, err := runCommand(ctx, timeout, dir, "ruff", args...)
	if err != nil && len(output) == 0 {
		result.Status = classifyCommandError(err)
		result.Detail = err.Error()
		return result
	}

	findings, parseErr := parseRuffOutput(output, dir, cfg)
	if parseErr != nil {
		result.Status = StatusParseFailed
		result.Detail = parseErr.Error()
		return result
	}

	result = resultWithFindings("ruff", findings)
	logger.Infof("[SCANNER:ruff] status=%s findings=%d", result.Status, len(findings))
	return result
}

func parseRuffOutput(output []byte, dir string, cfg Config) ([]Finding, error) {
	var issues []ruffIssue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, err
	}

	var findings []Finding
	for _, issue := range issues {
		severity := ruffSeverity(issue.Code)
		if !meetsMinSeverity(severity, cfg.LinterMinSeverity) {
			continue
		}
		file := strings.TrimPrefix(issue.Filename, dir)
		file = strings.TrimPrefix(file, string(filepath.Separator))
		findings = append(findings, Finding{
			ID:          stableLinterRuleID("LINT-RUFF", issue.Code),
			Source:      "ruff",
			Category:    "lint",
			Severity:    severity,
			Title:       issue.Message,
			Description: fmt.Sprintf("%s (%s)", issue.Message, issue.Code),
			File:        file,
			Line:        issue.Location.Row,
			Confidence:  0.9,
			Reference:   issue.Code,
			Code:        issue.Message,
		})
	}
	return findings, nil
}

func ruffSeverity(code string) string {
	if strings.HasPrefix(strings.ToUpper(code), "S") || strings.HasPrefix(strings.ToUpper(code), "B") {
		return "high"
	}
	return "medium"
}

type shellcheckIssue struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Level   string `json:"level"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func runShellcheck(ctx context.Context, logger *logrus.Logger, dir string, files []string, cfg Config) RunResult {
	result := RunResult{Scanner: "shellcheck"}
	if !commandAvailable("shellcheck") {
		logger.Warn("[SCANNER:shellcheck] binary not found")
		result.Status = StatusBinaryMissing
		return result
	}

	var targets []string
	for _, relPath := range files {
		target := filepath.Join(dir, filepath.FromSlash(relPath))
		if _, err := os.Stat(target); err == nil {
			targets = append(targets, target)
		}
	}
	if len(targets) == 0 {
		result.Status = StatusClean
		return result
	}

	args := append([]string{"-f", "json"}, targets...)
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	output, err := runCommand(ctx, timeout, dir, "shellcheck", args...)
	if err != nil && len(output) == 0 {
		result.Status = classifyCommandError(err)
		result.Detail = err.Error()
		return result
	}

	findings, parseErr := parseShellcheckOutput(output, dir, cfg)
	if parseErr != nil {
		result.Status = StatusParseFailed
		result.Detail = parseErr.Error()
		return result
	}

	result = resultWithFindings("shellcheck", findings)
	logger.Infof("[SCANNER:shellcheck] status=%s findings=%d", result.Status, len(findings))
	return result
}

func parseShellcheckOutput(output []byte, dir string, cfg Config) ([]Finding, error) {
	payload := output
	if raw, err := extractJSONArray(output); err == nil {
		payload = raw
	}
	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" || trimmed == "[]" || trimmed == "null" {
		return nil, nil
	}

	// ShellCheck 0.10+ emits a flat [{...}] array; older builds used [[{...}]].
	var flat []shellcheckIssue
	if err := json.Unmarshal(payload, &flat); err == nil {
		return shellcheckIssuesToFindings([][]shellcheckIssue{flat}, dir, cfg), nil
	}

	var reports [][]shellcheckIssue
	if err := json.Unmarshal(payload, &reports); err != nil {
		return nil, err
	}
	return shellcheckIssuesToFindings(reports, dir, cfg), nil
}

func shellcheckIssuesToFindings(reports [][]shellcheckIssue, dir string, cfg Config) []Finding {
	var findings []Finding
	for _, group := range reports {
		for _, issue := range group {
			severity := shellcheckSeverity(issue.Level)
			if !meetsMinSeverity(severity, cfg.LinterMinSeverity) {
				continue
			}
			file := strings.TrimPrefix(issue.File, dir)
			file = strings.TrimPrefix(file, string(filepath.Separator))
			findings = append(findings, Finding{
				ID:          stableLinterRuleID("LINT-SHELL", fmt.Sprintf("%d", issue.Code)),
				Source:      "shellcheck",
				Category:    "lint",
				Severity:    severity,
				Title:       issue.Message,
				Description: fmt.Sprintf("ShellCheck SC%d: %s", issue.Code, issue.Message),
				File:        file,
				Line:        issue.Line,
				Confidence:  0.88,
				Reference:   fmt.Sprintf("SC%d", issue.Code),
				Code:        issue.Message,
			})
		}
	}
	return findings
}

// ParseShellcheckOutputForTest exposes shellcheck JSON parsing for unit tests.
func ParseShellcheckOutputForTest(output []byte, dir string, cfg Config) ([]Finding, error) {
	return parseShellcheckOutput(output, dir, cfg)
}

func shellcheckSeverity(level string) string {
	switch strings.ToLower(level) {
	case "error":
		return "high"
	case "warning":
		return "medium"
	case "info", "style":
		return "low"
	default:
		return "medium"
	}
}

func linterSeverityFromName(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "err"), strings.Contains(lower, "security"), strings.Contains(lower, "govet"):
		return "high"
	case strings.Contains(lower, "staticcheck"):
		return "medium"
	default:
		return "medium"
	}
}

// stableLinterRuleID returns a source+rule key suitable for calibration learning.
// Line numbers stay on Finding.Line; rule_id must aggregate across occurrences.
func stableLinterRuleID(prefix, ruleKey string) string {
	ruleKey = strings.TrimSpace(ruleKey)
	if ruleKey == "" {
		ruleKey = "unknown"
	}
	return prefix + "-" + ruleKey
}
