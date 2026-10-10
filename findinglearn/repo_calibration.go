package findinglearn

import (
	"path/filepath"
	"strings"
	"time"
)

// RepoCalibrationRule is a repo-scoped calibration input (decoupled from store to avoid import cycles).
type RepoCalibrationRule struct {
	Source      string
	RuleID      string
	PathPattern string
	Action      string
	Reason      string
	Active      bool
	ExpiresAt   *time.Time
}

// ApplyRepoRule adjusts severity/confidence for a repo-scoped calibration rule.
// Findings remain visible; high/critical severities are never downgraded.
func ApplyRepoRule(severity string, confidence float64, source, ruleID, filePath string, rule RepoCalibrationRule, now time.Time) (string, float64, string) {
	if !rule.Active {
		return severity, confidence, ""
	}
	if rule.ExpiresAt != nil && !rule.ExpiresAt.After(now) {
		return severity, confidence, ""
	}
	if rule.Source != "" && !strings.EqualFold(rule.Source, source) {
		return severity, confidence, ""
	}
	if rule.RuleID != "" && !strings.EqualFold(rule.RuleID, ruleID) {
		return severity, confidence, ""
	}
	if rule.PathPattern != "" && !pathMatchesCalibrationPattern(filePath, rule.PathPattern) {
		return severity, confidence, ""
	}
	if severity == "high" || severity == "critical" {
		return severity, confidence, ""
	}
	switch rule.Action {
	case "downgrade_confidence", "report_only", "informational":
		outSev := severity
		if severity == "medium" || severity == "low" {
			outSev = "info"
		}
		outConf := confidence
		if outConf > 0.55 {
			outConf = 0.55
		}
		note := strings.TrimSpace(rule.Reason)
		if note == "" {
			note = "Repo-scoped calibration — informational (finding remains visible)."
		}
		return outSev, outConf, note
	default:
		return severity, confidence, ""
	}
}

// ApplyRepoRules applies the first matching active repo calibration rule.
func ApplyRepoRules(severity string, confidence float64, source, ruleID, filePath string, rules []RepoCalibrationRule) (string, float64, string) {
	now := time.Now().UTC()
	for _, rule := range rules {
		if sev, conf, note := ApplyRepoRule(severity, confidence, source, ruleID, filePath, rule, now); note != "" {
			return sev, conf, note
		}
	}
	return severity, confidence, ""
}

// ApplyRepoRoutingForForge sets ReportingAction to report_only when an active
// repo calibration rule matches. Unlike ApplyRepoRule, this applies to
// high/critical findings too — severity stays visible on the dashboard, but
// forge filing is quieted for operator-approved calibrations.
func ApplyRepoRoutingForForge(reportingAction, source, ruleID, filePath string, rules []RepoCalibrationRule) (string, string) {
	now := time.Now().UTC()
	for _, rule := range rules {
		if !rule.Active {
			continue
		}
		if rule.ExpiresAt != nil && !rule.ExpiresAt.After(now) {
			continue
		}
		if rule.Source != "" && !strings.EqualFold(rule.Source, source) {
			continue
		}
		if rule.RuleID != "" && !strings.EqualFold(rule.RuleID, ruleID) {
			continue
		}
		if rule.PathPattern != "" && !pathMatchesCalibrationPattern(filePath, rule.PathPattern) {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(rule.Action)) {
		case "report_only", "informational", "downgrade_confidence":
			note := strings.TrimSpace(rule.Reason)
			if note == "" {
				note = "Repo calibration — forge report_only (finding remains visible)."
			}
			return "report_only", note
		}
	}
	return reportingAction, ""
}

func pathMatchesCalibrationPattern(path, pattern string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	pattern = filepath.ToSlash(strings.TrimSpace(pattern))
	if pattern == "" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(path, strings.TrimSuffix(pattern, "*"))
	}
	return path == pattern || strings.Contains(path, pattern)
}
