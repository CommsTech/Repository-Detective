package remediation

import (
	"strings"
)

type recipe struct {
	fixStrategy string
	complexity  string
	summary     string
	blocked     []string
	humanReview bool
	safeAutoPR  bool
}

// ApplyRecipe returns a deterministic plan draft from finding context.
func ApplyRecipe(ctx FindingContext) Plan {
	category := strings.ToLower(strings.TrimSpace(ctx.Category))
	source := strings.ToLower(strings.TrimSpace(ctx.Source))
	rule := strings.ToLower(strings.TrimSpace(ctx.RuleID))

	r := matchRecipe(category, source, rule, ctx)
	affected := affectedFiles(ctx)
	hints := InferRepoHints(append(affected, ctx.FilePath)...)
	requiredTests, validationCommands := SuggestTests(ctx, hints)

	regressionRisk, requiresHuman, blocked := AssessRisk(ctx, r.complexity)
	if r.humanReview {
		requiresHuman = true
	}
	if len(r.blocked) > 0 {
		blocked = append(blocked, r.blocked...)
	}
	safe := r.safeAutoPR && SafeForAutoPR(ctx, r.complexity, regressionRisk, blocked)

	summary := ctx.Summary
	if summary == "" {
		summary = ctx.Title
	}
	if r.summary != "" {
		summary = r.summary
	}

	return Plan{
		FindingID:           ctx.FindingID,
		Fingerprint:         ctx.Fingerprint,
		RepositoryID:        ctx.RepositoryID,
		AuditID:             ctx.AuditID,
		Category:            category,
		Severity:            strings.ToLower(ctx.Severity),
		Source:              source,
		RuleID:              ctx.RuleID,
		Title:               ctx.Title,
		Summary:             summary,
		FixStrategy:         r.fixStrategy,
		AffectedFiles:       affected,
		TargetLine:          ctx.Line,
		RequiredTests:       requiredTests,
		ValidationCommands:  validationCommands,
		RegressionRisk:      regressionRisk,
		FixComplexity:       r.complexity,
		SafeForAutoPR:       safe,
		RequiresHumanReview: requiresHuman,
		BlockedReasons:      uniqueStrings(blocked),
		Advisory:            false,
		Status:              StatusProposed,
	}
}

func matchRecipe(category, source, rule string, ctx FindingContext) recipe {
	switch {
	case category == "secret" || source == "gitleaks":
		return recipe{
			fixStrategy: "Remove secret from code, rotate credential out-of-band, purge history if required, add secret scanning/pre-commit protection",
			complexity:  ComplexityMedium,
			summary:     "Hardcoded secret detected — remove from source and rotate credential outside the repository",
			blocked:     []string{"do not commit rotated secrets", "do not auto-fix by deleting production credentials without rotation plan"},
			humanReview: true,
			safeAutoPR:  false,
		}
	case category == "dependency" || source == "govulncheck" || source == "grype" || source == "trivy":
		versionHint := ""
		if ctx.PackageName != "" {
			versionHint = " for " + ctx.PackageName
		}
		return recipe{
			fixStrategy: "Upgrade vulnerable dependency" + versionHint + ", update lockfiles, and verify with tests and vulnerability rescan",
			complexity:  ComplexitySmall,
			summary:     "Dependency vulnerability — prefer patched version from scanner advisory when available",
			humanReview: strings.Contains(strings.ToLower(ctx.Summary), "major"),
			safeAutoPR:  !strings.Contains(strings.ToLower(ctx.Summary), "major"),
		}
	case source == "gosec" || strings.HasPrefix(rule, "gosec"):
		return recipe{
			fixStrategy: "Replace unsafe pattern (command execution, path traversal, weak crypto, excessive permissions) with validated safe alternative",
			complexity:  ComplexityMedium,
			summary:     "Go security finding — apply minimal fix and add regression test",
			humanReview: true,
			safeAutoPR:  false,
		}
	case source == "staticcheck" || strings.HasPrefix(rule, "sa") || strings.HasPrefix(rule, "st"):
		return recipe{
			fixStrategy: "Apply staticcheck-guided fix with no behavior change unless rule requires intentional behavior update",
			complexity:  ComplexitySmall,
			summary:     "Static analysis finding — small targeted code quality fix",
			humanReview: false,
			safeAutoPR:  true,
		}
	case source == "hadolint" || strings.Contains(rule, "dl"):
		return recipe{
			fixStrategy: "Update Dockerfile instruction to satisfy hadolint rule (pin base image, use COPY instead of ADD, etc.)",
			complexity:  ComplexitySmall,
			summary:     "Container lint finding — adjust Dockerfile and re-run hadolint",
			humanReview: false,
			safeAutoPR:  true,
		}
	case source == "checkov" || category == "misconfiguration":
		return recipe{
			fixStrategy: "Adjust IaC resource configuration to meet policy; validate with Checkov",
			complexity:  ComplexityMedium,
			summary:     "Infrastructure misconfiguration — update IaC and re-run Checkov",
			humanReview: true,
			safeAutoPR:  false,
		}
	case category == "test_gap" || source == "test_gap":
		return recipe{
			fixStrategy: "Add focused tests covering the referenced code path before larger refactors",
			complexity:  ComplexityMedium,
			summary:     "Test coverage gap — add tests to prove expected behavior",
			humanReview: true,
			safeAutoPR:  false,
		}
	case category == "architecture" || source == "graph":
		return recipe{
			fixStrategy: "Review disconnected or orphaned code — wire, document, or remove after confirming no runtime use",
			complexity:  ComplexityLarge,
			summary:     "Graph/architecture signal — manual review required before removal",
			blocked:     []string{"do not delete code solely because graph marks it orphaned"},
			humanReview: true,
			safeAutoPR:  false,
		}
	case category == "tech_debt" || category == "maintainability" || category == "code_quality":
		return recipe{
			fixStrategy: "Apply small refactor or lint fix; keep behavior unchanged unless tests prove otherwise",
			complexity:  ComplexitySmall,
			summary:     "Maintainability finding — targeted cleanup with tests",
			humanReview: ctx.FromAI,
			safeAutoPR:  !ctx.FromAI,
		}
	case category == "reliability" || category == "performance":
		return recipe{
			fixStrategy: "Address reliability/performance smell with minimal change and regression tests",
			complexity:  ComplexityMedium,
			summary:     "Reliability or performance finding — fix with validation tests",
			humanReview: true,
			safeAutoPR:  false,
		}
	default:
		return recipe{
			fixStrategy: "Review finding, apply minimal targeted fix, and validate with project test suite",
			complexity:  ComplexityMedium,
			summary:     "Review finding and apply conservative fix with tests",
			humanReview: true,
			safeAutoPR:  false,
		}
	}
}

func affectedFiles(ctx FindingContext) []string {
	if ctx.FilePath == "" {
		return nil
	}
	return []string{ctx.FilePath}
}
