package remediation

import (
	"strings"
	"testing"
)

// Dependency upgrades must never be advertised as safe unattended PRs
// while the executor lacks isolated compatibility, lockfile, and re-scan gates.
func TestDependencyUpgradeAlwaysNeedsReview(t *testing.T) {
	cases := []FindingContext{
		{Category: "dependency", Source: "trivy", Severity: "low", PackageName: "example", FixedVersion: "1.2.3"},
		{Category: "dependency_vulnerability", Source: "grype", Severity: "high", PackageName: "example", FixedVersion: "1.2.3"},
		{Category: "dependency", Source: "govulncheck", Severity: "critical", PackageName: "example"},
		{Category: "dependency", Source: "trivy", Severity: "high", Summary: "major version required", FixedVersion: "2.0.0"},
	}
	for _, finding := range cases {
		plan := ApplyRecipe(finding)
		if !plan.RequiresHumanReview || plan.SafeForAutoPR {
			t.Errorf("finding %+v advertised unsafe dependency automation: %+v", finding, plan)
		}
		if SafeForAutoPR(finding, ComplexitySmall, RiskLow, nil) {
			t.Errorf("direct eligibility bypass for %+v", finding)
		}
		if !strings.Contains(strings.ToLower(plan.FixStrategy), "lockfiles") ||
			!strings.Contains(strings.ToLower(plan.FixStrategy), "rescan") ||
			!strings.Contains(strings.ToLower(plan.FixStrategy), "transitive") {
			t.Errorf("missing dependency remediation safeguards: %+v", plan)
		}
	}
}

func TestDependencyMissingFixRecordsBlocker(t *testing.T) {
	plan := ApplyRecipe(FindingContext{Category: "dependency", Severity: "high", Source: "grype"})
	if len(plan.BlockedReasons) == 0 {
		t.Fatal("missing safe version must produce a visible blocker")
	}
	if !strings.Contains(strings.ToLower(strings.Join(plan.BlockedReasons, " ")), "fixed version") {
		t.Fatalf("unexpected blocked reasons: %+v", plan.BlockedReasons)
	}
}

func TestMisconfigurationRecipeRemainsEligible(t *testing.T) {
	plan := ApplyRecipe(FindingContext{Category: "security", Source: "trivy", RuleID: "TRIVY-MIS-DS029", Severity: "medium", FilePath: "Dockerfile"})
	if plan.Category != "misconfiguration" || plan.RequiresHumanReview || !plan.SafeForAutoPR {
		t.Fatalf("misconfiguration routing regressed: %+v", plan)
	}
}
