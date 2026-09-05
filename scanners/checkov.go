package scanners

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

func init() {
	RegisterDeterministicSource("checkov")
}

type checkovReport struct {
	CheckType string              `json:"check_type"`
	Results   checkovResultsBlock `json:"results"`
}

type checkovResultsBlock struct {
	FailedChecks []checkovFailedCheck `json:"failed_checks"`
}

type checkovFailedCheck struct {
	CheckID       string `json:"check_id"`
	CheckName     string `json:"check_name"`
	FilePath      string `json:"file_path"`
	FileLineRange []int  `json:"file_line_range"`
	Resource      string `json:"resource"`
	Guideline     string `json:"guideline"`
	Severity      string `json:"severity"`
}

// RunCheckov scans IaC/config files using checkov.
func RunCheckov(ctx context.Context, logger *logrus.Logger, dir string, entries []FileEntry, cfg Config) RunResult {
	return runCheckovWithCommand(ctx, logger, dir, entries, cfg, "checkov")
}

func runCheckovWithCommand(ctx context.Context, logger *logrus.Logger, dir string, entries []FileEntry, cfg Config, commandName string) RunResult {
	result := RunResult{Scanner: "checkov"}
	if !WorkspaceHasIaC(entries) {
		result.Status = StatusClean
		result.Detail = "no IaC/config files"
		return result
	}
	if !commandAvailable(commandName) {
		if logger != nil {
			logger.Warn("[SCANNER:checkov] binary not found — install checkov (python3 -m pip install --user checkov)")
		}
		result.Status = StatusBinaryMissing
		return result
	}

	timeout := scannerTimeoutSeconds(cfg.CheckovTimeoutSeconds, cfg.TimeoutSeconds)
	output, err := runCommand(ctx, timeout, dir, commandName, "-d", ".", "-o", "json", "--quiet", "--skip-download")
	if err != nil && len(output) == 0 {
		result.Status = classifyCommandError(err)
		result.Detail = err.Error()
		return result
	}

	parsed, parseErr := parseCheckovOutput(output, dir, cfg)
	if parseErr != nil && len(parsed.Findings) == 0 {
		result.Status = StatusParseFailed
		result.Detail = parseErr.Error()
		return result
	}

	result = resultWithFindings("checkov", parsed.Findings)
	if parsed.Truncated {
		max := iacScannerMaxFindings(cfg)
		result.Detail = truncateDetailForScanner("findings", max, parsed.Total)
	}
	if logger != nil {
		logResultInfo(logger, "checkov", result.Status, len(parsed.Findings), result.Detail)
	}
	return result
}

func parseCheckovOutput(output []byte, dir string, cfg Config) (cappedFindings, error) {
	checks, err := extractCheckovFailedChecks(output)
	if err != nil {
		return cappedFindings{}, err
	}
	findings := make([]Finding, 0, len(checks))
	for _, check := range checks {
		findings = append(findings, checkovFinding(check, dir))
	}
	return capFindings(findings, iacScannerMaxFindings(cfg)), nil
}

func extractCheckovFailedChecks(output []byte) ([]checkovFailedCheck, error) {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return nil, nil
	}

	var reports []checkovReport
	if err := json.Unmarshal(output, &reports); err == nil {
		return flattenCheckovReports(reports), nil
	}

	var single checkovReport
	if err := json.Unmarshal(output, &single); err == nil && single.Results.FailedChecks != nil {
		return single.Results.FailedChecks, nil
	}

	var resultsOnly checkovResultsBlock
	if err := json.Unmarshal(output, &resultsOnly); err == nil {
		return resultsOnly.FailedChecks, nil
	}

	return nil, fmt.Errorf("parse checkov json: no failed_checks block found")
}

func flattenCheckovReports(reports []checkovReport) []checkovFailedCheck {
	var out []checkovFailedCheck
	for _, report := range reports {
		out = append(out, report.Results.FailedChecks...)
	}
	return out
}

func checkovFinding(check checkovFailedCheck, dir string) Finding {
	ruleID := strings.TrimSpace(check.CheckID)
	if ruleID == "" {
		ruleID = "CHECKOV"
	}
	line := 0
	if len(check.FileLineRange) > 0 {
		line = check.FileLineRange[0]
	}
	title := strings.TrimSpace(check.CheckName)
	if title == "" {
		title = ruleID
	}
	evidenceParts := []string{check.CheckName, check.Resource, check.Guideline}
	evidence := trimEvidence(strings.TrimSpace(strings.Join(filterNonEmpty(evidenceParts), " · ")))
	return Finding{
		Source:      "checkov",
		Category:    mapCheckovCategory(ruleID, check.Resource, check.CheckName),
		Severity:    mapCheckovSeverity(check.Severity),
		Title:       title,
		Description: evidence,
		File:        relPath(dir, check.FilePath),
		Line:        line,
		Code:        evidence,
		Confidence:  0.90,
		Reference:   ruleID,
		ID:          ruleID,
	}
}

func mapCheckovSeverity(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "CRITICAL":
		return "critical"
	case "HIGH":
		return "high"
	case "LOW":
		return "low"
	case "MEDIUM":
		return "medium"
	default:
		return "medium"
	}
}

func mapCheckovCategory(ruleID, resource, checkName string) string {
	upper := strings.ToUpper(ruleID + " " + resource + " " + checkName)
	switch {
	case strings.Contains(upper, "SECRET") || strings.Contains(upper, "IAM") || strings.Contains(upper, "SG_"):
		return "security"
	case strings.Contains(upper, "DOCKER") || strings.Contains(upper, "K8S") || strings.Contains(upper, "KUBERNETES"):
		return "container"
	case strings.Contains(upper, "TF") || strings.Contains(upper, "TERRAFORM") || strings.Contains(upper, "CLOUDFORMATION") || strings.Contains(upper, "CKV_AWS") || strings.Contains(upper, "CKV_AZURE") || strings.Contains(upper, "CKV_GCP"):
		return "iac"
	default:
		return "misconfiguration"
	}
}

func filterNonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func ParseCheckovOutputForTest(output []byte, dir string, cfg Config) (cappedFindings, error) {
	return parseCheckovOutput(output, dir, cfg)
}

func MapCheckovSeverityForTest(value string) string {
	return mapCheckovSeverity(value)
}

func MapCheckovCategoryForTest(ruleID, resource, checkName string) string {
	return mapCheckovCategory(ruleID, resource, checkName)
}
