package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Embedder generates vector embeddings for semantic dedup.
type Embedder struct {
	baseURL    string
	apiKey     string
	model      string
	dimensions int
	httpClient *http.Client
}

// EmbedderConfig configures the embedding HTTP client.
type EmbedderConfig struct {
	BaseURL    string
	APIKey     string
	Model      string
	Dimensions int
	Timeout    time.Duration
}

// NewEmbedder creates an OpenAI-compatible embedding client.
func NewEmbedder(cfg EmbedderConfig) *Embedder {
	if cfg.Model == "" {
		cfg.Model = "text-embedding-3-small"
	}
	if cfg.Dimensions <= 0 {
		cfg.Dimensions = 1536
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	base := strings.TrimRight(cfg.BaseURL, "/")
	return &Embedder{
		baseURL:    base,
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		dimensions: cfg.Dimensions,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}
}

// Enabled reports whether embeddings can be requested.
func (e *Embedder) Enabled() bool {
	return e != nil && e.baseURL != ""
}

// Embed returns a vector for the given text.
func (e *Embedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if !e.Enabled() {
		return nil, fmt.Errorf("embedder is not configured")
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("empty embedding text")
	}

	endpoint := e.baseURL
	if !strings.HasSuffix(endpoint, "/embeddings") {
		if strings.HasSuffix(endpoint, "/v1") {
			endpoint += "/embeddings"
		} else {
			endpoint += "/v1/embeddings"
		}
	}

	payload, err := json.Marshal(map[string]any{
		"model":           e.model,
		"input":           text,
		"dimensions":      e.dimensions,
		"encoding_format": "float",
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("embedding API returned status %d: %s", resp.StatusCode, string(raw))
	}

	var parsed struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Data) == 0 || len(parsed.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("embedding API returned no vectors")
	}

	vector := make([]float32, len(parsed.Data[0].Embedding))
	for i, value := range parsed.Data[0].Embedding {
		vector[i] = float32(value)
	}
	return vector, nil
}
