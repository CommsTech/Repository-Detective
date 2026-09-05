package remediation

import "strings"

// AssessRisk returns regression risk and whether human review is required.
func AssessRisk(ctx FindingContext, complexity string) (regressionRisk string, requiresHuman bool, blocked []string) {
	category := strings.ToLower(ctx.Category)
	source := strings.ToLower(ctx.Source)
	severity := strings.ToLower(ctx.Severity)

	switch category {
	case "secret", "hardcoded_secret":
		return RiskHigh, true, []string{"secrets require out-of-band rotation and history review"}
	case "dependency", "dependency_vulnerability":
		if strings.Contains(strings.ToLower(ctx.Summary), "major") {
			return RiskHigh, true, []string{"major dependency upgrade requires human approval"}
		}
		return RiskMedium, severity == "critical", nil
	case "architecture":
		return RiskHigh, true, []string{"architecture changes need design review"}
	case "security":
		if ctx.FromAI || source == "gosec" {
			return RiskHigh, true, []string{"security finding requires validated fix and review"}
		}
		return RiskMedium, true, nil
	case "test_gap":
		return RiskLow, true, nil
	default:
		if complexity == ComplexityLarge {
			return RiskHigh, true, []string{"large change scope requires review"}
		}
		if severity == "critical" || severity == "high" {
			return RiskMedium, true, nil
		}
		return RiskLow, false, nil
	}
}

// SafeForAutoPR determines whether a future automated PR phase may consider this plan.
func SafeForAutoPR(ctx FindingContext, complexity, regressionRisk string, blocked []string) bool {
	if len(blocked) > 0 {
		return false
	}
	if ctx.RequiresHumanReviewFlag() {
		return false
	}
	category := strings.ToLower(ctx.Category)
	switch category {
	case "secret", "architecture", "security":
		return false
	case "dependency":
		if regressionRisk == RiskHigh {
			return false
		}
		return complexity == ComplexitySmall && !ctx.FromAI
	case "code_quality", "maintainability", "tech_debt":
		return complexity == ComplexitySmall && regressionRisk == RiskLow && !ctx.FromAI
	case "misconfiguration", "container":
		source := strings.ToLower(ctx.Source)
		if source == "hadolint" || source == "checkov" {
			return complexity == ComplexitySmall && regressionRisk != RiskHigh
		}
		return false
	default:
		return false
	}
}

// RequiresHumanReviewFlag reports conservative human-review signals on context.
func (ctx FindingContext) RequiresHumanReviewFlag() bool {
	category := strings.ToLower(ctx.Category)
	if category == "secret" {
		return true
	}
	if ctx.FromAI {
		return true
	}
	if strings.ToLower(ctx.Source) == "gosec" {
		return true
	}
	return false
}
