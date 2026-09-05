package issues

import "strings"

const (
	baseLabel            = "repository-detective"
	automatedReviewLabel = "automated-review"
)

// DefaultIssueBaseLabels returns configured base labels for new issues.
func DefaultIssueBaseLabels() []string {
	return uniqueStrings(append(BaseLabelsForWrite(), automatedReviewLabel))
}

// BaseLabelsForWrite returns product base labels for new issue submissions.
func BaseLabelsForWrite() []string {
	return []string{baseLabel}
}

// IssueLookupBaseLabels returns base labels searched when locating existing issues.
func IssueLookupBaseLabels() []string {
	return []string{baseLabel}
}

// ExpandBrandLabel returns category (or other paired) labels for write mode.
func ExpandBrandLabel(_legacyLabel, newLabel string) []string {
	return []string{newLabel}
}

// CategoryLabelForWrite returns the category label applied when filing Gitea issues.
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

// ExpandLifecycleLabels expands lifecycle labels for write mode.
func ExpandLifecycleLabels(lifecycleLabel string) []string {
	suffix := lifecycleSuffix(lifecycleLabel)
	if suffix == "" {
		suffix = "open"
	}
	return []string{newLifecycleLabel(suffix)}
}

func lifecycleSuffix(label string) string {
	label = strings.TrimSpace(label)
	prefix := baseLabel + "/"
	if strings.HasPrefix(label, prefix) {
		return strings.TrimPrefix(label, prefix)
	}
	return label
}

func newLifecycleLabel(suffix string) string {
	return baseLabel + "/" + suffix
}

// CategoryLabelNew returns the Repository Detective category label slug.
func CategoryLabelNew(category string) string {
	switch NormalizeCategory(category, "") {
	case CategorySecurity, CategoryMisconfiguration:
		return baseLabel + "/security"
	case CategorySecret:
		return baseLabel + "/secret"
	case CategoryDependency:
		return baseLabel + "/dependency"
	case CategoryCodeQuality:
		return baseLabel + "/code-quality"
	case CategoryTechDebt:
		return baseLabel + "/tech-debt"
	case CategoryReliability:
		return baseLabel + "/reliability"
	case CategoryMaintainability:
		return baseLabel + "/maintainability"
	case CategoryPerformance:
		return baseLabel + "/performance"
	case CategoryTestGap:
		return baseLabel + "/test-gap"
	case CategoryAIGeneratedRisk:
		return baseLabel + "/ai-generated-risk"
	case CategoryArchitecture:
		return baseLabel + "/architecture"
	default:
		return baseLabel + "/code-quality"
	}
}

// FingerprintBodyMarker is the marker written into new issue bodies.
const FingerprintBodyMarker = "Repository Detective fingerprint:"
