package store

import "strings"

// NormalizeLearningRuleID collapses legacy per-line linter IDs so calibration aggregates evidence.
func NormalizeLearningRuleID(source, ruleID string) string {
	ruleID = strings.TrimSpace(ruleID)
	src := strings.ToLower(strings.TrimSpace(source))
	if src == "golangci-lint" && strings.HasPrefix(ruleID, "LINT-GO-typecheck") {
		return "LINT-GO-typecheck"
	}
	return ruleID
}
