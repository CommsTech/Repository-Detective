package scanners

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
		logger.Warn("[SCANNER:gitleaks] binary not found — install gitleaks or use the official Repository Detective Docker image")
		result.Status = StatusBinaryMissing
		return result
	}

	reportFile, err := os.CreateTemp("", "gitleaks-report-*.json")
	if err != nil {
		result.Status = StatusFailed
		result.Detail = fmt.Sprintf("create report file: %v", err)
		return result
	}
	reportPath := reportFile.Name()
	_ = reportFile.Close()
	defer func() { _ = os.Remove(reportPath) }()

	args := gitleaksArgs(dir, cfg, reportPath)
	timeout := gitleaksTimeout(cfg)
	output, err := runCommand(ctx, timeout, dir, commandName, args...)

	reportBytes, readErr := os.ReadFile(filepath.Clean(reportPath)) //nosec G304 -- reportPath from os.CreateTemp in this function
	if readErr != nil {
		reportBytes = nil
	}
	// gitleaks exits non-zero when it finds leaks, so a command error only means the
	// scan failed when it also produced no report to parse.
	if err != nil && strings.TrimSpace(string(stripANSI(reportBytes))) == "" {
		result.Status = classifyCommandError(err)
		result.Detail = gitleaksFailureDetail(err, output)
		logger.Warnf("[SCANNER:gitleaks] scan failed: status=%s detail=%s", result.Status, result.Detail)
		return result
	}

	findings, parseErr := parseGitleaksScanOutput(reportBytes, output, dir)
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

func gitleaksArgs(dir string, cfg Config, reportPath string) []string {
	args := []string{
		"dir",
		dir,
		"--report-format", "json",
		"--report-path", reportPath,
		"--no-banner",
		"--no-color",
		"--redact",
		"--log-level", "error",
	}
	if cfgPath := resolveGitleaksConfig(cfg.GitleaksConfig); cfgPath != "" {
		args = append(args, "--config", cfgPath)
	}
	return args
}

// resolveGitleaksConfig makes the allowlist path absolute before it reaches gitleaks.
// The scanner runs with the scan workspace as its working directory, so a relative
// path like "config/gitleaks.toml" resolves inside the repository under scan, and
// gitleaks aborts the whole run when it cannot load it. An unusable path is dropped
// so the scan falls back to the default rules instead of failing.
func resolveGitleaksConfig(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return ""
		}
		path = abs
	}
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}

// gitleaksFailureDetail keeps the tool's own output next to the exit status so a
// config load failure is distinguishable from a timeout or a crash.
func gitleaksFailureDetail(err error, output []byte) string {
	detail := err.Error()
	if msg := redactScannerDetail(string(output)); msg != "" {
		detail = fmt.Sprintf("%s: %s", detail, msg)
	}
	return detail
}

func gitleaksTimeout(cfg Config) time.Duration {
	seconds := cfg.GitleaksTimeoutSeconds
	if seconds <= 0 {
		seconds = cfg.TimeoutSeconds
	}
	return time.Duration(seconds) * time.Second
}

// parseGitleaksScanOutput prefers the JSON report file (gitleaks 8.x does not write JSON to stdout for report-path "-").
func parseGitleaksScanOutput(reportBytes, commandOutput []byte, dir string) ([]Finding, error) {
	if strings.TrimSpace(string(stripANSI(reportBytes))) != "" {
		return parseGitleaksOutput(reportBytes, dir)
	}
	findings, err := parseGitleaksOutput(commandOutput, dir)
	// A clean run writes no report and logs only "no leaks found" to stderr. Output
	// with no JSON at all therefore means zero findings, not an unparsable report.
	if errors.Is(err, errNoJSONValue) {
		return nil, nil
	}
	return findings, err
}

func parseGitleaksOutput(output []byte, dir string) ([]Finding, error) {
	clean := stripANSI(output)
	trimmed := strings.TrimSpace(string(clean))
	if trimmed == "" {
		return nil, nil
	}

	payload := []byte(trimmed)
	if trimmed[0] != '[' {
		extracted, err := extractJSONArray(clean)
		if err != nil {
			return nil, err
		}
		payload = extracted
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
	file := normalizeGitleaksPath(item.File, dir)

	ruleID := strings.TrimSpace(item.RuleID)
	if ruleID == "" {
		ruleID = "unknown-rule"
	}
	ruleID = sanitizeScannerRuleID(ruleID)

	title := fmt.Sprintf("Potential secret detected by gitleaks: %s", ruleID)
	if desc := strings.TrimSpace(item.Description); desc != "" {
		title = fmt.Sprintf("%s (%s)", title, desc)
	}

	description := buildGitleaksDescription(item)
	evidence := gitleaksRedactedEvidence(item)

	// Prefer stable rule+path+line IDs. Raw gitleaks fingerprints embed absolute
	// workspace paths (/tmp/rd-archive-*), which created a new RuleID/fingerprint
	// on every scan and flooded the open queue with duplicates.
	id := fmt.Sprintf("GITLEAKS-%s:%s:%d", ruleID, file, item.StartLine)
	if file == "" {
		id = fmt.Sprintf("GITLEAKS-%s:%d", ruleID, item.StartLine)
	}

	desc := description
	if !strings.Contains(desc, "scope:") {
		desc = fmt.Sprintf("%s; scope: %s", desc, secretScopeLabel(SecretScopeCurrentTree))
	}
	return Finding{
		ID:          id,
		Source:      TreeScannerName,
		Category:    "secret",
		Severity:    "high",
		Title:       title,
		Description: desc,
		File:        file,
		Line:        item.StartLine,
		Confidence:  0.95,
		Reference:   firstNonEmpty(sanitizeGitleaksFingerprint(item.Fingerprint, dir), ruleID),
		Code:        evidence,
	}
}

func normalizeGitleaksPath(file, dir string) string {
	file = strings.TrimSpace(file)
	file = strings.ReplaceAll(file, "\\", "/")
	dir = strings.ReplaceAll(strings.TrimSpace(dir), "\\", "/")
	if dir != "" {
		file = strings.TrimPrefix(file, dir)
		file = strings.TrimPrefix(file, dir+"/")
	}
	file = strings.TrimPrefix(file, "/")
	// Strip leftover temp scan prefixes if gitleaks reported an absolute path.
	if idx := strings.Index(file, "/tmp/rd-"); idx >= 0 {
		rest := file[idx+1:]
		if slash := strings.Index(rest, "/"); slash >= 0 {
			file = rest[slash+1:]
		}
	}
	if strings.HasPrefix(file, "tmp/rd-") {
		if slash := strings.Index(file, "/"); slash >= 0 {
			file = file[slash+1:]
		}
	}
	return file
}

func sanitizeGitleaksFingerprint(fp, dir string) string {
	fp = strings.TrimSpace(fp)
	if fp == "" {
		return ""
	}
	fp = strings.ReplaceAll(fp, "\\", "/")
	dir = strings.ReplaceAll(strings.TrimSpace(dir), "\\", "/")
	if dir != "" {
		fp = strings.ReplaceAll(fp, dir+"/", "")
		fp = strings.ReplaceAll(fp, dir, "")
	}
	// Drop absolute temp workspace segments from historical fingerprints.
	for _, marker := range []string{"/tmp/rd-archive-", "/tmp/rd-scan-"} {
		for {
			idx := strings.Index(fp, marker)
			if idx < 0 {
				break
			}
			rest := fp[idx+1:]
			slash := strings.Index(rest, "/")
			if slash < 0 {
				fp = fp[:idx]
				break
			}
			fp = fp[:idx] + rest[slash+1:]
		}
	}
	return strings.Trim(fp, ":/")
}

func sanitizeScannerRuleID(ruleID string) string {
	ruleID = strings.TrimSpace(ruleID)
	ruleID = strings.Trim(ruleID, "`\"'")
	return strings.TrimSpace(ruleID)
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
