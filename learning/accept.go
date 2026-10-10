package learning

import (
	"fmt"
	"strings"
)

// ValidateCalibrationAccept checks whether a calibration recommendation may be accepted.
// Severity is not used here: high/critical protection is enforced again at scan persist time.
// Category protection blocks secrets/security classes from becoming suppressions via accept.
// Global-scope recommendations are allowed: accept expands them into repo-scoped rules
// (never a fleet-wide global suppression).
//
// source/ruleID may be empty for legacy callers; when present, known informational
// tooling rules (gosec G104/G304/G204, lint noise) may be accepted even if the
// scanner labeled the finding category as "security".
func ValidateCalibrationAccept(category, scope string) error {
	return ValidateCalibrationAcceptRule(category, scope, "", "")
}

// ValidateCalibrationAcceptRule is the rule-aware accept gate used by auto-apply
// and operator accept paths.
func ValidateCalibrationAcceptRule(category, scope, source, ruleID string) error {
	_ = scope // retained for callers / future scope policy
	if strings.TrimSpace(scope) == "" {
		return fmt.Errorf("recommendation scope is required")
	}
	if IsInformationalToolingRule(source, ruleID) {
		return nil
	}
	if IsProtectedFromAutoDowngrade("", category) {
		return fmt.Errorf("recommendation affects protected security category — mark findings false-positive individually or use an explicit operator override")
	}
	return nil
}

// IsInformationalToolingRule reports rules that scanners often tag as "security"
// but that are safe for report_only calibration when FP rates are high.
func IsInformationalToolingRule(source, ruleID string) bool {
	src := strings.ToLower(strings.TrimSpace(source))
	rule := strings.ToUpper(strings.TrimSpace(ruleID))
	rule = strings.Trim(rule, "`\"' ")

	if strings.Contains(src, "gosec") {
		switch rule {
		case "G104", "G304", "G204":
			return true
		}
	}
	if strings.EqualFold(rule, "LINT-GO-TYPECHECK") || strings.HasPrefix(rule, "LINT-GO-TYPECHECK") {
		return true
	}
	if strings.HasPrefix(rule, "LINT-SHELL-") || strings.HasPrefix(rule, "LINT-RUFF-") {
		return true
	}
	// OPT-HTTP-CLIENT-PER-CALL is only informational when the client is shared/long-lived;
	// keep it calibratable for product tooling, but U1000 unused-code is never "informational".
	if rule == "U1000" || strings.HasPrefix(rule, "U1") {
		return false
	}
	if strings.HasPrefix(rule, "GRAPH-") || strings.HasPrefix(rule, "OPT-") || strings.HasPrefix(rule, "QUAL-") {
		return true
	}
	if strings.HasPrefix(rule, "HEALTH-") && !strings.Contains(rule, "SECRET") {
		return true
	}
	return false
}
