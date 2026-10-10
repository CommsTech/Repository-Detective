package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
)

type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
	System      string             `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Model   string `json:"model"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// AnthropicTransport talks to Anthropic Messages API.
type AnthropicTransport struct {
	endpointURL string
	apiKey      string
	httpClient  *http.Client
	logger      *logrus.Logger
}

// NewAnthropicTransport creates an Anthropic Messages API transport.
func NewAnthropicTransport(baseURL, apiKey string, insecureSkipTLSVerify bool, logger *logrus.Logger) *AnthropicTransport {
	return &AnthropicTransport{
		endpointURL: normalizeBaseURL(baseURL) + "/messages",
		apiKey:      apiKey,
		httpClient:  NewHTTPClient(insecureSkipTLSVerify),
		logger:      logger,
	}
}

func (t *AnthropicTransport) Name() string {
	return string(ProviderAnthropic)
}

func (t *AnthropicTransport) Complete(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	payload := anthropicRequest{
		Model:     req.Model,
		MaxTokens: req.MaxTokens,
	}
	if payload.MaxTokens <= 0 {
		payload.MaxTokens = 4096
	}
	if req.Temperature > 0 {
		payload.Temperature = req.Temperature
	}

	for _, message := range req.Messages {
		switch message.Role {
		case "system":
			if payload.System == "" {
				payload.System = message.Content
			} else {
				payload.System += "\n\n" + message.Content
			}
		default:
			role := message.Role
			if role == "assistant" {
				role = "assistant"
			} else if role != "assistant" {
				role = "user"
			}
			payload.Messages = append(payload.Messages, anthropicMessage{
				Role:    role,
				Content: message.Content,
			})
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.endpointURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", t.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed anthropicResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var content string
	for _, block := range parsed.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}
	if content == "" {
		return nil, fmt.Errorf("anthropic returned empty content")
	}

	model := parsed.Model
	if model == "" {
		model = req.Model
	}

	return &ChatResponse{
		Model:   model,
		Content: content,
	}, nil
}
