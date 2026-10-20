package scanners

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// ParseTrivyOutputForTest exposes trivy JSON parsing for unit tests.
func ParseTrivyOutputForTest(output []byte, dir string) ([]Finding, error) {
	return parseTrivyOutput(output, dir)
}

// CommandAvailableForTest exposes binary lookup for tests.
func CommandAvailableForTest(name string) bool {
	return commandAvailable(name)
}

// ParseGitleaksOutputForTest exposes gitleaks JSON parsing for unit tests.
func ParseGitleaksOutputForTest(output []byte, dir string) ([]Finding, error) {
	return parseGitleaksOutput(output, dir)
}

// ParseGitleaksScanOutputForTest exposes report-file-first gitleaks parsing for unit tests.
func ParseGitleaksScanOutputForTest(reportBytes, commandOutput []byte, dir string) ([]Finding, error) {
	return parseGitleaksScanOutput(reportBytes, commandOutput, dir)
}

// ParseSemgrepOutputForTest exposes Semgrep JSON parsing for unit tests.
func ParseSemgrepOutputForTest(output []byte, dir string, cfg Config) (semgrepParseResult, error) {
	return parseSemgrepOutput(output, dir, cfg)
}

// MapSemgrepSeverityForTest exposes Semgrep severity mapping for unit tests.
func MapSemgrepSeverityForTest(value string) string {
	return mapSemgrepSeverity(value)
}

// RunSemgrepWithCommandForTest runs the Semgrep scan path using a substitute command name.
func RunSemgrepWithCommandForTest(ctx context.Context, logger *logrus.Logger, dir string, cfg Config, commandName string, commandArgs ...string) RunResult {
	result := RunResult{Scanner: "semgrep"}
	if !commandAvailable(commandName) {
		result.Status = StatusBinaryMissing
		return result
	}

	timeout := semgrepTimeout(cfg)
	output, err := runCommand(ctx, timeout, dir, commandName, commandArgs...)
	if err != nil && len(output) == 0 {
		result.Status = classifyCommandError(err)
		result.Detail = err.Error()
		return result
	}

	parsed, parseErr := parseSemgrepOutput(output, dir, cfg)
	if parseErr != nil {
		result.Status = StatusParseFailed
		result.Detail = parseErr.Error()
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
	return result
}

// AnnotateHistoryFindingsForTest exposes history finding annotation for unit tests.
func AnnotateHistoryFindingsForTest(findings []Finding, scope, currentTreeDir string, redact bool) []Finding {
	return annotateHistoryFindings(findings, scope, currentTreeDir, redact)
}

// GitleaksHistoryArgsForTest exposes gitleaks detect argv for unit tests.
func GitleaksHistoryArgsForTest(gitDir string, cfg Config, scope, reportPath string) []string {
	return gitleaksHistoryArgs(gitDir, cfg, scope, reportPath)
}

// RunGitleaksGitHistoryWithCommandForTest runs git-history gitleaks detect with a substitute command.
func RunGitleaksGitHistoryWithCommandForTest(ctx context.Context, logger *logrus.Logger, gitDir string, cfg Config, scope, currentTreeDir, commandName string, commandArgs ...string) RunResult {
	return runGitleaksGitHistoryWithCommand(ctx, logger, gitDir, cfg, scope, currentTreeDir, commandName)
}

// RunGitleaksWithCommandForTest runs the gitleaks scan path using a substitute command name.
func RunGitleaksWithCommandForTest(ctx context.Context, logger *logrus.Logger, dir string, cfg Config, commandName string, commandArgs ...string) RunResult {
	result := RunResult{Scanner: "gitleaks"}
	if !commandAvailable(commandName) {
		result.Status = StatusBinaryMissing
		return result
	}

	timeout := gitleaksTimeout(cfg)
	output, err := runCommand(ctx, timeout, dir, commandName, commandArgs...)
	if err != nil && len(output) == 0 {
		result.Status = classifyCommandError(err)
		result.Detail = err.Error()
		return result
	}

	findings, parseErr := parseGitleaksOutput(output, dir)
	if parseErr != nil {
		result.Status = StatusParseFailed
		result.Detail = parseErr.Error()
		return result
	}

	return resultWithFindings("gitleaks", findings)
}

// RunHadolintWithCommandForTest runs hadolint scan path with a substitute command.
func RunHadolintWithCommandForTest(ctx context.Context, logger *logrus.Logger, dir string, entries []FileEntry, cfg Config, commandName string) RunResult {
	return runHadolintWithCommand(ctx, logger, dir, entries, cfg, commandName)
}

// RunCheckovWithCommandForTest runs checkov scan path with a substitute command.
func RunCheckovWithCommandForTest(ctx context.Context, logger *logrus.Logger, dir string, entries []FileEntry, cfg Config, commandName string) RunResult {
	return runCheckovWithCommand(ctx, logger, dir, entries, cfg, commandName)
}

// RunGovulncheckWithCommandForTest runs govulncheck scan path with a substitute command.
func RunGovulncheckWithCommandForTest(ctx context.Context, logger *logrus.Logger, dir string, cfg Config, commandName string) RunResult {
	return runGovulncheckWithCommand(ctx, logger, dir, cfg, commandName)
}

// RunGosecWithCommandForTest runs gosec scan path with a substitute command.
func RunGosecWithCommandForTest(ctx context.Context, logger *logrus.Logger, dir string, cfg Config, commandName string) RunResult {
	return runGosecWithCommand(ctx, logger, dir, cfg, commandName)
}

// RunStaticcheckWithCommandForTest runs staticcheck scan path with a substitute command.
func RunStaticcheckWithCommandForTest(ctx context.Context, logger *logrus.Logger, dir string, cfg Config, commandName string) RunResult {
	return runStaticcheckWithCommand(ctx, logger, dir, cfg, commandName)
}

// CapFindingsForTest exposes finding cap logic for unit tests.
func CapFindingsForTest(findings []Finding, max int) cappedFindings {
	return capFindings(findings, max)
}

// MapGosecSeverityForTest exposes gosec severity mapping.
func MapGosecSeverityForTest(value string) string {
	return mapGosecSeverity(value)
}

// MapGosecConfidenceForTest exposes gosec confidence mapping.
func MapGosecConfidenceForTest(value string) float64 {
	return mapGosecConfidence(value)
}

// MapStaticcheckCodeForTest exposes staticcheck code mapping.
func MapStaticcheckCodeForTest(code string) (category, severity string, confidence float64) {
	return mapStaticcheckCode(code)
}

// RunTrivyWithCommandForTest runs the trivy scan path using a substitute command name.
func RunTrivyWithCommandForTest(ctx context.Context, logger *logrus.Logger, dir string, cfg Config, commandName string, commandArgs ...string) RunResult {
	result := RunResult{Scanner: "trivy"}
	if !commandAvailable(commandName) {
		result.Status = StatusBinaryMissing
		return result
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	output, err := runCommand(ctx, timeout, dir, commandName, commandArgs...)
	if err != nil {
		if len(output) == 0 {
			result.Status = classifyCommandError(err)
			result.Detail = err.Error()
			return result
		}
	}

	findings, parseErr := parseTrivyOutput(output, dir)
	if parseErr != nil {
		result.Status = StatusParseFailed
		result.Detail = parseErr.Error()
		return result
	}

	return resultWithFindings("trivy", findings)
}
