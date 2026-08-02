package scanners

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const semgrepMaxCodeSnippetLen = 500

type semgrepReport struct {
	Results []semgrepMatch `json:"results"`
}

type semgrepMatch struct {
	CheckID string           `json:"check_id"`
	Path    string           `json:"path"`
	Start   semgrepPosition  `json:"start"`
	End     semgrepPosition  `json:"end"`
	Extra   semgrepMatchExtra `json:"extra"`
}

type semgrepPosition struct {
	Line   int `json:"line"`
	Column int `json:"col"`
	Offset int `json:"offset"`
}

type semgrepMatchExtra struct {
	Message     string                 `json:"message"`
	Metadata    map[string]interface{} `json:"metadata"`
	Severity    string                 `json:"severity"`
	Fingerprint string                 `json:"fingerprint"`
	Lines       json.RawMessage        `json:"lines"`
}

type semgrepParseResult struct {
	Findings  []Finding
	Total     int
	Truncated bool
}

func init() {
	RegisterDeterministicSource("semgrep")
}

// RunSemgrep scans a workspace directory with Semgrep SAST.
func RunSemgrep(ctx context.Context, logger *logrus.Logger, dir string, cfg Config) RunResult {
	return runSemgrepWithCommand(ctx, logger, dir, cfg, "semgrep")
}

func runSemgrepWithCommand(ctx context.Context, logger *logrus.Logger, dir string, cfg Config, commandName string) RunResult {
	result := RunResult{Scanner: "semgrep"}
	if !commandAvailable(commandName) {
		logger.Warn("[SCANNER:semgrep] binary not found — install semgrep or use the official Repository Detective Docker image")
		result.Status = StatusBinaryMissing
		return result
	}

	args := semgrepArgs(dir, cfg)
	timeout := semgrepTimeout(cfg)
	output, err := runCommand(ctx, timeout, dir, commandName, args...)
	if err != nil && len(output) == 0 {
		result.Status = classifyCommandError(err)
		result.Detail = err.Error()
		logger.Warnf("[SCANNER:semgrep] scan failed: status=%s err=%v", result.Status, err)
		return result
	}

	parsed, parseErr := parseSemgrepOutput(output, dir, cfg)
	if parseErr != nil {
		result.Status = StatusParseFailed
		result.Detail = parseErr.Error()
		logger.Warnf("[SCANNER:semgrep] failed to parse output: %v", parseErr)
		return result
	}

	result = resultWithFindings("semgrep", parsed.Findings)
	if parsed.Truncated {
		max := cfg.SemgrepMaxFindings
		if max <= 0 {
			max = defaultSemgrepMaxFindings()
		}
		result.Detail = fmt.Sprintf("truncated to %d findings (%d total)", max, parsed.Total)
	}
	logResultInfo(logger, "semgrep", result.Status, len(parsed.Findings), result.Detail)
	return result
}

func semgrepArgs(dir string, cfg Config) []string {
	config := strings.TrimSpace(cfg.SemgrepConfig)
	if config == "" {
		config = defaultSemgrepConfig()
	}
	return []string{
		"scan",
		"--json",
		"--quiet",
		"--metrics=off",
		"--config", config,
		dir,
	}
}

func defaultSemgrepConfig() string {
	return "p/ci"
}

func defaultSemgrepMaxFindings() int {
	return 100
}

func semgrepTimeout(cfg Config) time.Duration {
	seconds := cfg.SemgrepTimeoutSeconds
	if seconds <= 0 {
		seconds = cfg.TimeoutSeconds
	}
	return time.Duration(seconds) * time.Second
}

func parseSemgrepOutput(output []byte, dir string, cfg Config) (semgrepParseResult, error) {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return semgrepParseResult{}, nil
	}

	var report semgrepReport
	if err := json.Unmarshal(output, &report); err != nil {
		return semgrepParseResult{}, err
	}

	threshold := cfg.SemgrepSeverityThreshold
	if strings.TrimSpace(threshold) == "" {
		threshold = "INFO"
	}

	var findings []Finding
	for _, match := range report.Results {
		if !semgrepMeetsThreshold(match.Extra.Severity, threshold) {
			continue
		}
		findings = append(findings, semgrepMatchToFinding(match, dir))
	}

	total := len(findings)
	maxFindings := cfg.SemgrepMaxFindings
	if maxFindings <= 0 {
		maxFindings = defaultSemgrepMaxFindings()
	}

	truncated := false
	if len(findings) > maxFindings {
		findings = findings[:maxFindings]
		truncated = true
	}

	return semgrepParseResult{
		Findings:  findings,
		Total:     total,
		Truncated: truncated,
	}, nil
}

func semgrepMatchToFinding(match semgrepMatch, dir string) Finding {
	file := strings.TrimPrefix(match.Path, dir)
	file = strings.TrimPrefix(file, "/")
	file = strings.TrimPrefix(file, "\\")

	checkID := strings.TrimSpace(match.CheckID)
	if checkID == "" {
		checkID = "unknown-check"
	}

	severity := mapSemgrepSeverity(match.Extra.Severity)
	category := semgrepCategory(match.Extra.Metadata)
	description := buildSemgrepDescription(match, checkID)
	code := boundSemgrepSnippet(semgrepLinesSnippet(match.Extra.Lines))

	id := fmt.Sprintf("SEMGREP-%s", checkID)
	if match.Extra.Fingerprint != "" {
		id = fmt.Sprintf("SEMGREP-%s", match.Extra.Fingerprint)
	} else if file != "" {
		id = fmt.Sprintf("SEMGREP-%s-%s-%d", checkID, file, match.Start.Line)
	}

	return Finding{
		ID:          id,
		Source:      "semgrep",
		Category:    category,
		Severity:    severity,
		Title:       fmt.Sprintf("Semgrep finding: %s", checkID),
		Description: description,
		File:        file,
		Line:        match.Start.Line,
		Confidence:  0.90,
		Reference:   checkID,
		Code:        code,
	}
}

func mapSemgrepSeverity(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "ERROR":
		return "high"
	case "WARNING":
		return "medium"
	case "INFO":
		return "low"
	default:
		return "medium"
	}
}

func semgrepSeverityRank(value string) int {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "ERROR":
		return 3
	case "WARNING":
		return 2
	case "INFO":
		return 1
	default:
		return 2
	}
}

func semgrepMeetsThreshold(severity, threshold string) bool {
	return semgrepSeverityRank(severity) >= semgrepSeverityRank(threshold)
}

func semgrepCategory(metadata map[string]interface{}) string {
	if metadata == nil {
		return "security"
	}
	for _, key := range []string{"category", "vulnerability_class", "technology"} {
		if value, ok := metadata[key]; ok {
			if category := strings.TrimSpace(fmt.Sprint(value)); category != "" {
				return mapSemgrepMetadataCategory(category)
			}
		}
	}
	return "security"
}

func mapSemgrepMetadataCategory(category string) string {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "maintainability", "best-practice", "correctness":
		return "code_quality"
	case "performance":
		return "performance"
	default:
		return "security"
	}
}

func buildSemgrepDescription(match semgrepMatch, checkID string) string {
	parts := []string{fmt.Sprintf("check_id: %s", checkID)}
	if msg := strings.TrimSpace(match.Extra.Message); msg != "" {
		parts = append(parts, msg)
	}
	if match.Extra.Metadata != nil {
		if cwe := metadataStringList(match.Extra.Metadata, "cwe"); len(cwe) > 0 {
			parts = append(parts, "CWE: "+strings.Join(cwe, ", "))
		}
		if owasp := metadataStringList(match.Extra.Metadata, "owasp"); len(owasp) > 0 {
			parts = append(parts, "OWASP: "+strings.Join(owasp, ", "))
		}
	}
	return strings.Join(parts, "; ")
}

func metadataStringList(metadata map[string]interface{}, key string) []string {
	raw, ok := metadata[key]
	if !ok || raw == nil {
		return nil
	}
	switch values := raw.(type) {
	case string:
		if strings.TrimSpace(values) == "" {
			return nil
		}
		return []string{values}
	case []interface{}:
		out := make([]string, 0, len(values))
		for _, item := range values {
			if s := strings.TrimSpace(fmt.Sprint(item)); s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		if s := strings.TrimSpace(fmt.Sprint(values)); s != "" {
			return []string{s}
		}
	}
	return nil
}

func semgrepLinesSnippet(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return single
	}
	var lines []string
	if err := json.Unmarshal(raw, &lines); err == nil {
		return strings.Join(lines, "\n")
	}
	return ""
}

func boundSemgrepSnippet(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= semgrepMaxCodeSnippetLen {
		return value
	}
	return value[:semgrepMaxCodeSnippetLen] + "..."
}
