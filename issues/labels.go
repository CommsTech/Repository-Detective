package issues

import "strings"

// Label compatibility modes for Gitea issue labels during branding migration.
const (
	LabelCompatLegacyOnly = "legacy_only"
	LabelCompatDual       = "dual"
	LabelCompatNewOnly    = "new_only"
)

const (
	legacyBaseLabel      = "bugbot"
	newBaseLabel         = "repository-detective"
	automatedReviewLabel = "automated-review"
)

var currentLabelCompatMode = LabelCompatNewOnly

// SetLabelCompatMode configures how issue labels are written.
func SetLabelCompatMode(mode string) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case LabelCompatLegacyOnly:
		currentLabelCompatMode = LabelCompatLegacyOnly
	case LabelCompatNewOnly:
		currentLabelCompatMode = LabelCompatNewOnly
	default:
		currentLabelCompatMode = LabelCompatDual
	}
}

// LabelCompatMode returns the active label compatibility mode.
func LabelCompatMode() string {
	return currentLabelCompatMode
}

// DefaultIssueBaseLabels returns configured base labels for new issues.
func DefaultIssueBaseLabels() []string {
	return uniqueStrings(append(BaseLabelsForWrite(), automatedReviewLabel))
}

// BaseLabelsForWrite returns product base labels for new issue submissions.
func BaseLabelsForWrite() []string {
	if currentLabelCompatMode == LabelCompatLegacyOnly {
		return []string{legacyBaseLabel}
	}
	return []string{newBaseLabel}
}

// IssueLookupBaseLabels returns base labels searched when locating existing issues.
func IssueLookupBaseLabels() []string {
	return []string{legacyBaseLabel, newBaseLabel}
}

// ExpandBrandLabel returns category (or other paired) labels for write mode.
// Dual mode no longer writes legacy bugbot/* labels — only repository-detective/*.
func ExpandBrandLabel(legacyLabel, newLabel string) []string {
	if currentLabelCompatMode == LabelCompatLegacyOnly {
		return []string{legacyLabel}
	}
	return []string{newLabel}
}

// CategoryLabelForWrite returns the category label applied when filing Gitea issues.
func CategoryLabelForWrite(category string) string {
	if currentLabelCompatMode == LabelCompatLegacyOnly {
		return CategoryLabelLegacy(category)
	}
	return CategoryLabelNew(category)
}

// ExpandLifecycleLabel expands a legacy lifecycle constant for Gitea label APIs.
func ExpandLifecycleLabel(lifecycleLabel string) []any {
	labels := ExpandLifecycleLabels(lifecycleLabel)
	out := make([]any, len(labels))
	for i, label := range labels {
		out[i] = label
	}
	return out
}

// ExpandLifecycleLabels expands lifecycle labels for write mode (Repository Detective by default).
func ExpandLifecycleLabels(lifecycleLabel string) []string {
	suffix := lifecycleSuffix(lifecycleLabel)
	if suffix == "" {
		suffix = "open"
	}
	if currentLabelCompatMode == LabelCompatLegacyOnly {
		return []string{legacyLifecycleLabel(suffix)}
	}
	return []string{newLifecycleLabel(suffix)}
}

func lifecycleSuffix(label string) string {
	label = strings.TrimSpace(label)
	for _, prefix := range []string{legacyBaseLabel + "/", newBaseLabel + "/"} {
		if strings.HasPrefix(label, prefix) {
			return strings.TrimPrefix(label, prefix)
		}
	}
	return label
}

func legacyLifecycleLabel(suffix string) string {
	return legacyBaseLabel + "/" + suffix
}

func newLifecycleLabel(suffix string) string {
	return newBaseLabel + "/" + suffix
}

// CategoryLabelNew returns the Repository Detective category label slug.
func CategoryLabelNew(category string) string {
	switch NormalizeCategory(category, "") {
	case CategorySecurity, CategoryMisconfiguration:
		return newBaseLabel + "/security"
	case CategorySecret:
		return newBaseLabel + "/secret"
	case CategoryDependency:
		return newBaseLabel + "/dependency"
	case CategoryCodeQuality:
		return newBaseLabel + "/code-quality"
	case CategoryTechDebt:
		return newBaseLabel + "/tech-debt"
	case CategoryReliability:
		return newBaseLabel + "/reliability"
	case CategoryMaintainability:
		return newBaseLabel + "/maintainability"
	case CategoryPerformance:
		return newBaseLabel + "/performance"
	case CategoryTestGap:
		return newBaseLabel + "/test-gap"
	case CategoryAIGeneratedRisk:
		return newBaseLabel + "/ai-generated-risk"
	case CategoryArchitecture:
		return newBaseLabel + "/architecture"
	default:
		return newBaseLabel + "/code-quality"
	}
}

// FingerprintBodyMarker is the marker written into new issue bodies.
const FingerprintBodyMarker = "Repository Detective fingerprint:"
