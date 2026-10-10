package ai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ConnectionTestMode controls how AI connectivity is verified.
type ConnectionTestMode string

const (
	TestModeMetadataOnly   ConnectionTestMode = "metadata_only"
	TestModeManual         ConnectionTestMode = "manual"
	TestModeChatCompletion ConnectionTestMode = "chat_completion"
)

// UsageRecord tracks approximate AI usage for a scan or test.
type UsageRecord struct {
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	CallCount    int       `json:"call_count"`
	InputTokens  int       `json:"input_tokens,omitempty"`
	OutputTokens int       `json:"output_tokens,omitempty"`
	TestedAt     time.Time `json:"tested_at,omitempty"`
	Source       string    `json:"source"`
}

// ProviderStatus is exposed via /api/v1/ai/status without secrets.
type ProviderStatus struct {
	Configured     bool               `json:"configured"`
	Provider       string             `json:"provider"`
	Model          string             `json:"model"`
	PolicyDisabled bool               `json:"policy_disabled"`
	TestMode       ConnectionTestMode `json:"test_mode"`
	LastTestAt     *time.Time         `json:"last_test_at,omitempty"`
	LastTestOK     bool               `json:"last_test_ok"`
	LastTestError  string             `json:"last_test_error,omitempty"`
	LastTestSource string             `json:"last_test_source,omitempty"`
	Usage          UsageRecord        `json:"usage"`
}

var (
	statusMu     sync.RWMutex
	cachedStatus ProviderStatus
	lastTestTime time.Time
	testCacheTTL = 60 * time.Minute
)

// SetInitialProviderStatus seeds status before startup tests run.
func SetInitialProviderStatus(st ProviderStatus) {
	setProviderStatus(st)
}

// ConfigureStatusCache sets connection test cache duration.
func ConfigureStatusCache(minutes int) {
	if minutes > 0 {
		testCacheTTL = time.Duration(minutes) * time.Minute
	}
}

// GetProviderStatus returns the last known AI provider status.
func GetProviderStatus() ProviderStatus {
	statusMu.RLock()
	defer statusMu.RUnlock()
	return cachedStatus
}

func setProviderStatus(st ProviderStatus) {
	statusMu.Lock()
	defer statusMu.Unlock()
	cachedStatus = st
}

// TestMetadata performs a cheap non-generation connectivity check when possible.
func (c *Client) TestMetadata(ctx context.Context) error {
	if c == nil || c.transport == nil {
		return fmt.Errorf("ai client not configured")
	}
	if ot, ok := c.transport.(*OpenAICompatibleTransport); ok {
		return ot.TestModelsEndpoint(ctx)
	}
	return fmt.Errorf("metadata test not supported for provider %s", c.provider)
}

// TestConnectionChat uses a minimal chat completion (may incur cost).
func (c *Client) TestConnectionChat(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("ai client not configured")
	}
	_, err := c.chat(ctx, []ChatMessage{{Role: "user", Content: "ping"}}, 0, 5)
	return err
}

// RunConnectionTest executes a connection test respecting mode and cache.
func RunConnectionTest(ctx context.Context, client *Client, mode ConnectionTestMode, force bool) (ProviderStatus, error) {
	st := GetProviderStatus()
	if client == nil {
		st.Configured = false
		setProviderStatus(st)
		return st, fmt.Errorf("ai client not configured")
	}
	st.Configured = true
	st.Provider = string(client.provider)
	st.Model = client.modelName()
	st.TestMode = mode

	if !force {
		statusMu.RLock()
		cached := lastTestTime
		ok := cachedStatus.LastTestOK
		statusMu.RUnlock()
		if !cached.IsZero() && time.Since(cached) < testCacheTTL && ok {
			st.LastTestAt = &cached
			st.LastTestOK = ok
			st.LastTestSource = cachedStatus.LastTestSource
			setProviderStatus(st)
			return st, nil
		}
	}

	var err error
	switch mode {
	case TestModeChatCompletion:
		err = client.TestConnectionChat(ctx)
		st.LastTestSource = "chat_completion"
	default:
		err = client.TestMetadata(ctx)
		st.LastTestSource = "metadata_only"
		if err != nil && mode == TestModeMetadataOnly {
			st.LastTestError = err.Error()
			st.LastTestOK = false
			now := time.Now().UTC()
			st.LastTestAt = &now
			setProviderStatus(st)
			return st, fmt.Errorf("metadata test failed: %w", err)
		}
	}
	now := time.Now().UTC()
	st.LastTestAt = &now
	if err != nil {
		st.LastTestOK = false
		st.LastTestError = err.Error()
	} else {
		st.LastTestOK = true
		st.LastTestError = ""
	}
	statusMu.Lock()
	lastTestTime = now
	statusMu.Unlock()
	setProviderStatus(st)
	return st, err
}

// TestModelsEndpoint hits GET /v1/models on OpenAI-compatible APIs.
func (t *OpenAICompatibleTransport) TestModelsEndpoint(ctx context.Context) error {
	if t == nil {
		return fmt.Errorf("transport not configured")
	}
	base := strings.TrimSuffix(t.endpointURL, "/chat/completions")
	base = strings.TrimSuffix(base, "/")
	url := base
	if !strings.HasSuffix(url, "/models") {
		if strings.HasSuffix(url, "/v1") {
			url += "/models"
		} else {
			url += "/v1/models"
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if t.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+t.apiKey)
	}
	for k, v := range t.extraHeaders {
		req.Header.Set(k, v)
	}
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("models endpoint HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// RecordUsage updates global usage stats from a chat response.
func RecordUsage(provider, model, source string, inputTokens, outputTokens int) {
	statusMu.Lock()
	defer statusMu.Unlock()
	cachedStatus.Usage.Provider = provider
	cachedStatus.Usage.Model = model
	cachedStatus.Usage.Source = source
	cachedStatus.Usage.CallCount++
	cachedStatus.Usage.InputTokens += inputTokens
	cachedStatus.Usage.OutputTokens += outputTokens
	cachedStatus.Usage.TestedAt = time.Now().UTC()
}
