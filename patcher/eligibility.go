package patcher

import (
	"strings"

	"git.commsnet.org/commstech/repository-detective/remediation"
)

// SafeFixRolloutPolicy is shown in UI and docs for owned-repo remediation PRs.
const SafeFixRolloutPolicy = "Repository Detective creates PRs only for approved low-risk plans. It never auto-merges."

var forbiddenCategories = map[string]struct{}{
	"secret": {}, "hardcoded_secret": {}, "dependency": {}, "dependency_vulnerability": {},
	"architecture": {}, "security": {},
}

var forbiddenSources = map[string]struct{}{
	"gitleaks": {}, "govulncheck": {}, "grype": {}, "trivy": {}, "gosec": {}, "graph": {},
}

// CheckPREligibility applies hard rules for safe remediation PRs.
func CheckPREligibility(plan remediation.Plan, repo RepoContext, cfg Config) EligibilityResult {
	checks := map[string]bool{}
	var blocked []string

	addBlock := func(key, reason string) {
		checks[key] = false
		blocked = append(blocked, reason)
	}
	pass := func(key string) { checks[key] = true }

	if !cfg.Enabled {
		addBlock("enabled", "remediation PR feature disabled globally")
	} else {
		pass("enabled")
	}

	if cfg.RequireApproval {
		if plan.Status != remediation.StatusApproved {
			addBlock("approved", "plan must be approved")
		} else {
			pass("approved")
		}
	} else {
		pass("approved")
	}

	if !plan.SafeForAutoPR {
		addBlock("safe_for_auto_pr", "safe_for_auto_pr must be true")
	} else {
		pass("safe_for_auto_pr")
	}
	if plan.RequiresHumanReview {
		addBlock("requires_human_review", "requires_human_review must be false")
	} else {
		pass("requires_human_review")
	}
	if strings.ToLower(plan.RegressionRisk) != remediation.RiskLow {
		addBlock("regression_risk", "regression_risk must be low")
	} else {
		pass("regression_risk")
	}
	if strings.ToLower(plan.FixComplexity) != remediation.ComplexitySmall {
		addBlock("fix_complexity", "fix_complexity must be small")
	} else {
		pass("fix_complexity")
	}
	if !repo.ConnectedRepo {
		addBlock("connected_repo", "connected repository required")
	} else {
		pass("connected_repo")
	}
	if plan.AuditID != "" {
		addBlock("audit_only", "pre-install audit plans cannot create PRs")
	} else {
		pass("audit_only")
	}
	if plan.Advisory {
		addBlock("advisory", "advisory plans cannot create PRs")
	} else {
		pass("advisory")
	}

	category := strings.ToLower(plan.Category)
	if _, ok := forbiddenCategories[category]; ok {
		addBlock("category", "forbidden finding category: "+category)
	} else {
		pass("category")
	}
	source := strings.ToLower(plan.Source)
	if _, ok := forbiddenSources[source]; ok {
		addBlock("source", "forbidden finding source: "+source)
	} else {
		pass("source")
	}
	if strings.Contains(strings.ToLower(plan.Summary), "major") && category == "dependency" {
		addBlock("dependency_major", "dependency major upgrade not allowed")
	} else {
		pass("dependency_major")
	}

	hasValidation := false
	for _, cmd := range plan.ValidationCommands {
		if _, err := ParseAllowedCommand(cmd); err == nil {
			hasValidation = true
			break
		}
	}
	if !hasValidation {
		addBlock("validation", "at least one allowlisted validation command required")
	} else {
		pass("validation")
	}

	if len(plan.BlockedReasons) > 0 {
		addBlock("plan_blocked", "plan has blocked reasons")
	} else {
		pass("plan_blocked")
	}

	if !SupportsPatch(plan) {
		addBlock("patcher", "no patcher available for this rule yet")
	} else {
		pass("patcher")
	}

	severity := strings.ToLower(strings.TrimSpace(plan.Severity))
	if cfg.BlockHighCriticalWithoutOverride && (severity == "high" || severity == "critical") {
		addBlock("severity", "high/critical findings blocked without manual override")
	} else if len(cfg.AllowedSeverities) > 0 && severity != "" {
		allowed := false
		for _, s := range cfg.AllowedSeverities {
			if strings.ToLower(strings.TrimSpace(s)) == severity {
				allowed = true
				break
			}
		}
		if !allowed {
			addBlock("severity", "severity not in remediation_pr_allowed_severities")
		} else {
			pass("severity")
		}
	} else {
		pass("severity")
	}

	return EligibilityResult{
		Eligible:       len(blocked) == 0,
		BlockedReasons: blocked,
		Checklist:      checks,
	}
}
