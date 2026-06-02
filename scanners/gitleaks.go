package scanners

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type gitleaksFinding struct {
	RuleID      string  `json:"RuleID"`
	Description string  `json:"Description"`
	StartLine   int     `json:"StartLine"`
	EndLine     int     `json:"EndLine"`
	Match       string  `json:"Match"`
	Secret      string  `json:"Secret"`
	File        string  `json:"File"`
	Commit      string  `json:"Commit"`
	Entropy     float64 `json:"Entropy"`
	Fingerprint string  `json:"Fingerprint"`
}

func init() {
	RegisterDeterministicSource("gitleaks")
}

// RunGitleaks scans a workspace directory for secrets using gitleaks dir mode (filesystem snapshot, no git history).
func RunGitleaks(ctx context.Context, logger *logrus.Logger, dir string, cfg Config) RunResult {
	return runGitleaksWithCommand(ctx, logger, dir, cfg, "gitleaks")
}

func runGitleaksWithCommand(ctx context.Context, logger *logrus.Logger, dir string, cfg Config, commandName string) RunResult {
	result := RunResult{Scanner: "gitleaks"}
	if !commandAvailable(commandName) {
		logger.Warn("[SCANNER:gitleaks] binary not found — install gitleaks or use the official Bugbot Docker image")
		result.Status = StatusBinaryMissing
		return result
	}

	args := gitleaksArgs(dir, cfg)
	timeout := gitleaksTimeout(cfg)
	output, err := runCommand(ctx, timeout, dir, commandName, args...)
	if err != nil && len(output) == 0 {
		result.Status = classifyCommandError(err)
		result.Detail = err.Error()
		logger.Warnf("[SCANNER:gitleaks] scan failed: status=%s err=%v", result.Status, err)
		return result
	}

	findings, parseErr := parseGitleaksOutput(output, dir)
	if parseErr != nil {
		result.Status = StatusParseFailed
		result.Detail = parseErr.Error()
		logger.Warnf("[SCANNER:gitleaks] failed to parse output: %v", parseErr)
		return result
	}

	result = resultWithFindings("gitleaks", findings)
	logger.Infof("[SCANNER:gitleaks] status=%s findings=%d", result.Status, len(findings))
	return result
}

func gitleaksArgs(dir string, cfg Config) []string {
	args := []string{
		"dir",
		dir,
		"--report-format", "json",
		"--report-path", "-",
		"--no-banner",
		"--redact",
	}
	if strings.TrimSpace(cfg.GitleaksConfig) != "" {
		args = append(args, "--config", cfg.GitleaksConfig)
	}
	return args
}

func gitleaksTimeout(cfg Config) time.Duration {
	seconds := cfg.GitleaksTimeoutSeconds
	if seconds <= 0 {
		seconds = cfg.TimeoutSeconds
	}
	return time.Duration(seconds) * time.Second
}

func parseGitleaksOutput(output []byte, dir string) ([]Finding, error) {
	trimmed := strings.TrimSpace(string(stripANSI(output)))
	if trimmed == "" {
		return nil, nil
	}

	payload, err := extractJSONArray(output)
	if err != nil {
		return nil, err
	}

	var report []gitleaksFinding
	if err := json.Unmarshal(payload, &report); err != nil {
		return nil, err
	}

	findings := make([]Finding, 0, len(report))
	for _, item := range report {
		findings = append(findings, gitleaksFindingToFinding(item, dir))
	}
	return findings, nil
}

func gitleaksFindingToFinding(item gitleaksFinding, dir string) Finding {
	file := strings.TrimPrefix(item.File, dir)
	file = strings.TrimPrefix(file, "/")
	file = strings.TrimPrefix(file, "\\")

	ruleID := strings.TrimSpace(item.RuleID)
	if ruleID == "" {
		ruleID = "unknown-rule"
	}

	title := fmt.Sprintf("Potential secret detected by gitleaks: %s", ruleID)
	if desc := strings.TrimSpace(item.Description); desc != "" {
		title = fmt.Sprintf("%s (%s)", title, desc)
	}

	description := buildGitleaksDescription(item)
	evidence := gitleaksRedactedEvidence(item)

	id := fmt.Sprintf("GITLEAKS-%s", ruleID)
	if item.Fingerprint != "" {
		id = fmt.Sprintf("GITLEAKS-%s", item.Fingerprint)
	} else if file != "" {
		id = fmt.Sprintf("GITLEAKS-%s-%s-%d", ruleID, file, item.StartLine)
	}

	return Finding{
		ID:          id,
		Source:      "gitleaks",
		Category:    "secret",
		Severity:    "high",
		Title:       title,
		Description: description,
		File:        file,
		Line:        item.StartLine,
		Confidence:  0.95,
		Reference:   firstNonEmpty(item.Fingerprint, ruleID),
		Code:        evidence,
	}
}

func buildGitleaksDescription(item gitleaksFinding) string {
	parts := []string{fmt.Sprintf("Rule ID: %s", item.RuleID)}
	if item.Entropy > 0 {
		parts = append(parts, fmt.Sprintf("entropy: %.2f", item.Entropy))
	}
	if commit := strings.TrimSpace(item.Commit); commit != "" && commit != "0000000000000000" {
		parts = append(parts, fmt.Sprintf("commit: %s", commit))
	}
	if desc := strings.TrimSpace(item.Description); desc != "" {
		parts = append(parts, desc)
	}
	evidence := gitleaksRedactedEvidence(item)
	if evidence != "" {
		parts = append(parts, fmt.Sprintf("match: %s", evidence))
	}
	return strings.Join(parts, "; ")
}

func gitleaksRedactedEvidence(item gitleaksFinding) string {
	for _, candidate := range []string{item.Secret, item.Match} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		return candidate
	}
	return ""
}
