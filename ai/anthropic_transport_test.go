package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestAnthropicTransportComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "anthropic-key" {
			t.Fatalf("missing anthropic api key header")
		}

		body, _ := io.ReadAll(r.Body)
		var req anthropicRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.System == "" {
			t.Fatal("expected system prompt")
		}
		if req.Model != "claude-3-5-haiku-latest" {
			t.Fatalf("unexpected model: %s", req.Model)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(anthropicResponse{
			Model: "claude-3-5-haiku-latest",
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{{Type: "text", Text: `{"confidence":0.9,"arguments":"valid"}`}},
		})
	}))
	defer server.Close()

	transport := NewAnthropicTransport(server.URL+"/v1", "anthropic-key", false, logrus.New())
	resp, err := transport.Complete(context.Background(), ChatRequest{
		Model: "claude-3-5-haiku-latest",
		Messages: []ChatMessage{
			{Role: "system", Content: "You are a security expert."},
			{Role: "user", Content: "Analyze this claim."},
		},
		MaxTokens: 1000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content == "" {
		t.Fatal("expected response content")
	}
}
