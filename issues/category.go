package issues

import "strings"

// Formal issue categories used in labels and templates.
const (
	CategorySecurity         = "security"
	CategorySecret           = "secret"
	CategoryDependency       = "dependency"
	CategoryMisconfiguration = "misconfiguration"
	CategoryCodeQuality      = "code_quality"
	CategoryTechDebt         = "tech_debt"
	CategoryReliability      = "reliability"
	CategoryPerformance      = "performance"
	CategoryMaintainability  = "maintainability"
	CategoryTestGap          = "test_gap"
	CategoryAIGeneratedRisk  = "ai_generated_risk"
	CategoryArchitecture     = "architecture"
	CategoryUnknown          = "unknown"
)

// NormalizeCategory maps legacy/scanner categories to formal Repository Detective categories.
func NormalizeCategory(category, source string) string {
	category = strings.ToLower(strings.TrimSpace(category))
	source = strings.ToLower(strings.TrimSpace(source))

	switch category {
	case CategorySecurity, CategorySecret, CategoryDependency, CategoryMisconfiguration,
		CategoryCodeQuality, CategoryTechDebt, CategoryReliability, CategoryPerformance,
		CategoryMaintainability, CategoryTestGap, CategoryAIGeneratedRisk, CategoryArchitecture:
		return category
	case "hardcoded_secret":
		return CategorySecret
	case "dependency_vulnerability", "dependency-vulnerability":
		return CategoryDependency
	case "misconfig":
		return CategoryMisconfiguration
	case "sql_injection", "xss", "code_injection", "command_injection", "injection", "sast":
		return CategorySecurity
	case "quality", "lint":
		if source == "ruff" || source == "shellcheck" || source == "golangci-lint" {
			return CategoryMaintainability
		}
		return CategoryCodeQuality
	default:
		if source == "gitleaks" {
			return CategorySecret
		}
		if source == "grype" || source == "trivy" && category == "" {
			return CategoryDependency
		}
		switch source {
		case "tech_debt", "maintainability", "health":
			if category == "" {
				return CategoryTechDebt
			}
		case "reliability":
			if category == "" {
				return CategoryReliability
			}
		case "test_gap":
			if category == "" {
				return CategoryTestGap
			}
		case "performance":
			if category == "" {
				return CategoryPerformance
			}
		case "ai_generated_risk":
			return CategoryAIGeneratedRisk
		case "graph":
			if category == "" || category == "maintainability" {
				return CategoryMaintainability
			}
			if category == "architecture" {
				return CategoryArchitecture
			}
		}
		if category == "" {
			return CategoryUnknown
		}
		return category
	}
}

// MapSemgrepCategory maps Semgrep metadata category to formal category.
func MapSemgrepCategory(metadataCategory string) string {
	meta := strings.ToLower(strings.TrimSpace(metadataCategory))
	switch meta {
	case "security", "security-audit":
		return CategorySecurity
	case "maintainability", "best-practice", "correctness":
		return CategoryCodeQuality
	case "performance":
		return CategoryPerformance
	default:
		if meta == "" {
			return CategorySecurity
		}
		return NormalizeCategory(meta, "semgrep")
	}
}

// CategoryLabel returns the Gitea category label for writes (Repository Detective namespace).
func CategoryLabel(category string) string {
	return CategoryLabelNew(category)
}
