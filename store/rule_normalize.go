package store

import "strings"

// NormalizeLearningRuleID collapses legacy per-line linter IDs so calibration aggregates evidence.
func NormalizeLearningRuleID(source, ruleID string) string {
	ruleID = strings.Trim(strings.TrimSpace(ruleID), "`\"' ")
	src := strings.ToLower(strings.TrimSpace(source))
	if src == "golangci-lint" && strings.HasPrefix(ruleID, "LINT-GO-typecheck") {
		return "LINT-GO-typecheck"
	}
	// Collapse legacy per-finding ruff suffixes (LINT-RUFF-E902-1 → LINT-RUFF-E902).
	if (src == "ruff" || strings.HasPrefix(ruleID, "LINT-RUFF-")) && strings.HasPrefix(ruleID, "LINT-RUFF-") {
		parts := strings.Split(ruleID, "-")
		// LINT RUFF CODE [optional numeric instance]
		if len(parts) >= 4 {
			last := parts[len(parts)-1]
			numeric := true
			for _, r := range last {
				if r < '0' || r > '9' {
					numeric = false
					break
				}
			}
			if numeric {
				return strings.Join(parts[:len(parts)-1], "-")
			}
		}
	}
	return ruleID
}
