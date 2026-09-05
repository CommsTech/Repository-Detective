package ai

import "context"

// ChatMessage is a single message in a chat completion request.
type ChatMessage struct {
	Role    string
	Content string
}

// ChatRequest is a provider-agnostic chat completion request.
type ChatRequest struct {
	Model       string
	Messages    []ChatMessage
	Temperature float64
	MaxTokens   int
}

// ChatResponse is a normalized chat completion response.
type ChatResponse struct {
	Model   string
	Content string
}

// ChatTransport sends chat completion requests to an AI backend.
type ChatTransport interface {
	Name() string
	Complete(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}
