package store

import (
	"strings"
	"time"
)

// MatchSuppression reports whether an active, non-expired suppression applies.
func MatchSuppression(sup FindingSuppression, in FindingMatchInput, now time.Time) bool {
	if !sup.Active {
		return false
	}
	if sup.ExpiresAt != nil && !sup.ExpiresAt.After(now) {
		return false
	}
	scope := strings.ToLower(strings.TrimSpace(sup.Scope))
	if scope == SuppressionScopeRepo {
		if sup.RepositoryID == nil || *sup.RepositoryID != in.RepositoryID {
			return false
		}
	} else if scope == SuppressionScopeGlobal {
		if sup.RepositoryID != nil && *sup.RepositoryID != 0 {
			return false
		}
	} else {
		return false
	}
	if sup.Fingerprint != "" && sup.Fingerprint != in.Fingerprint {
		return false
	}
	if sup.Source != "" && !strings.EqualFold(sup.Source, in.Source) {
		return false
	}
	if sup.RuleID != "" && sup.RuleID != in.RuleID {
		return false
	}
	if sup.Category != "" && !strings.EqualFold(sup.Category, in.Category) {
		return false
	}
	if sup.Severity != "" && !strings.EqualFold(sup.Severity, in.Severity) {
		return false
	}
	if sup.Fingerprint == "" && sup.Source == "" && sup.RuleID == "" && sup.Category == "" && sup.Severity == "" {
		return false
	}
	return true
}

// IsSuppressedByList checks a preloaded suppression list (repo + global).
func IsSuppressedByList(suppressions []FindingSuppression, in FindingMatchInput, now time.Time) (bool, FindingSuppression) {
	for _, sup := range suppressions {
		if MatchSuppression(sup, in, now) {
			return true, sup
		}
	}
	return false, FindingSuppression{}
}

// NormalizeSuppressionScope validates scope values.
func NormalizeSuppressionScope(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case SuppressionScopeGlobal:
		return SuppressionScopeGlobal
	default:
		return SuppressionScopeRepo
	}
}
