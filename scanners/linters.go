package scanners

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type linterSpec struct {
	name     string
	lang     string
	quality  bool
	matchExt map[string]bool
	run      func(ctx context.Context, logger *logrus.Logger, dir string, files []string, cfg Config) ([]Finding, error)
}

// RunLinters executes language-specific linters against files in the workspace.
func RunLinters(ctx context.Context, logger *logrus.Logger, dir string, entries []FileEntry, enableSecurity, enableQuality bool, cfg Config) ([]Finding, error) {
	byExt := groupFilesByExtension(entries)
	if len(byExt) == 0 {
		return nil, nil
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

	var all []Finding
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
			continue
		}
		if !spec.quality && !enableSecurity {
			continue
		}

		findings, err := spec.run(ctx, logger, dir, matched, cfg)
		if err != nil {
			logger.Warnf("[SCANNER:%s] %v", spec.name, err)
			continue
		}
		all = append(all, findings...)
	}

	logger.Infof("[SCANNER:linters] found %d issue(s)", len(all))
	return all, nil
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
}

func runGolangciLint(ctx context.Context, logger *logrus.Logger, dir string, files []string, cfg Config) ([]Finding, error) {
	if !commandAvailable("golangci-lint") {
		logger.Warn("[SCANNER:golangci-lint] binary not found")
		return nil, nil
	}

	var findings []Finding
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second

	for _, relPath := range files {
		target := filepath.Join(dir, filepath.FromSlash(relPath))
		if _, err := os.Stat(target); err != nil {
			continue
		}

		args := []string{
			"run",
			"--out-format", "json",
			"--issues-exit-code=0",
			target,
		}
		output, err := runCommand(ctx, timeout, dir, "golangci-lint", args...)
		if err != nil && len(output) == 0 {
			continue
		}

		var report golangciReport
		if err := json.Unmarshal(output, &report); err != nil {
			continue
		}

		for _, issue := range report.Issues {
			severity := normalizeSeverity(issue.Severity)
			if severity == "medium" && issue.FromLinter != "" {
				severity = linterSeverityFromName(issue.FromLinter)
			}
			if !meetsMinSeverity(severity, cfg.LinterMinSeverity) {
				continue
			}

			file := strings.TrimPrefix(issue.Pos.Filename, dir)
			file = strings.TrimPrefix(file, string(filepath.Separator))
			findings = append(findings, Finding{
				ID:          fmt.Sprintf("LINT-GO-%s-%d", issue.FromLinter, issue.Pos.Line),
				Source:      "golangci-lint",
				Category:    "lint",
				Severity:    severity,
				Title:       issue.Text,
				Description: fmt.Sprintf("%s reported by golangci-lint (%s)", issue.Text, issue.FromLinter),
				File:        firstNonEmpty(file, relPath),
				Line:        issue.Pos.Line,
				Confidence:  0.9,
				Reference:   issue.FromLinter,
				Code:        issue.Text,
			})
		}
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

func runRuff(ctx context.Context, logger *logrus.Logger, dir string, files []string, cfg Config) ([]Finding, error) {
	if !commandAvailable("ruff") {
		logger.Warn("[SCANNER:ruff] binary not found")
		return nil, nil
	}

	var targets []string
	for _, relPath := range files {
		target := filepath.Join(dir, filepath.FromSlash(relPath))
		if _, err := os.Stat(target); err == nil {
			targets = append(targets, target)
		}
	}
	if len(targets) == 0 {
		return nil, nil
	}

	args := append([]string{"check", "--output-format", "json"}, targets...)
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	output, err := runCommand(ctx, timeout, dir, "ruff", args...)
	if err != nil && len(output) == 0 {
		return nil, nil
	}

	var issues []ruffIssue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, nil
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
			ID:          fmt.Sprintf("LINT-RUFF-%s-%d", issue.Code, issue.Location.Row),
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

func runShellcheck(ctx context.Context, logger *logrus.Logger, dir string, files []string, cfg Config) ([]Finding, error) {
	if !commandAvailable("shellcheck") {
		logger.Warn("[SCANNER:shellcheck] binary not found")
		return nil, nil
	}

	var targets []string
	for _, relPath := range files {
		target := filepath.Join(dir, filepath.FromSlash(relPath))
		if _, err := os.Stat(target); err == nil {
			targets = append(targets, target)
		}
	}
	if len(targets) == 0 {
		return nil, nil
	}

	args := append([]string{"-f", "json"}, targets...)
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	output, err := runCommand(ctx, timeout, dir, "shellcheck", args...)
	if err != nil && len(output) == 0 {
		return nil, nil
	}

	var reports [][]shellcheckIssue
	if err := json.Unmarshal(output, &reports); err != nil {
		return nil, nil
	}

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
				ID:          fmt.Sprintf("LINT-SHELL-%d-%d", issue.Code, issue.Line),
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

	return findings, nil
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
