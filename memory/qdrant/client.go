package qdrant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrCollectionMismatch is returned when an existing Qdrant collection does not match config.
var ErrCollectionMismatch = errors.New("qdrant collection vector size mismatch")

// Config connects Bugbot to an existing Qdrant server.
type Config struct {
	Enabled             bool
	URL                 string
	APIKey              string
	Collection          string
	VectorSize          int
	SimilarityThreshold float64
	TimeoutSeconds      int
}

// DefaultConfig returns defaults for cah_findings compatibility.
func DefaultConfig() Config {
	return Config{
		Enabled:             false,
		URL:                 "http://127.0.0.1:6333",
		Collection:          "cah_findings",
		VectorSize:          1024,
		SimilarityThreshold: 0.7,
		TimeoutSeconds:      15,
	}
}

// Client talks to Qdrant over HTTP.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

// NewClient creates a Qdrant HTTP client.
func NewClient(cfg Config) *Client {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Enabled reports whether semantic dedup is active.
func (c *Client) Enabled() bool {
	return c != nil && c.cfg.Enabled && c.cfg.URL != ""
}

// EnsureCollection creates the collection if missing.
func (c *Client) EnsureCollection(ctx context.Context) error {
	if !c.Enabled() {
		return nil
	}

	status, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/collections/%s", c.cfg.Collection), nil)
	if err == nil && status == http.StatusOK {
		return nil
	}

	body := map[string]any{
		"vectors": map[string]any{
			"size":     c.vectorSize(),
			"distance": "Cosine",
		},
		"on_disk_payload": true,
	}
	_, err = c.do(ctx, http.MethodPut, fmt.Sprintf("/collections/%s", c.cfg.Collection), body)
	return err
}

// ValidateCollection verifies an existing collection matches configured vector size.
func (c *Client) ValidateCollection(ctx context.Context) error {
	if !c.Enabled() {
		return nil
	}
	respBody, err := c.doRaw(ctx, http.MethodGet, fmt.Sprintf("/collections/%s", c.cfg.Collection), nil)
	if err != nil {
		return err
	}
	var parsed struct {
		Result struct {
			Config struct {
				Params struct {
					Vectors struct {
						Size json.Number `json:"size"`
					} `json:"vectors"`
				} `json:"params"`
			} `json:"config"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return err
	}
	size, err := parsed.Result.Config.Params.Vectors.Size.Int64()
	if err != nil {
		return fmt.Errorf("qdrant collection size unreadable: %w", err)
	}
	if int(size) != c.vectorSize() {
		return fmt.Errorf("%w: collection=%s have=%d want=%d", ErrCollectionMismatch, c.cfg.Collection, size, c.vectorSize())
	}
	return nil
}

// ValidateVectorLen rejects vectors that do not match configured dimensions.
func (c *Client) ValidateVectorLen(length int) error {
	if !c.Enabled() {
		return nil
	}
	want := c.vectorSize()
	if length != want {
		return fmt.Errorf("%w: vector length=%d want=%d", ErrCollectionMismatch, length, want)
	}
	return nil
}

// FindingPayload is the legacy payload stored alongside each vector in Qdrant.
type FindingPayload struct {
	Repository  string  `json:"repository"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	File        string  `json:"file"`
	Line        int     `json:"line"`
	Severity    string  `json:"severity"`
	Category    string  `json:"category"`
	Confidence  float64 `json:"confidence"`
	IssueURL    string  `json:"issue_url"`
	IssueNumber int     `json:"issue_number"`
	ClusterID   string  `json:"cluster_id"`
	Fingerprint string  `json:"fingerprint"`
}

// ScoredFinding is a Qdrant search hit.
type ScoredFinding struct {
	ID      string
	Score   float64
	Payload map[string]any
}

// Upsert stores or updates a finding vector with a redacted cah_findings payload.
func (c *Client) Upsert(ctx context.Context, pointID string, vector []float32, payload CAHFindingPayload) error {
	if !c.Enabled() {
		return nil
	}
	if err := c.ValidateVectorLen(len(vector)); err != nil {
		return err
	}
	payloadMap, err := PayloadToMap(payload)
	if err != nil {
		return err
	}

	body := map[string]any{
		"points": []map[string]any{
			{
				"id":      pointID,
				"vector":  vector,
				"payload": payloadMap,
			},
		},
	}
	_, err = c.do(ctx, http.MethodPut, fmt.Sprintf("/collections/%s/points?wait=true", c.cfg.Collection), body)
	return err
}

// SearchSimilar finds nearest findings in the same repository.
func (c *Client) SearchSimilar(ctx context.Context, repository string, vector []float32, limit int) ([]ScoredFinding, error) {
	if !c.Enabled() {
		return nil, nil
	}
	if err := c.ValidateVectorLen(len(vector)); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 3
	}

	body := map[string]any{
		"vector":       vector,
		"limit":        limit,
		"with_payload": true,
		"filter": map[string]any{
			"should": []map[string]any{
				{
					"key":   "repository",
					"match": map[string]any{"value": repository},
				},
				{
					"key":   "target",
					"match": map[string]any{"value": repository},
				},
			},
		},
	}

	respBody, err := c.doRaw(ctx, http.MethodPost, fmt.Sprintf("/collections/%s/points/search", c.cfg.Collection), body)
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Result []struct {
			ID      any            `json:"id"`
			Score   float64        `json:"score"`
			Payload map[string]any `json:"payload"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}

	findings := make([]ScoredFinding, 0, len(parsed.Result))
	for _, item := range parsed.Result {
		findings = append(findings, ScoredFinding{
			ID:      fmt.Sprint(item.ID),
			Score:   item.Score,
			Payload: item.Payload,
		})
	}
	return findings, nil
}

func (c *Client) vectorSize() int {
	if c.cfg.VectorSize > 0 {
		return c.cfg.VectorSize
	}
	return 1024
}

func (c *Client) do(ctx context.Context, method, path string, body any) (int, error) {
	respBody, err := c.doRaw(ctx, method, path, body)
	if err != nil {
		return 0, err
	}
	if len(respBody) == 0 {
		return http.StatusOK, nil
	}
	return http.StatusOK, nil
}

func (c *Client) doRaw(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, stringsTrimRightSlash(c.cfg.URL)+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.cfg.APIKey != "" {
		req.Header.Set("api-key", c.cfg.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("qdrant returned status %d: %s", resp.StatusCode, string(raw))
	}
	return raw, nil
}

func stringsTrimRightSlash(value string) string {
	for len(value) > 0 && value[len(value)-1] == '/' {
		value = value[:len(value)-1]
	}
	return value
}
