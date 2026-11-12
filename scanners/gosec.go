package scanners

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

func init() {
	RegisterDeterministicSource("gosec")
}

type gosecReport struct {
	Issues []gosecIssue `json:"Issues"`
}

type gosecIssue struct {
	Severity   string `json:"severity"`
	Confidence string `json:"confidence"`
	RuleID     string `json:"rule_id"`
	Details    string `json:"details"`
	File       string `json:"file"`
	Line       string `json:"line"`
	Code       string `json:"code"`
}

// RunGosec scans Go code for security issues using gosec.
func RunGosec(ctx context.Context, logger *logrus.Logger, dir string, cfg Config) RunResult {
	return runGosecWithCommand(ctx, logger, dir, cfg, "gosec")
}

func runGosecWithCommand(ctx context.Context, logger *logrus.Logger, dir string, cfg Config, commandName string) RunResult {
	result := RunResult{Scanner: "gosec"}
	if !commandAvailable(commandName) {
		if logger != nil {
			logger.Warn("[SCANNER:gosec] binary not found — install gosec (go install github.com/securego/gosec/v2/cmd/gosec@latest)")
		}
		result.Status = StatusBinaryMissing
		return result
	}

	timeout := scannerTimeoutSeconds(cfg.GosecTimeoutSeconds, cfg.TimeoutSeconds)
	stdout, stderr, err := runCommandStreams(ctx, timeout, dir, commandName, "-fmt=json", "./...")
	output := gosecParseOutput(stdout, stderr)
	if err != nil && len(bytes.TrimSpace(output)) == 0 {
		result.Status = classifyCommandError(err)
		result.Detail = err.Error()
		return result
	}

	parsed, parseErr := parseGosecOutput(output, dir, cfg)
	if parseErr != nil && len(parsed.Findings) == 0 {
		result.Status = StatusParseFailed
		result.Detail = parseErr.Error()
		return result
	}

	result = resultWithFindings("gosec", parsed.Findings)
	if parsed.Truncated {
		max := goScannerMaxFindings(cfg)
		result.Detail = truncateDetailForScanner("findings", max, parsed.Total)
	}
	if logger != nil {
		logResultInfo(logger, "gosec", result.Status, len(parsed.Findings), result.Detail)
	}
	return result
}

// gosecParseOutput prefers a JSON report on stdout. When gosec exits early it
// often leaves stdout empty and writes `[gosec]` logs (and sometimes the report)
// on stderr. Those logs must be searched too, or the first `[` is parsed as JSON.
func gosecParseOutput(stdout, stderr []byte) []byte {
	if _, err := extractJSONObject(stdout); err == nil {
		return stdout
	}
	if len(bytes.TrimSpace(stderr)) == 0 {
		return stdout
	}
	merged := make([]byte, 0, len(stdout)+len(stderr)+1)
	merged = append(merged, stdout...)
	merged = append(merged, '\n')
	merged = append(merged, stderr...)
	return merged
}

func parseGosecOutput(output []byte, dir string, cfg Config) (cappedFindings, error) {
	payload, err := extractJSONObject(output)
	if err != nil {
		if errors.Is(err, errNoJSONValue) {
			line := strings.TrimSpace(string(output))
			if i := strings.IndexByte(line, '\n'); i >= 0 {
				line = line[:i]
			}
			if len(line) > 240 {
				line = line[:240]
			}
			if line == "" {
				return cappedFindings{}, err
			}
			return cappedFindings{}, fmt.Errorf("gosec produced no JSON: %s", line)
		}
		return cappedFindings{}, err
	}
	var report gosecReport
	if err := json.Unmarshal(payload, &report); err != nil {
		return cappedFindings{}, fmt.Errorf("parse gosec json: %w", err)
	}
	findings := make([]Finding, 0, len(report.Issues))
	for _, issue := range report.Issues {
		findings = append(findings, gosecFinding(issue, dir))
	}
	capped := capFindings(findings, goScannerMaxFindings(cfg))
	return capped, nil
}

func gosecFinding(issue gosecIssue, dir string) Finding {
	line, _ := strconv.Atoi(strings.TrimSpace(issue.Line))
	ruleID := strings.TrimSpace(issue.RuleID)
	if ruleID == "" {
		ruleID = "GOSEC"
	}
	title := ruleID
	if issue.Details != "" {
		title = issue.Details
	}
	return Finding{
		Source:      "gosec",
		Category:    "security",
		Severity:    mapGosecSeverity(issue.Severity),
		Title:       title,
		Description: trimEvidence(issue.Details),
		File:        relPath(dir, issue.File),
		Line:        line,
		Code:        trimEvidence(issue.Code),
		Confidence:  mapGosecConfidence(issue.Confidence),
		Reference:   ruleID,
		ID:          ruleID,
	}
}

func mapGosecSeverity(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "HIGH":
		return "high"
	case "LOW":
		return "low"
	case "MEDIUM":
		return "medium"
	default:
		return "medium"
	}
}

func mapGosecConfidence(value string) float64 {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "HIGH":
		return 0.90
	case "LOW":
		return 0.65
	case "MEDIUM":
		return 0.80
	default:
		return 0.80
	}
}

func ParseGosecOutputForTest(output []byte, dir string, cfg Config) (cappedFindings, error) {
	return parseGosecOutput(output, dir, cfg)
}
