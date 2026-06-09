package runner

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

// Client talks to the Repository Detective runner API with HMAC auth.
type Client struct {
	CoreURL    string
	Secret     string
	HTTPClient *http.Client
}

// NewClient creates a runner API client.
func NewClient(coreURL, secret string) *Client {
	return &Client{
		CoreURL: strings.TrimSuffix(strings.TrimSpace(coreURL), "/"),
		Secret:  strings.TrimSpace(secret),
		HTTPClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

// Ping registers a worker heartbeat.
func (c *Client) Ping(ctx context.Context, runnerID, version string, capabilities []string) error {
	body, err := json.Marshal(map[string]any{
		"runner_id":    runnerID,
		"version":      version,
		"capabilities": capabilities,
	})
	if err != nil {
		return err
	}
	_, err = c.do(ctx, http.MethodPost, "/api/v1/runner/ping", body)
	return err
}

// ClaimNextJob claims the oldest queued runner job.
func (c *Client) ClaimNextJob(ctx context.Context) (string, JobSpec, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/runner/jobs/claim", []byte(`{}`))
	if err != nil {
		return "", JobSpec{}, err
	}
	var payload struct {
		Job struct {
			JobID string `json:"job_id"`
		} `json:"job"`
		Spec JobSpec `json:"spec"`
	}
	if err := json.Unmarshal(resp, &payload); err != nil {
		return "", JobSpec{}, err
	}
	return payload.Job.JobID, payload.Spec, nil
}

// GetJobSpec fetches a job spec by ID.
func (c *Client) GetJobSpec(ctx context.Context, jobID string) (JobSpec, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/runner/jobs/"+jobID+"/spec", nil)
	if err != nil {
		return JobSpec{}, err
	}
	var spec JobSpec
	if err := json.Unmarshal(resp, &spec); err != nil {
		return JobSpec{}, err
	}
	return spec, nil
}

// SubmitResult uploads a completed or failed job result.
func (c *Client) SubmitResult(ctx context.Context, jobID string, result JobResult) error {
	body, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return c.SubmitResultBody(ctx, jobID, body)
}

// SubmitResultBody uploads a pre-encoded JSON result body (single marshal preserves HMAC stability).
func (c *Client) SubmitResultBody(ctx context.Context, jobID string, body []byte) error {
	_, err := c.do(ctx, http.MethodPost, "/api/v1/runner/jobs/"+jobID+"/result", body)
	return err
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	if c == nil || c.CoreURL == "" || c.Secret == "" {
		return nil, fmt.Errorf("runner client not configured")
	}
	if body == nil {
		body = []byte{}
	}
	ts := fmt.Sprintf("%d", time.Now().UTC().Unix())
	nonce := fmt.Sprintf("rn-%d", time.Now().UnixNano())
	sig := SignRequest(c.Secret, ts, nonce, method, path, body)
	req, err := http.NewRequestWithContext(ctx, method, c.CoreURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(HeaderTimestamp, ts)
	req.Header.Set(HeaderNonce, nonce)
	req.Header.Set(HeaderSignature, sig)
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 2 * time.Minute}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("core returned %d: %s", resp.StatusCode, RedactLogLine(string(out)))
	}
	return out, nil
}
