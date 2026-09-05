package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Stream      bool                `json:"stream"`
	Temperature float64             `json:"temperature,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// OpenAICompatibleTransport talks to OpenAI-compatible /v1/chat/completions APIs.
type OpenAICompatibleTransport struct {
	name         string
	endpointURL  string
	apiKey       string
	extraHeaders map[string]string
	httpClient   *http.Client
	logger       *logrus.Logger
}

// NewOpenAICompatibleTransport creates a transport for OpenAI-style APIs.
func NewOpenAICompatibleTransport(name, baseURL, apiKey string, extraHeaders map[string]string, insecureSkipTLSVerify bool, logger *logrus.Logger) *OpenAICompatibleTransport {
	base := normalizeBaseURL(baseURL)
	endpoint := base
	if !strings.HasSuffix(base, "/chat/completions") {
		endpoint = base + "/chat/completions"
	}

	headers := map[string]string{}
	for key, value := range extraHeaders {
		headers[key] = value
	}

	return &OpenAICompatibleTransport{
		name:         name,
		endpointURL:  endpoint,
		apiKey:       apiKey,
		extraHeaders: headers,
		httpClient:   NewHTTPClient(insecureSkipTLSVerify),
		logger:       logger,
	}
}

func (t *OpenAICompatibleTransport) Name() string {
	return t.name
}

func (t *OpenAICompatibleTransport) Complete(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	payload := openAIChatRequest{
		Model:       req.Model,
		Stream:      false,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
	for _, message := range req.Messages {
		payload.Messages = append(payload.Messages, openAIChatMessage(message))
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
	if t.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+t.apiKey)
	}
	for key, value := range t.extraHeaders {
		httpReq.Header.Set(key, value)
	}

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
		return nil, fmt.Errorf("%s returned status %d: %s", t.name, resp.StatusCode, string(respBody))
	}

	var parsed openAIChatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("%s returned no choices", t.name)
	}

	model := parsed.Model
	if model == "" {
		model = req.Model
	}

	return &ChatResponse{
		Model:   model,
		Content: parsed.Choices[0].Message.Content,
	}, nil
}
