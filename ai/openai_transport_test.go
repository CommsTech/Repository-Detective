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

func TestOpenAICompatibleTransportComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Fatalf("unexpected auth header: %s", auth)
		}

		body, _ := io.ReadAll(r.Body)
		var req openAIChatRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.Model != "gpt-4o-mini" {
			t.Fatalf("unexpected model: %s", req.Model)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(openAIChatResponse{
			Model: "gpt-4o-mini",
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{{Message: struct {
				Content string `json:"content"`
			}{Content: `{"issues":[]}`}}},
		})
	}))
	defer server.Close()

	transport := NewOpenAICompatibleTransport("openai", server.URL+"/v1", "test-token", nil, false, logrus.New())
	resp, err := transport.Complete(context.Background(), ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []ChatMessage{
			{Role: "user", Content: "hello"},
		},
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != `{"issues":[]}` {
		t.Fatalf("unexpected content: %s", resp.Content)
	}
}

func TestClientUsesTransport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(openAIChatResponse{
			Model: "test-model",
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{{Message: struct {
				Content string `json:"content"`
			}{Content: `{"findings":[]}`}}},
		})
	}))
	defer server.Close()

	transport := NewOpenAICompatibleTransport("test", server.URL+"/v1", "", nil, false, logrus.New())
	client := NewClientWithTransport(transport, "test-model", logrus.New())

	resp, err := client.RunAuditor(context.Background(), &AuditorRequest{
		RepositoryName:     "org/repo",
		VulnerabilityClass: "sql_injection",
		AuditorType:        "sql",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected auditor response")
	}
}
