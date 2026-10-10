package scanners

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

func init() {
	RegisterDeterministicSource("hadolint")
}

type hadolintIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	File    string `json:"file"`
	Level   string `json:"level"`
}

// RunHadolint scans Dockerfiles using hadolint.
func RunHadolint(ctx context.Context, logger *logrus.Logger, dir string, entries []FileEntry, cfg Config) RunResult {
	return runHadolintWithCommand(ctx, logger, dir, entries, cfg, "hadolint")
}

func runHadolintWithCommand(ctx context.Context, logger *logrus.Logger, dir string, entries []FileEntry, cfg Config, commandName string) RunResult {
	result := RunResult{Scanner: "hadolint"}
	paths := CollectDockerfilePaths(dir, entries)
	if len(paths) == 0 {
		result.Status = StatusClean
		result.Detail = "no Dockerfiles"
		return result
	}
	if !commandAvailable(commandName) {
		if logger != nil {
			logger.Warn("[SCANNER:hadolint] binary not found — install hadolint from your package manager or release binary")
		}
		result.Status = StatusBinaryMissing
		return result
	}

	timeout := scannerTimeoutSeconds(cfg.HadolintTimeoutSeconds, cfg.TimeoutSeconds)
	args := append([]string{"--format", "json"}, paths...)
	output, err := runCommand(ctx, timeout, dir, commandName, args...)
	if err != nil && len(output) == 0 {
		result.Status = classifyCommandError(err)
		result.Detail = err.Error()
		return result
	}

	parsed, parseErr := parseHadolintOutput(output, dir, cfg)
	if parseErr != nil && len(parsed.Findings) == 0 {
		result.Status = StatusParseFailed
		result.Detail = parseErr.Error()
		return result
	}

	result = resultWithFindings("hadolint", parsed.Findings)
	if parsed.Truncated {
		max := iacScannerMaxFindings(cfg)
		result.Detail = truncateDetailForScanner("findings", max, parsed.Total)
	}
	if logger != nil {
		logResultInfo(logger, "hadolint", result.Status, len(parsed.Findings), result.Detail)
	}
	return result
}

func parseHadolintOutput(output []byte, dir string, cfg Config) (cappedFindings, error) {
	payload := output
	if raw, err := extractJSONArray(output); err == nil {
		payload = raw
	}
	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" || trimmed == "[]" || trimmed == "null" {
		return cappedFindings{}, nil
	}

	var issues []hadolintIssue
	if err := json.Unmarshal(payload, &issues); err != nil {
		return cappedFindings{}, fmt.Errorf("parse hadolint json: %w", err)
	}
	findings := make([]Finding, 0, len(issues))
	for _, issue := range issues {
		findings = append(findings, hadolintFinding(issue, dir))
	}
	return capFindings(findings, iacScannerMaxFindings(cfg)), nil
}

func hadolintFinding(issue hadolintIssue, dir string) Finding {
	ruleID := strings.TrimSpace(issue.Code)
	if ruleID == "" {
		ruleID = "HADOLINT"
	}
	title := issue.Message
	if title == "" {
		title = ruleID
	}
	evidence := trimEvidence(firstNonEmpty(issue.Message, ruleID))
	return Finding{
		Source:      "hadolint",
		Category:    "container",
		Severity:    mapHadolintSeverity(issue.Level),
		Title:       title,
		Description: evidence,
		File:        relPath(dir, issue.File),
		Line:        issue.Line,
		Code:        evidence,
		Confidence:  0.90,
		Reference:   ruleID,
		ID:          ruleID,
	}
}

func mapHadolintSeverity(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
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

func ParseHadolintOutputForTest(output []byte, dir string, cfg Config) (cappedFindings, error) {
	return parseHadolintOutput(output, dir, cfg)
}

func MapHadolintSeverityForTest(level string) string {
	return mapHadolintSeverity(level)
}
