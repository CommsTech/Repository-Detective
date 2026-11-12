package doctor

import (
	"strings"
)

// RecommendProfile suggests a scan profile from repository language/path hints.
// Profile REQUIRED scanner sets remain authoritative (RD-012A) — this never
// shrinks requirements based on scanner availability.
func RecommendProfile(language string, pathHints []string) (profile string, reason string) {
	lang := strings.ToLower(strings.TrimSpace(language))
	joined := strings.ToLower(strings.Join(pathHints, " "))

	infraHints := []string{"terraform", ".tf", "kubernetes", "k8s", "helm", "dockerfile", "compose.yaml", "docker-compose", "ansible", "pulumi"}
	for _, h := range infraHints {
		if strings.Contains(joined, h) || lang == "hcl" || lang == "dockerfile" {
			return "standard", "Infrastructure / IaC signals detected — recommend Standard (full deterministic + issues when policy allows)"
		}
	}
	docOnly := len(pathHints) > 0
	codeExt := false
	for _, p := range pathHints {
		pl := strings.ToLower(p)
		switch {
		case strings.HasSuffix(pl, ".md"), strings.HasSuffix(pl, ".txt"), strings.Contains(pl, "docs/"), strings.HasSuffix(pl, ".rst"):
			continue
		case strings.Contains(pl, "."):
			codeExt = true
			docOnly = false
		}
	}
	if docOnly && !codeExt && lang == "" {
		return "light", "Documentation-oriented repository — recommend Light (fast read-only)"
	}
	switch lang {
	case "go":
		return "standard", "Go service detected — recommend Standard"
	case "python", "javascript", "typescript", "java", "rust":
		return "standard", lang + " codebase detected — recommend Standard"
	case "markdown", "text":
		return "light", "Documentation language signal — recommend Light"
	}
	if lang == "" && !codeExt {
		return "light", "Limited code signals — recommend Light; override if needed"
	}
	return "standard", "Default Community recommendation: Standard (Observe policy until you intentionally tighten)"
}
