package remediation

import (
	"context"
	"testing"
)

func enabledPlanner() *Planner {
	return NewPlanner(Config{
		Enabled:       true,
		MinSeverity:   "medium",
		MinConfidence: 0.5,
	}, nil, func() string { return "rp-test" })
}

func TestHadolintContainerCategorySafeForAutoPR(t *testing.T) {
	plan := ApplyRecipe(FindingContext{Category: "container", Source: "hadolint", RuleID: "DL3018", Severity: "medium", FilePath: "Dockerfile", Confidence: 0.9})
	if !plan.SafeForAutoPR {
		t.Fatal("hadolint container findings should be safe for auto PR when low risk")
	}
}

func TestSecretRecipeHumanReviewNoAutoPR(t *testing.T) {
	plan := ApplyRecipe(FindingContext{Category: "secret", Source: "gitleaks", Severity: "high", Confidence: 0.95})
	if plan.SafeForAutoPR {
		t.Fatal("secrets must not be auto-PR safe")
	}
	if !plan.RequiresHumanReview {
		t.Fatal("secrets require human review")
	}
}

func TestGovulncheckDependencyPlan(t *testing.T) {
	plan := ApplyRecipe(FindingContext{
		Category: "dependency", Source: "govulncheck", Severity: "high",
		PackageName: "example.com/mod", Confidence: 0.9,
	})
	if plan.FixStrategy == "" {
		t.Fatal("expected fix strategy")
	}
	if len(plan.ValidationCommands) == 0 && len(plan.RequiredTests) == 0 {
		t.Fatal("expected validation guidance")
	}
}

func TestGosecHighRiskHumanReview(t *testing.T) {
	plan := ApplyRecipe(FindingContext{Category: "security", Source: "gosec", Severity: "high", RuleID: "G104", Confidence: 0.9})
	if !plan.RequiresHumanReview {
		t.Fatal("gosec should require human review")
	}
	if plan.SafeForAutoPR {
		t.Fatal("gosec should not be auto-PR safe")
	}
}

func TestStaticcheckSimpleCandidate(t *testing.T) {
	plan := ApplyRecipe(FindingContext{Category: "code_quality", Source: "staticcheck", Severity: "medium", RuleID: "SA1006", Confidence: 0.95})
	if plan.FixComplexity != ComplexitySmall {
		t.Fatalf("expected small complexity, got %s", plan.FixComplexity)
	}
}

func TestHadolintDockerfilePlan(t *testing.T) {
	plan := ApplyRecipe(FindingContext{Category: "misconfiguration", Source: "hadolint", Severity: "medium", FilePath: "Dockerfile", Confidence: 0.9})
	found := false
	for _, cmd := range plan.ValidationCommands {
		if cmd == "hadolint Dockerfile" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected hadolint validation command")
	}
}

func TestStaticcheckPackageScopedValidation(t *testing.T) {
	plan := ApplyRecipe(FindingContext{
		Category: "code_quality", Source: "staticcheck", Severity: "low", RuleID: "S1039",
		FilePath: "internal/dogfood/staticcheck_e2e_marker.go", Confidence: 0.9,
	})
	want := []string{"go test ./internal/dogfood/...", "staticcheck ./internal/dogfood/..."}
	for _, cmd := range want {
		found := false
		for _, got := range plan.ValidationCommands {
			if got == cmd {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected %q in validation commands, got %v", cmd, plan.ValidationCommands)
		}
	}
}

func TestCheckovIACPlan(t *testing.T) {
	plan := ApplyRecipe(FindingContext{Category: "misconfiguration", Source: "checkov", Severity: "high", Confidence: 0.9})
	found := false
	for _, cmd := range plan.ValidationCommands {
		if cmd == "checkov -d ." {
			found = true
		}
	}
	if !found {
		t.Fatal("expected checkov validation command")
	}
}

func TestTestGapPlan(t *testing.T) {
	plan := ApplyRecipe(FindingContext{Category: "test_gap", Source: "test_gap", Severity: "medium", Confidence: 0.85})
	if !plan.RequiresHumanReview {
		t.Fatal("test gap should require human review")
	}
}

func TestGraphOrphanReviewOnly(t *testing.T) {
	plan := ApplyRecipe(FindingContext{Category: "architecture", Source: "graph", Severity: "medium", Confidence: 0.9})
	if plan.SafeForAutoPR {
		t.Fatal("graph orphan must not be auto-PR safe")
	}
}

func TestPlannerDisabledSuppresses(t *testing.T) {
	p := NewPlanner(Config{Enabled: false}, nil, nil)
	if p.ShouldPlan(FindingContext{Severity: "critical", Confidence: 1, ConnectedRepo: true}) {
		t.Fatal("disabled planner should not plan")
	}
}

func TestLowFindingNotPlannedByDefault(t *testing.T) {
	p := enabledPlanner()
	if p.ShouldPlan(FindingContext{Severity: "low", Confidence: 0.5, ConnectedRepo: true}) {
		t.Fatal("low finding should not plan by default")
	}
}

func TestGeneratePlan(t *testing.T) {
	p := enabledPlanner()
	plan, err := p.Generate(context.Background(), FindingContext{
		FindingID: 1, RepositoryID: 1, ConnectedRepo: true,
		Category: "code_quality", Source: "staticcheck", Severity: "medium", Confidence: 0.9,
		Title: "unused parameter",
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.ID == "" || plan.Status != StatusProposed {
		t.Fatalf("unexpected plan: %+v", plan)
	}
}

func TestAIAdvisoryWhenEnabled(t *testing.T) {
	p := NewPlanner(Config{
		Enabled: true, MinSeverity: "medium", MinConfidence: 0.5,
		UseAI: true, GlobalAIAllowed: true,
	}, StubAIAdvisor{}, func() string { return "rp-ai" })
	plan, err := p.Generate(context.Background(), FindingContext{
		FindingID: 1, RepositoryID: 1, ConnectedRepo: true,
		Category: "unknown", Source: "custom", Severity: "high", Confidence: 0.95,
		Title: "unknown issue",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Advisory {
		t.Fatal("expected advisory flag when AI stub used")
	}
}

func TestRenderIssueCommentSafe(t *testing.T) {
	body := RenderIssueComment(Plan{
		FixStrategy:    "Remove secret and rotate credential",
		RequiredTests:  []string{"Re-run secret scanner"},
		RegressionRisk: RiskHigh,
		SafeForAutoPR:  false,
	})
	if body == "" || !contains(body, "Planning only") {
		t.Fatal("expected safe issue comment")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
