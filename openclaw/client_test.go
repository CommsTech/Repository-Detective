package openclaw_test

import (
	"context"
	"encoding/json"
	"testing"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/openclaw"
)

type stubTransport struct {
	content string
	err     error
}

func (s stubTransport) Name() string { return "stub" }
func (s stubTransport) Complete(_ context.Context, _ ai.ChatRequest) (*ai.ChatResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &ai.ChatResponse{Content: s.content, Model: "test-model"}, nil
}

func TestValidResponseStored(t *testing.T) {
	resp := openclaw.ReviewResponse{
		ReviewID: "air-test", OverallAssessment: "ok",
		Recommendations: []openclaw.Recommendation{{
			Fingerprint: "fp1", Classification: openclaw.ClassPossibleFalsePositive,
			SuggestedAction: "calibrate_repo_scope", Reason: "test",
		}},
	}
	raw, _ := json.Marshal(resp)
	cfg := openclaw.DefaultConfig()
	cfg.MaxTokensPerScan = 500
	cfg.FallbackEndpoint = "http://127.0.0.1:1/v1"
	client, err := openclaw.NewClient(cfg, stubTransport{content: string(raw)})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Review(context.Background(), cfg, "air-test", openclaw.ReviewPacket{
		Findings: []openclaw.FindingInput{{Fingerprint: "fp1", Title: "test"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "completed" || result.RecommendationsCount != 1 {
		t.Fatalf("got %+v", result)
	}
}

func TestMalformedResponseRejected(t *testing.T) {
	cfg := openclaw.DefaultConfig()
	cfg.MaxTokensPerScan = 500
	cfg.FallbackEndpoint = "http://127.0.0.1:1/v1"
	client, err := openclaw.NewClient(cfg, stubTransport{content: "not json"})
	if err != nil {
		t.Fatal(err)
	}
	result, _ := client.Review(context.Background(), cfg, "air-bad", openclaw.ReviewPacket{})
	if result.Status != "failed" || result.Error == "" {
		t.Fatalf("expected failed parse, got %+v", result)
	}
}
