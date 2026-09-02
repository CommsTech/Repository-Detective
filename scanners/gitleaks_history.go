package scanners

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// RunGitleaksGitHistory scans git commit history using gitleaks detect mode.
func RunGitleaksGitHistory(ctx context.Context, logger *logrus.Logger, gitDir string, cfg Config, scope string, currentTreeDir string) RunResult {
	return runGitleaksGitHistoryWithCommand(ctx, logger, gitDir, cfg, scope, currentTreeDir, "gitleaks")
}

func runGitleaksGitHistoryWithCommand(ctx context.Context, logger *logrus.Logger, gitDir string, cfg Config, scope, currentTreeDir, commandName string) RunResult {
	result := RunResult{Scanner: HistoryScannerName, Detail: secretScopeLabel(scope)}
	if !commandAvailable(commandName) {
		logger.Warn("[SCANNER:gitleaks-history] binary not found")
		result.Status = StatusBinaryMissing
		return result
	}
	if strings.TrimSpace(gitDir) == "" {
		result.Status = StatusFailed
		result.Detail = "git workspace not available for history scan"
		return result
	}

	reportFile, err := os.CreateTemp("", "gitleaks-history-report-*.json")
	if err != nil {
		result.Status = StatusFailed
		result.Detail = fmt.Sprintf("create report file: %v", err)
		return result
	}
	reportPath := reportFile.Name()
	_ = reportFile.Close()
	defer func() { _ = os.Remove(reportPath) }()

	args := gitleaksHistoryArgs(gitDir, cfg, scope, reportPath)
	timeout := gitleaksHistoryTimeout(cfg)
	output, err := runCommand(ctx, timeout, gitDir, commandName, args...)

	reportBytes, readErr := os.ReadFile(filepath.Clean(reportPath)) //nosec G304 -- reportPath from os.CreateTemp
	if readErr != nil {
		reportBytes = nil
	}
	parseInput := reportBytes
	if strings.TrimSpace(string(reportBytes)) == "" {
		parseInput = output
	}
	if err != nil && len(parseInput) == 0 {
		result.Status = classifyCommandError(err)
		if result.Status == StatusTimedOut {
			result.Detail = fmt.Sprintf("git history secret scan timed out after %s (%s)", timeout, secretScopeLabel(scope))
		} else {
			result.Detail = fmt.Sprintf("%s: %v", secretScopeLabel(scope), err)
		}
		logger.Warnf("[SCANNER:gitleaks-history] scan failed: status=%s err=%v", result.Status, err)
		return result
	}

	findings, parseErr := parseGitleaksScanOutput(reportBytes, output, gitDir)
	if parseErr != nil {
		result.Status = StatusParseFailed
		result.Detail = parseErr.Error()
		return result
	}

	findings = annotateHistoryFindings(findings, scope, currentTreeDir, cfg.SecretScanRedact)
	result = resultWithFindings(HistoryScannerName, findings)
	result.Detail = secretScopeLabel(scope)
	logger.Infof("[SCANNER:gitleaks-history] status=%s scope=%s findings=%d", result.Status, scope, len(findings))
	return result
}

func gitleaksHistoryArgs(gitDir string, cfg Config, scope, reportPath string) []string {
	args := []string{
		"detect",
		"--source", gitDir,
		"--report-format", "json",
		"--report-path", reportPath,
		"--no-banner",
		"--no-color",
		"--log-level", "error",
	}
	if cfg.SecretScanRedact {
		args = append(args, "--redact")
	}
	if cfgPath := resolveGitleaksConfig(cfg.GitleaksConfig); cfgPath != "" {
		args = append(args, "--config", cfgPath)
	}
	logOpts := historyLogOpts(cfg, scope)
	if logOpts != "" {
		args = append(args, "--log-opts", logOpts)
	}
	return args
}

func historyLogOpts(cfg Config, scope string) string {
	switch scope {
	case SecretScopeRecentCommits:
		if cfg.SecretScanRecentCommitsMax > 0 {
			return fmt.Sprintf("-n %d", cfg.SecretScanRecentCommitsMax)
		}
		if cfg.SecretScanHistoryMaxCommits > 0 {
			return fmt.Sprintf("-n %d", cfg.SecretScanHistoryMaxCommits)
		}
	case SecretScopeChangedFiles:
		if cfg.SecretScanRecentCommitsMax > 0 {
			return fmt.Sprintf("-n %d", cfg.SecretScanRecentCommitsMax)
		}
	}
	return ""
}

func gitleaksHistoryTimeout(cfg Config) time.Duration {
	seconds := cfg.SecretScanHistoryTimeoutSeconds
	if seconds <= 0 {
		seconds = cfg.GitleaksTimeoutSeconds
	}
	if seconds <= 0 {
		seconds = cfg.TimeoutSeconds
	}
	if seconds <= 0 {
		seconds = 600
	}
	return time.Duration(seconds) * time.Second
}

// GitleaksHistoryBudget returns the context timeout for git history secret scans.
func GitleaksHistoryBudget(cfg Config) time.Duration {
	return gitleaksHistoryTimeout(cfg)
}

func annotateHistoryFindings(findings []Finding, scope, currentTreeDir string, redact bool) []Finding {
	out := make([]Finding, 0, len(findings))
	for _, f := range findings {
		inTree := fileExistsInTree(currentTreeDir, f.File)
		commit := extractCommitFromDescription(f.Description)
		f.Source = HistoryScannerName
		f.Category = "secret"
		if !strings.HasPrefix(f.ID, "GITLEAKS-HIST-") {
			f.ID = "GITLEAKS-HIST-" + strings.TrimPrefix(f.ID, "GITLEAKS-")
		}
		if inTree {
			f.Severity = "critical"
		}
		scopeNote := secretScopeLabel(scope)
		treeNote := "not in current tree"
		if inTree {
			treeNote = "also present in current tree"
		}
		f.Description = fmt.Sprintf("%s; scope: %s; %s; %s",
			strings.TrimSuffix(f.Description, "; "+RemediationRotateGuidance),
			scopeNote, treeNote, RemediationRotateGuidance)
		if commit != "" {
			f.Reference = commit
		}
		if redact {
			f.Code = redactSecretEvidence(f.Code)
			f.Description = redactDescriptionMatches(f.Description)
		}
		out = append(out, f)
	}
	return out
}

func extractCommitFromDescription(desc string) string {
	for _, part := range strings.Split(desc, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "commit: ") {
			return strings.TrimPrefix(part, "commit: ")
		}
	}
	return ""
}

func redactSecretEvidence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "…" + s[len(s)-4:]
}

func redactDescriptionMatches(desc string) string {
	parts := strings.Split(desc, "; ")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "match: ") {
			parts[i] = "match: " + redactSecretEvidence(strings.TrimSpace(strings.TrimPrefix(part, "match: ")))
		}
	}
	return strings.Join(parts, "; ")
}
