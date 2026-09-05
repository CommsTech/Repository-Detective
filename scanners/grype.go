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
		logger.Warn("[SCANNER:grype] binary not found — install grype or use the official Repository Detective Docker image")
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
	if len(output) == 0 && err != nil {
		result.Status = classifyCommandError(err)
		result.Detail = err.Error()
		logger.Warnf("[SCANNER:grype] scan failed: status=%s err=%v", result.Status, err)
		return result
	}

	findings, status, detail := parseGrypeOutput(output, dir, cfg)
	result.Status = status
	result.Detail = detail
	result.Findings = findings
	if status == StatusParseFailed || status == StatusFailed || status == StatusScannerUnavailable {
		logger.Warnf("[SCANNER:grype] failed to parse output: %v", detail)
	} else {
		logger.Infof("[SCANNER:grype] status=%s findings=%d", result.Status, len(findings))
	}
	return result
}

func parseGrypeOutput(output []byte, dir string, cfg Config) ([]Finding, Status, string) {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return nil, StatusNoSupportedManifest, "empty grype output — no supported dependency manifest detected"
	}

	if status, detail, handled := classifyGrypePlaintext(text); handled {
		return nil, status, detail
	}

	payload, err := extractJSONObject(output)
	if err != nil {
		if status, detail, handled := classifyGrypePlaintext(text); handled {
			return nil, status, detail
		}
		return nil, StatusParseFailed, err.Error()
	}

	var report grypeReport
	if err := json.Unmarshal(payload, &report); err != nil {
		if status, detail, handled := classifyGrypePlaintext(text); handled {
			return nil, status, detail
		}
		return nil, StatusParseFailed, err.Error()
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

	status := StatusClean
	if len(findings) > 0 {
		status = StatusFound
	}
	return findings, status, ""
}

func classifyGrypePlaintext(text string) (Status, string, bool) {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "failed to load vulnerability db"),
		strings.Contains(lower, "database disk image is malformed"),
		strings.Contains(lower, "unable to get namespaces"):
		return StatusScannerUnavailable, strings.TrimSpace(firstLine(text)), true
	case strings.Contains(lower, "no packages were discovered"),
		strings.Contains(lower, "no package catalog"),
		strings.Contains(lower, "unable to catalog"),
		strings.Contains(lower, "no supported package"):
		return StatusNoSupportedManifest, strings.TrimSpace(firstLine(text)), true
	}
	return "", "", false
}

func firstLine(text string) string {
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		return text[:idx]
	}
	return text
}
