package ai_test

import (
	"context"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/ai"
	"github.com/sirupsen/logrus"
)

type cannedTransport struct{ body string }

func (c cannedTransport) Name() string { return "canned" }
func (c cannedTransport) Complete(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	return &ai.ChatResponse{Content: c.body}, nil
}

func TestAIInvalidOutputNeverBecomesSuccessfulScan(t *testing.T) {
	for _, body := range []string{"not json", "null", "{}", `{"findings":null}`} {
		t.Run(body, func(t *testing.T) {
			c := ai.NewClientWithTransport(cannedTransport{body}, "canned", logrus.New())
			_, err := c.RunAuditor(context.Background(), &ai.AuditorRequest{AuditorType: "sql", VulnerabilityClass: "sql injection"})
			if err == nil {
				t.Fatalf("auditor accepted unparseable/incomplete response %q", body)
			}
		})
	}
}

func TestCodeAnalysisMalformedOutputIsFailureNotFabricatedFinding(t *testing.T) {
	c := ai.NewClientWithTransport(cannedTransport{"garbled"}, "canned", logrus.New())
	result, err := c.AnalyzeCode(context.Background(), &ai.CodeAnalysisRequest{RepositoryName: "test", FilePath: "a.go", CodeContent: "package a"})
	if err == nil || result != nil {
		t.Fatalf("malformed output returned result %v err %v", result, err)
	}
	if strings.Contains(err.Error(), "Analysis Response Parsing Failed") {
		t.Fatal("fabricated finding")
	}
}

func TestAttackSurfaceRequiresCompleteSchemaAndDecodesSnakeCase(t *testing.T) {
	for _, body := range []string{"{}", "null", `{"entry_points":[],"attack_surface":[]}`, `{"entry_points":null,"attack_surface":[],"trust_boundaries":[]}`} {
		c := ai.NewClientWithTransport(cannedTransport{body}, "canned", logrus.New())
		if _, err := c.AnalyzeAttackSurface(context.Background(), &ai.AttackSurfaceRequest{RepositoryName: "repo"}); err == nil {
			t.Fatalf("accepted incomplete attack surface schema %q", body)
		}
	}
	c := ai.NewClientWithTransport(cannedTransport{`{"entry_points":[{"file":"cmd/main.go","function_name":"main","auth_required":false}],"attack_surface":[],"trust_boundaries":[]}`}, "canned", logrus.New())
	got, err := c.AnalyzeAttackSurface(context.Background(), &ai.AttackSurfaceRequest{RepositoryName: "repo"})
	if err != nil || len(got.EntryPoints) != 1 || got.EntryPoints[0].File != "cmd/main.go" || got.EntryPoints[0].FunctionName != "main" {
		t.Fatalf("failed to decode snake_case attack surface evidence: %+v err=%v", got, err)
	}
}

func TestCodeReviewNestedOrNullFindingListNotAcceptedAsClean(t *testing.T) {
	for _, body := range []string{`{"details":{"issues":[]}}`, `{"issues":null,"suggestions":[]}`, `{"issues":{},"suggestions":[]}`} {
		c := ai.NewClientWithTransport(cannedTransport{body}, "canned", logrus.New())
		if result, err := c.AnalyzeCode(context.Background(), &ai.CodeAnalysisRequest{RepositoryName: "r", FilePath: "main.go", CodeContent: "package main"}); err == nil || result != nil {
			t.Fatalf("nested or null analysis accepted: %q", body)
		}
	}
}
