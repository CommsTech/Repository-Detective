package issues

import "strings"

const (
	sourceLabel = "source/repository-detective"
)

// DefaultIssueBaseLabels returns provenance labels for new issues.
func DefaultIssueBaseLabels() []string {
	return BaseLabelsForWrite()
}

// BaseLabelsForWrite returns product base labels for new issue submissions.
func BaseLabelsForWrite() []string {
	return []string{sourceLabel}
}

// IssueLookupBaseLabels returns base labels searched when locating existing issues.
// Includes legacy product labels so historical issues remain discoverable.
func IssueLookupBaseLabels() []string {
	return []string{sourceLabel, "repository-detective", "automated-review"}
}

// ExpandBrandLabel returns category (or other paired) labels for write mode.
func ExpandBrandLabel(_legacyLabel, newLabel string) []string {
	return []string{newLabel}
}

// CategoryLabelForWrite returns the scoped category label for forge issues.
func CategoryLabelForWrite(category string) string {
	return CategoryLabelNew(category)
}

// ExpandLifecycleLabel expands a lifecycle constant for Gitea label APIs.
func ExpandLifecycleLabel(lifecycleLabel string) []any {
	labels := ExpandLifecycleLabels(lifecycleLabel)
	out := make([]any, len(labels))
	for i, label := range labels {
		out[i] = label
	}
	return out
}

// ExpandLifecycleLabels maps lifecycle constants onto forge labels.
// Closure/remediation states keep repository-detective/* for engine compatibility;
// triage spam labels (open/still-present) are not written on every rescan.
func ExpandLifecycleLabels(lifecycleLabel string) []string {
	switch lifecycleLabel {
	case LifecycleOpen, "":
		return []string{TriageNeedsReview}
	case LifecycleNeedsHumanReview:
		return []string{TriageNeedsReview}
	case LifecycleStillPresent:
		return nil
	case LifecycleFalsePositive:
		return []string{TriageFalsePositive, newLifecycleLabel("false-positive")}
	case LifecycleSuppressed:
		return []string{TriageAcceptedRisk, newLifecycleLabel("suppressed")}
	case LifecycleFixed:
		return []string{TriageConfirmed, newLifecycleLabel("fixed")}
	case LifecycleResolvedVerified:
		return []string{TriageConfirmed, newLifecycleLabel("resolved-verified")}
	case LifecycleNotReproduced:
		return []string{newLifecycleLabel("not-reproduced")}
	case LifecycleRemediationCandidate:
		return []string{RemediationAutoPR, newLifecycleLabel("remediation-candidate")}
	case LifecycleFixPROpened:
		return []string{RemediationAutoPR, newLifecycleLabel("fix-pr-opened")}
	case LifecycleFixPRMerged:
		return []string{RemediationAutoPR, TriageConfirmed, newLifecycleLabel("fix-pr-merged")}
	case LifecyclePendingRescan:
		return []string{newLifecycleLabel("pending-rescan")}
	case LifecycleClosureBlocked:
		return []string{RemediationBlocked, newLifecycleLabel("closure-blocked")}
	case LifecycleDuplicate:
		return []string{newLifecycleLabel("duplicate")}
	default:
		suffix := lifecycleSuffix(lifecycleLabel)
		if suffix == "open" || suffix == "" {
			return []string{TriageNeedsReview}
		}
		return []string{newLifecycleLabel(suffix)}
	}
}

func newLifecycleLabel(suffix string) string {
	return "repository-detective/" + suffix
}

func lifecycleSuffix(label string) string {
	label = strings.TrimSpace(label)
	for _, prefix := range []string{"repository-detective/", "source/"} {
		if strings.HasPrefix(label, prefix) {
			return strings.TrimPrefix(label, prefix)
		}
	}
	return label
}

// CategoryLabelNew returns the scoped category label slug.
func CategoryLabelNew(category string) string {
	switch NormalizeCategory(category, "") {
	case CategorySecurity, CategoryMisconfiguration:
		return "category/security"
	case CategorySecret:
		return "category/secret"
	case CategoryDependency:
		return "category/dependency"
	case CategoryReliability:
		return "category/reliability"
	case CategoryMaintainability, CategoryCodeQuality, CategoryTechDebt, CategoryArchitecture, CategoryPerformance, CategoryTestGap:
		return "category/maintainability"
	case CategoryAIGeneratedRisk:
		return "category/security"
	default:
		return "category/maintainability"
	}
}

// ScannerLabel returns scanner/<name> for exclusive filtering.
func ScannerLabel(source string) string {
	src := strings.ToLower(strings.TrimSpace(source))
	src = strings.ReplaceAll(src, " ", "-")
	if src == "" || src == "unknown" {
		return ""
	}
	// Collapse AI auditors under source name still.
	if idx := strings.Index(src, "("); idx > 0 {
		src = strings.TrimSpace(src[:idx])
	}
	return "scanner/" + src
}

// RemediationLabel returns remediation/* from issue metadata.
func RemediationLabel(issueFixable string, safeForAutoPR bool) string {
	if safeForAutoPR && strings.EqualFold(issueFixable, "true") {
		return RemediationAutoPR
	}
	if strings.EqualFold(issueFixable, "true") {
		return RemediationManual
	}
	if strings.EqualFold(issueFixable, "false") {
		return RemediationBlocked
	}
	return RemediationUnknown
}

// FingerprintBodyMarker is the marker written into new issue bodies.
const FingerprintBodyMarker = "Fingerprint:"

// legacyFingerprintMarkers remain readable for historical forge issues.
var legacyFingerprintMarkers = []string{
	"Repository Detective fingerprint:",
	"Bugbot fingerprint:",
}

// Scoped triage / remediation labels (exclusive scopes in Gitea).
const (
	TriageNeedsReview   = "triage/needs-review"
	TriageConfirmed     = "triage/confirmed"
	TriageFalsePositive = "triage/false-positive"
	TriageAcceptedRisk  = "triage/accepted-risk"

	RemediationAutoPR  = "remediation/auto-pr"
	RemediationManual  = "remediation/manual"
	RemediationBlocked = "remediation/blocked"
	RemediationUnknown = "remediation/unknown"
)
