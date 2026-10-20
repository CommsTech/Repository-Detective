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
	// User is an optional OpenAI-compatible user id. OpenClaw derives a stable
	// agent session from this value (see gateway OpenAI HTTP API docs).
	User string
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

// TLSOptions bundles TLS trust settings for AI provider transports.
type TLSOptions struct {
	// InsecureSkipVerify is the legacy homelab escape hatch; it is ignored
	// when CACertPEM is set (safe CA trust always wins over skipping).
	InsecureSkipVerify bool
	// CACertPEM is an optional PEM CA bundle trusted in addition to system
	// roots. When non-empty, certificate verification stays fully enabled.
	CACertPEM string
}

// tlsOptionsFromConfig builds TLS options from a resolved provider config.
func tlsOptionsFromConfig(cfg Config) TLSOptions {
	return TLSOptions{
		InsecureSkipVerify: cfg.InsecureSkipTLSVerify,
		CACertPEM:          cfg.CACertPEM,
	}
}
