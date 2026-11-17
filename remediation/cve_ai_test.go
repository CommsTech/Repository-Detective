package remediation

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type cveAdvisor struct {
	called bool
	fail   bool
}

func (a *cveAdvisor) SuggestPlan(_ FindingContext, draft Plan) (Plan, error) {
	a.called = true
	if a.fail {
		return draft, errors.New("provider offline")
	}
	draft.Summary = "AI reviewed version evidence"
	return draft, nil
}

func TestCVEDependencyAIEnrichmentOptIn(t *testing.T) {
	advisor := &cveAdvisor{}
	p := NewPlanner(Config{Enabled: true, MinSeverity: "medium", MinConfidence: .8, UseAI: true, GlobalAIAllowed: true}, advisor, nil)
	plan, err := p.Generate(context.Background(), FindingContext{Category: "dependency", Source: "trivy", RuleID: "CVE-2026-12345", PackageName: "test", Severity: "high", Confidence: .95, ConnectedRepo: true})
	if err != nil {
		t.Fatal(err)
	}
	if !advisor.called || !plan.Advisory || plan.SafeForAutoPR || !plan.RequiresHumanReview {
		t.Fatalf("unexpected CVE AI plan: %+v", plan)
	}
}

func TestCVEDeterministicWhenAIUnavailable(t *testing.T) {
	advisor := &cveAdvisor{fail: true}
	p := NewPlanner(Config{Enabled: true, MinSeverity: "medium", MinConfidence: .8, UseAI: true, GlobalAIAllowed: true}, advisor, nil)
	plan, err := p.Generate(context.Background(), FindingContext{Category: "dependency", Source: "grype", Severity: "high", Confidence: .95, ConnectedRepo: true, PackageName: "ssl", FixedVersion: "1.2.3"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Advisory || plan.SafeForAutoPR || !plan.RequiresHumanReview || len(plan.BlockedReasons) == 0 {
		t.Fatalf("failure not explicit: %+v", plan)
	}
	if !strings.Contains(plan.FixStrategy, "1.2.3") {
		t.Fatalf("scanner fixed version lost: %s", plan.FixStrategy)
	}
}

func TestCVEScanningDoesNotSpendTokensWithoutOptIn(t *testing.T) {
	advisor := &cveAdvisor{}
	p := NewPlanner(Config{Enabled: true, MinSeverity: "medium", MinConfidence: .8, UseAI: false, GlobalAIAllowed: true}, advisor, nil)
	_, err := p.Generate(context.Background(), FindingContext{Category: "dependency", Source: "trivy", Severity: "high", Confidence: .9, ConnectedRepo: true})
	if err != nil {
		t.Fatal(err)
	}
	if advisor.called {
		t.Fatal("called AI when disabled")
	}
}
