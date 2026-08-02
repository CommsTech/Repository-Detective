package scanners

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
		logger.Debug("[SCANNER:trivy] binary not found — skipped (optional when grype is installed)")
		result.Status = StatusBinaryMissing
		result.Detail = "trivy not installed; use grype or install trivy in PATH"
		return result
	}

	severity := cfg.TrivySeverity
	if severity == "" {
		severity = "HIGH,CRITICAL"
	}

	reportPath := filepath.Join(dir, ".repository-detective-trivy.json")
	_ = os.Remove(reportPath)
	reportFile, err := os.Create(reportPath)
	if err != nil {
		tmp, tmpErr := os.CreateTemp("", "rd-trivy-*.json")
		if tmpErr != nil {
			result.Status = StatusFailed
			result.Detail = fmt.Sprintf("create trivy report file: %v", err)
			return result
		}
		reportPath = tmp.Name()
		_ = tmp.Close()
	} else {
		_ = reportFile.Close()
	}
	defer func() { _ = os.Remove(reportPath) }()

	cacheDir := filepath.Join(dir, ".rd-trivy-cache")
	_ = os.MkdirAll(cacheDir, 0o755)

	args := []string{
		"fs",
		"--scanners", "vuln,secret,misconfig",
		"--severity", severity,
		"--format", "json",
		"--quiet",
		"--cache-dir", cacheDir,
		"--output", reportPath,
		dir,
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	stdout, stderr, err := runCommandStreams(ctx, timeout, dir, "trivy", args...)
	reportBytes, readErr := os.ReadFile(reportPath)
	if readErr != nil {
		reportBytes = nil
	}
	defer func() { _ = os.RemoveAll(cacheDir) }()

	if err != nil && len(bytes.TrimSpace(reportBytes)) == 0 && len(bytes.TrimSpace(stdout)) == 0 && len(bytes.TrimSpace(stderr)) == 0 {
		result.Status = classifyCommandError(err)
		result.Detail = err.Error()
		logger.Warnf("[SCANNER:trivy] scan failed: status=%s err=%v", result.Status, err)
		return result
	}

	findings, parseErr := parseTrivyOutput(reportBytes, dir)
	if parseErr != nil && len(bytes.TrimSpace(stdout)) > 0 {
		findings, parseErr = parseTrivyOutput(stdout, dir)
	}
	if parseErr != nil && len(bytes.TrimSpace(stderr)) > 0 {
		findings, parseErr = parseTrivyOutput(stderr, dir)
	}
	if parseErr != nil {
		merged := append(append([]byte{}, stdout...), stderr...)
		findings, parseErr = parseTrivyOutput(merged, dir)
	}
	if parseErr != nil {
		errText := strings.TrimSpace(string(stripANSI(stderr)))
		if errText == "" {
			errText = strings.TrimSpace(string(stripANSI(stdout)))
		}
		errText = redactScannerDetail(errText)
		if len(bytes.TrimSpace(reportBytes)) == 0 && errText != "" {
			// Tool failed before producing JSON (cache lock, DB, permissions, etc.).
			result.Status = StatusFailed
			result.Detail = errText
			if err != nil {
				result.Detail += "; " + err.Error()
			}
			logger.Warnf("[SCANNER:trivy] scanner failed: %s", result.Detail)
			return result
		}
		result.Status = StatusParseFailed
		result.Detail = fmt.Sprintf("%v (report_bytes=%d stdout=%d stderr=%d)", parseErr, len(reportBytes), len(stdout), len(stderr))
		if errText != "" {
			result.Detail += "; " + errText
		}
		logger.Warnf("[SCANNER:trivy] failed to parse output: %v", result.Detail)
		return result
	}

	result = resultWithFindings("trivy", findings)
	logResultInfo(logger, "trivy", result.Status, len(findings), result.Detail)
	return result
}

func parseTrivyOutput(output []byte, dir string) ([]Finding, error) {
	payload, err := extractJSONObject(output)
	if err != nil {
		return nil, err
	}
	var report trivyReport
	if err := json.Unmarshal(payload, &report); err != nil {
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
