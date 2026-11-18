package analyzers

import (
	"context"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"github.com/sirupsen/logrus"
)

var testLogger = func() *logrus.Logger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	l.SetLevel(logrus.ErrorLevel)
	return l
}()

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

type countingTransport struct {
	calls atomic.Int32
}

func (c *countingTransport) Name() string { return "counting" }

func (c *countingTransport) Complete(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	c.calls.Add(1)
	return &ai.ChatResponse{Content: `{"entry_points":[],"attack_surface":[],"trust_boundaries":[]}`}, nil
}

func TestPrepareDeterministicOnlyNoAICalls(t *testing.T) {
	transport := &countingTransport{}
	client := ai.NewClientWithTransport(transport, "test-model", testLogger)
	engine := NewEngine(nil, nil, client, &Config{
		AnalysisDepth:     1,
		EnableLLMAuditors: false,
		EnableSecurity:    true,
	}, testLogger)

	_, err := engine.Prepare(testContext(t), "owner", "repo", "main", []string{"src/app.go"}, false)
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	if transport.calls.Load() != 0 {
		t.Fatalf("expected zero AI calls, got %d", transport.calls.Load())
	}
}

func TestValidateStaticFindingNoAICalls(t *testing.T) {
	transport := &countingTransport{}
	client := ai.NewClientWithTransport(transport, "test-model", testLogger)
	engine := NewEngine(nil, nil, client, &Config{
		AnalysisDepth:     1,
		EnableLLMAuditors: false,
	}, testLogger)

	validated, err := engine.validateOne(testContext(t), CandidateFinding{
		ID:          "static-1",
		Hypothesis:  "Hardcoded secret",
		AuditorType: "static",
		Confidence:  0.92,
		Severity:    "high",
		File:        "config.go",
		Line:        10,
	})
	if err != nil {
		t.Fatalf("validateOne failed: %v", err)
	}
	if validated == nil || validated.DebateResult.Outcome != "validated" {
		t.Fatalf("expected validated static finding, got %#v", validated)
	}
	if transport.calls.Load() != 0 {
		t.Fatalf("expected zero AI calls, got %d", transport.calls.Load())
	}
}

func TestValidateHealthFindingNoAICalls(t *testing.T) {
	transport := &countingTransport{}
	client := ai.NewClientWithTransport(transport, "test-model", testLogger)
	engine := NewEngine(nil, nil, client, &Config{
		AnalysisDepth:     3,
		EnableLLMAuditors: true,
	}, testLogger)

	validated, err := engine.validateOne(testContext(t), CandidateFinding{
		ID:          "health-1",
		Hypothesis:  "Technical debt marker found in code",
		AuditorType: "tech_debt",
		Category:    "tech_debt",
		Confidence:  0.7,
		Severity:    "low",
		File:        "main.go",
		Line:        3,
	})
	if err != nil {
		t.Fatalf("validateOne failed: %v", err)
	}
	if validated == nil || validated.DebateResult.Outcome != "validated" {
		t.Fatalf("expected validated health finding, got %#v", validated)
	}
	if transport.calls.Load() != 0 {
		t.Fatalf("expected zero AI calls, got %d", transport.calls.Load())
	}
}

func TestProveDeterministicNoAICalls(t *testing.T) {
	transport := &countingTransport{}
	client := ai.NewClientWithTransport(transport, "test-model", testLogger)
	engine := NewEngine(nil, nil, client, &Config{
		AnalysisDepth:     1,
		EnableLLMAuditors: false,
	}, testLogger)

	proven, err := engine.Prove(testContext(t), []DedupedFinding{{
		ID:          "trivy-1",
		AuditorType: "trivy",
		Title:       "CVE",
		Evidence:    Evidence{Code: "pkg@1.0.0"},
	}})
	if err != nil {
		t.Fatalf("prove failed: %v", err)
	}
	if len(proven) != 1 {
		t.Fatalf("expected one proven finding, got %d", len(proven))
	}
	if transport.calls.Load() != 0 {
		t.Fatalf("expected zero AI calls, got %d", transport.calls.Load())
	}
	if proven[0].ID != "trivy-1" {
		t.Fatalf("expected original finding ID to pass through, got %q", proven[0].ID)
	}
	if proven[0].ProofOfConcept.Type != "scanner" {
		t.Fatalf("expected scanner proof type, got %q", proven[0].ProofOfConcept.Type)
	}
}
