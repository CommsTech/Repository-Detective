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
		// No dependency-upgrade patcher has completed isolated integration and
		// lockfile/API-compatibility verification. Even a scanner-reported fixed
		// version is not evidence that an unattended upgrade is safe.
		if strings.Contains(strings.ToLower(ctx.Summary), "major") {
			return RiskHigh, true, []string{"major dependency upgrade requires human approval"}
		}
		if strings.TrimSpace(ctx.FixedVersion) == "" {
			return RiskMedium, true, []string{"no scanner-reported fixed version: verify a supported upgrade or mitigation"}
		}
		return RiskMedium, true, nil
	case "architecture":
		return RiskHigh, true, []string{"architecture changes need design review"}
	case "security":
		if ctx.FromAI || source == "gosec" {
			return RiskHigh, true, []string{"security finding requires validated fix and review"}
		}
		return RiskMedium, true, nil
	case "test_gap":
		return RiskLow, true, nil
	case "misconfiguration", "container":
		// Deterministic Dockerfile/IaC patches (DS029 / hadolint) are low-risk when small.
		if complexity == ComplexitySmall && (source == "hadolint" || source == "trivy" || source == "checkov") {
			return RiskLow, false, nil
		}
		if severity == "critical" || severity == "high" {
			return RiskMedium, true, nil
		}
		return RiskLow, false, nil
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
	case "dependency", "dependency_vulnerability":
		// Do not advertise auto-PR eligibility until an isolated executor can
		// validate manifest + lockfile changes, API compatibility and rescans.
		return false
	case "code_quality", "maintainability", "tech_debt":
		return complexity == ComplexitySmall && regressionRisk == RiskLow && !ctx.FromAI
	case "misconfiguration", "container":
		source := strings.ToLower(ctx.Source)
		if source == "hadolint" || source == "checkov" || source == "trivy" {
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
	if category == "secret" || category == "dependency" || category == "dependency_vulnerability" {
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
