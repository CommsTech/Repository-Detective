package scanners

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"
)

// Status describes the outcome of a single external scanner run.
type Status string

const (
	StatusDisabled            Status = "disabled"
	StatusBinaryMissing       Status = "binary_missing"
	StatusClean               Status = "clean"
	StatusFound               Status = "found"
	StatusFailed              Status = "failed"
	StatusTimedOut            Status = "timed_out"
	StatusParseFailed         Status = "parse_failed"
	StatusNoSupportedManifest Status = "no_supported_manifest"
	StatusScannerUnavailable  Status = "scanner_unavailable"
)

// RunResult is the normalized outcome for one scanner.
type RunResult struct {
	Scanner             string
	Status              Status
	Findings            []Finding
	Detail              string
	ApplicabilityReason string
}

// RunSummary aggregates scanner outcomes for one workspace scan.
type RunSummary struct {
	Results []RunResult
}

// Candidates converts all findings into pipeline candidates.
func (s RunSummary) Candidates() []Finding {
	var all []Finding
	for _, result := range s.Results {
		all = append(all, result.Findings...)
	}
	return all
}

// LogResults writes a concise status line per scanner.
func (s RunSummary) LogResults(logger *logrus.Logger, scanID string) {
	entry := logrus.NewEntry(logger)
	if scanID != "" {
		entry = logger.WithField("scan_id", scanID)
	}
	for _, result := range s.Results {
		entry.Infof("[SCANNER:%s] status=%s findings=%d detail=%q",
			result.Scanner, result.Status, len(result.Findings), result.Detail)
	}
}

func resultWithFindings(scanner string, findings []Finding) RunResult {
	status := StatusClean
	if len(findings) > 0 {
		status = StatusFound
	}
	return RunResult{
		Scanner:  scanner,
		Status:   status,
		Findings: findings,
	}
}

func classifyCommandError(err error) Status {
	if err == nil {
		return StatusClean
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return StatusTimedOut
	}
	var exitErr *commandExitError
	if errors.As(err, &exitErr) {
		if exitErr.timedOut {
			return StatusTimedOut
		}
		return StatusFailed
	}
	return StatusFailed
}
