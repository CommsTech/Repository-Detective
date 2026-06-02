package scanners

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type trivyReport struct {
	Results []trivyResult `json:"Results"`
}

type trivyResult struct {
	Target            string                  `json:"Target"`
	Class             string                  `json:"Class"`
	Vulnerabilities   []trivyVulnerability    `json:"Vulnerabilities"`
	Misconfigurations []trivyMisconfiguration `json:"Misconfigurations"`
	Secrets           []trivySecret           `json:"Secrets"`
}

type trivyVulnerability struct {
	VulnerabilityID  string `json:"VulnerabilityID"`
	PkgName          string `json:"PkgName"`
	InstalledVersion string `json:"InstalledVersion"`
	FixedVersion     string `json:"FixedVersion"`
	Severity         string `json:"Severity"`
	Title            string `json:"Title"`
	Description      string `json:"Description"`
	PrimaryURL       string `json:"PrimaryURL"`
}

type trivyMisconfiguration struct {
	ID          string `json:"ID"`
	Title       string `json:"Title"`
	Description string `json:"Description"`
	Severity    string `json:"Severity"`
	Message     string `json:"Message"`
}

type trivySecret struct {
	RuleID   string `json:"RuleID"`
	Title    string `json:"Title"`
	Severity string `json:"Severity"`
	Match    string `json:"Match"`
}

// RunTrivy scans a workspace directory with Trivy filesystem mode.
func RunTrivy(ctx context.Context, logger *logrus.Logger, dir string, cfg Config) RunResult {
	result := RunResult{Scanner: "trivy"}
	if !commandAvailable("trivy") {
		logger.Warn("[SCANNER:trivy] binary not found — install trivy or use the official Bugbot Docker image")
		result.Status = StatusBinaryMissing
		return result
	}

	severity := cfg.TrivySeverity
	if severity == "" {
		severity = "HIGH,CRITICAL"
	}

	args := []string{
		"fs",
		"--scanners", "vuln,secret,misconfig",
		"--severity", severity,
		"--format", "json",
		"--quiet",
		dir,
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	output, err := runCommand(ctx, timeout, dir, "trivy", args...)
	if err != nil {
		if len(output) == 0 {
			result.Status = classifyCommandError(err)
			result.Detail = err.Error()
			logger.Warnf("[SCANNER:trivy] scan failed: status=%s err=%v", result.Status, err)
			return result
		}
	}

	findings, parseErr := parseTrivyOutput(output, dir)
	if parseErr != nil {
		result.Status = StatusParseFailed
		result.Detail = parseErr.Error()
		logger.Warnf("[SCANNER:trivy] failed to parse output: %v", parseErr)
		return result
	}

	result = resultWithFindings("trivy", findings)
	logResultInfo(logger, "trivy", result.Status, len(findings), result.Detail)
	return result
}

func parseTrivyOutput(output []byte, dir string) ([]Finding, error) {
	var report trivyReport
	if err := json.Unmarshal(output, &report); err != nil {
		return nil, err
	}

	var findings []Finding
	for _, scanResult := range report.Results {
		target := strings.TrimPrefix(scanResult.Target, dir)
		target = strings.TrimPrefix(target, "/")

		for _, vuln := range scanResult.Vulnerabilities {
			title := vuln.Title
			if title == "" {
				title = fmt.Sprintf("%s in %s", vuln.VulnerabilityID, vuln.PkgName)
			}
			desc := vuln.Description
			if desc == "" {
				desc = fmt.Sprintf("%s %s installed, fixed in %s", vuln.PkgName, vuln.InstalledVersion, vuln.FixedVersion)
			}
			findings = append(findings, Finding{
				ID:          fmt.Sprintf("TRIVY-%s", vuln.VulnerabilityID),
				Source:      "trivy",
				Category:    "dependency_vulnerability",
				Severity:    normalizeSeverity(vuln.Severity),
				Title:       title,
				Description: desc,
				File:        firstNonEmpty(target, vuln.PkgName),
				Confidence:  0.98,
				Reference:   vuln.VulnerabilityID,
				Code:        fmt.Sprintf("%s@%s", vuln.PkgName, vuln.InstalledVersion),
			})
		}

		for _, mis := range scanResult.Misconfigurations {
			findings = append(findings, Finding{
				ID:          fmt.Sprintf("TRIVY-MIS-%s", mis.ID),
				Source:      "trivy",
				Category:    "misconfiguration",
				Severity:    normalizeSeverity(mis.Severity),
				Title:       firstNonEmpty(mis.Title, mis.ID),
				Description: firstNonEmpty(mis.Description, mis.Message),
				File:        target,
				Confidence:  0.95,
				Reference:   mis.ID,
				Code:        mis.Message,
			})
		}

		for _, secret := range scanResult.Secrets {
			findings = append(findings, Finding{
				ID:          fmt.Sprintf("TRIVY-SECRET-%s", secret.RuleID),
				Source:      "trivy",
				Category:    "hardcoded_secret",
				Severity:    normalizeSeverity(secret.Severity),
				Title:       firstNonEmpty(secret.Title, "Exposed secret"),
				Description: "Trivy detected a secret in source or config",
				File:        target,
				Confidence:  0.97,
				Reference:   secret.RuleID,
				Code:        secret.Match,
			})
		}
	}

	return findings, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
