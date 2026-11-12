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

	timeout := golangciTimeout(cfg.TimeoutSeconds)
	args := golangciRunArgs(ctx)
	output, err := runCommand(ctx, timeout, dir, "golangci-lint", args...)
	if golangciConfigRejected(output, err) {
		configPath, cleanup, writeErr := writeGolangciV2FallbackConfig()
		if writeErr == nil {
			defer cleanup()
			retryArgs := append(append([]string{}, args...), "--config", configPath)
			output, err = runCommand(ctx, timeout, dir, "golangci-lint", retryArgs...)
		}
	}
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

// golangciTimeout keeps typechecking from being killed by the shared scanner cap.
// golangci-lint v2 loads the whole module; a 3–5 minute cap returns no JSON.
func golangciTimeout(scannerTimeoutSeconds int) time.Duration {
	timeout := time.Duration(scannerTimeoutSeconds) * time.Second
	if timeout < 10*time.Minute {
		return 10 * time.Minute
	}
	return timeout
}

func golangciConfigRejected(output []byte, err error) bool {
	text := strings.ToLower(string(output))
	if err != nil {
		text += " " + strings.ToLower(err.Error())
	}
	return strings.Contains(text, "unsupported version of the configuration") ||
		strings.Contains(text, "can't load config") ||
		strings.Contains(text, "cannot load config")
}

// writeGolangciV2FallbackConfig is used when a repository still has a v1
// .golangci.yml. golangci-lint v2 refuses that file and prints a plain-text
// error, which the JSON parser then reports as a scan failure.
func writeGolangciV2FallbackConfig() (string, func(), error) {
	f, err := os.CreateTemp("", "rd-golangci-*.yml")
	if err != nil {
		return "", nil, err
	}
	if _, err := f.WriteString(golangciV2FallbackYAML); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", nil, err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", nil, err
	}
	path := f.Name()
	return path, func() { _ = os.Remove(path) }, nil
}

const golangciV2FallbackYAML = `version: "2"
run:
  timeout: 10m
linters:
  default: none
  enable:
    - errcheck
    - govet
    - ineffassign
    - staticcheck
    - unused
formatters:
  enable:
    - gofmt
issues:
  max-issues-per-linter: 0
  max-same-issues: 0
`

func golangciRunArgs(ctx context.Context) []string {
	out, _ := runCommand(ctx, 15*time.Second, "", "golangci-lint", "version")
	return golangciRunArgsForVersion(string(out))
}

// golangciRunArgsForVersion picks CLI flags that match the installed major version.
// v2 removed --out-format; v1 does not accept --output.json.path.
func golangciRunArgsForVersion(versionOutput string) []string {
	if golangciMajor(versionOutput) >= 2 {
		return []string{
			"run", "./...",
			"--output.json.path=stdout",
			"--output.text.path=/dev/null",
			"--issues-exit-code=0",
		}
	}
	return []string{"run", "./...", "--out-format", "json", "--issues-exit-code=0"}
}

func golangciMajor(versionOutput string) int {
	lower := strings.ToLower(versionOutput)
	const marker = "version "
	idx := strings.Index(lower, marker)
	if idx < 0 {
		return 1
	}
	rest := strings.TrimSpace(lower[idx+len(marker):])
	if rest == "" {
		return 1
	}
	major := 0
	for _, r := range rest {
		if r < '0' || r > '9' {
			break
		}
		major = major*10 + int(r-'0')
	}
	if major == 0 {
		return 1
	}
	return major
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
