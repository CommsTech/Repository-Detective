package remediation_test

import (
	"context"
	"testing"

	"git.commsnet.org/commstech/repository-detective/remediation"
)

type fakeChat struct {
	content string
	err     error
}

func (f fakeChat) Chat(context.Context, remediation.ChatRequest) (*remediation.ChatResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &remediation.ChatResult{Content: f.content}, nil
}

func TestProviderAdvisorSuggestPlan(t *testing.T) {
	advisor := remediation.ProviderAdvisor{
		Client: fakeChat{content: `{
			"summary": "Bump vulnerable dependency and re-run govulncheck",
			"fix_strategy": "Update package to patched version",
			"affected_files": ["go.mod", "go.sum"],
			"required_tests": ["go test ./..."],
			"validation_commands": ["go test ./...", "govulncheck ./..."],
			"regression_risk": "medium",
			"fix_complexity": "small",
			"blocked_reasons": [],
			"requires_human_review": true
		}`},
		MaxTok: 500,
	}
	out, err := advisor.SuggestPlan(remediation.FindingContext{
		Title: "vuln", Category: "security", Source: "semgrep", Severity: "high", Confidence: 0.9,
	}, remediation.Plan{
		Summary: "draft", FixStrategy: "old", RegressionRisk: remediation.RiskHigh, FixComplexity: remediation.ComplexityLarge,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Summary != "Bump vulnerable dependency and re-run govulncheck" {
		t.Fatalf("summary=%q", out.Summary)
	}
	if out.FixStrategy != "Update package to patched version" {
		t.Fatalf("fix_strategy=%q", out.FixStrategy)
	}
	if !out.Advisory || out.SafeForAutoPR {
		t.Fatalf("expected advisory=true safe_for_auto_pr=false")
	}
	if out.RegressionRisk != remediation.RiskMedium || out.FixComplexity != remediation.ComplexitySmall {
		t.Fatalf("risk/complexity not applied: %s/%s", out.RegressionRisk, out.FixComplexity)
	}
}
