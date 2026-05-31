package scanners

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type grypeReport struct {
	Matches []grypeMatch `json:"matches"`
}

type grypeMatch struct {
	Vulnerability grypeVulnerability `json:"vulnerability"`
	Artifact      grypeArtifact      `json:"artifact"`
}

type grypeVulnerability struct {
	ID          string   `json:"id"`
	Severity    string   `json:"severity"`
	Description string   `json:"description"`
	URLs        []string `json:"urls"`
}

type grypeArtifact struct {
	Name      string          `json:"name"`
	Version   string          `json:"version"`
	Locations []grypeLocation `json:"locations"`
}

type grypeLocation struct {
	Path string `json:"path"`
}

// RunGrype scans a workspace directory with Grype.
func RunGrype(ctx context.Context, logger *logrus.Logger, dir string, cfg Config) RunResult {
	result := RunResult{Scanner: "grype"}
	if !commandAvailable("grype") {
		logger.Warn("[SCANNER:grype] binary not found — install grype or use the official Bugbot Docker image")
		result.Status = StatusBinaryMissing
		return result
	}

	args := []string{
		"dir:" + dir,
		"-o", "json",
		"--quiet",
	}
	if cfg.GrypeFailOn != "" {
		args = append(args, "--fail-on", cfg.GrypeFailOn)
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	output, err := runCommand(ctx, timeout, dir, "grype", args...)
	if err != nil && len(output) == 0 {
		result.Status = classifyCommandError(err)
		result.Detail = err.Error()
		logger.Warnf("[SCANNER:grype] scan failed: status=%s err=%v", result.Status, err)
		return result
	}

	findings, parseErr := parseGrypeOutput(output, dir, cfg)
	if parseErr != nil {
		result.Status = StatusParseFailed
		result.Detail = parseErr.Error()
		logger.Warnf("[SCANNER:grype] failed to parse output: %v", parseErr)
		return result
	}

	result = resultWithFindings("grype", findings)
	logger.Infof("[SCANNER:grype] status=%s findings=%d", result.Status, len(findings))
	return result
}

func parseGrypeOutput(output []byte, dir string, cfg Config) ([]Finding, error) {
	var report grypeReport
	if err := json.Unmarshal(output, &report); err != nil {
		return nil, err
	}

	minSeverity := cfg.GrypeFailOn
	if minSeverity == "" {
		minSeverity = "high"
	}

	var findings []Finding
	for _, match := range report.Matches {
		severity := normalizeSeverity(match.Vulnerability.Severity)
		if !meetsMinSeverity(severity, minSeverity) {
			continue
		}

		file := ""
		if len(match.Artifact.Locations) > 0 {
			file = strings.TrimPrefix(match.Artifact.Locations[0].Path, dir)
			file = strings.TrimPrefix(file, "/")
		}

		title := match.Vulnerability.ID
		if match.Artifact.Name != "" {
			title = fmt.Sprintf("%s in %s", match.Vulnerability.ID, match.Artifact.Name)
		}

		findings = append(findings, Finding{
			ID:          fmt.Sprintf("GRYPE-%s", match.Vulnerability.ID),
			Source:      "grype",
			Category:    "dependency_vulnerability",
			Severity:    severity,
			Title:       title,
			Description: match.Vulnerability.Description,
			File:        file,
			Confidence:  0.97,
			Reference:   match.Vulnerability.ID,
			Code:        fmt.Sprintf("%s@%s", match.Artifact.Name, match.Artifact.Version),
		})
	}

	return findings, nil
}
