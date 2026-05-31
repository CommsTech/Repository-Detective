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
func RunGrype(ctx context.Context, logger *logrus.Logger, dir string, cfg Config) ([]Finding, error) {
	if !commandAvailable("grype") {
		logger.Warn("[SCANNER:grype] binary not found — install grype or use the official Bugbot Docker image")
		return nil, nil
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
		logger.Warnf("[SCANNER:grype] scan failed: %v", err)
		return nil, nil
	}

	var report grypeReport
	if err := json.Unmarshal(output, &report); err != nil {
		logger.Warnf("[SCANNER:grype] failed to parse output: %v", err)
		return nil, nil
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

	logger.Infof("[SCANNER:grype] found %d issue(s)", len(findings))
	return findings, nil
}
