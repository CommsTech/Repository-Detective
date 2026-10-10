package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// defaultHTTPPoster is the production HTTP client.
type defaultHTTPPoster struct {
	client *http.Client
}

func newDefaultHTTPPoster() HTTPPoster {
	return &defaultHTTPPoster{client: &http.Client{Timeout: 15 * time.Second}}
}

func (p *defaultHTTPPoster) Post(ctx context.Context, url string, contentType string, body []byte, headers map[string]string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", contentType)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return resp.StatusCode, fmt.Errorf("drain response body: %w", err)
	}
	return resp.StatusCode, nil
}

func postJSON(ctx context.Context, poster HTTPPoster, url string, payload any, headers map[string]string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	code, err := poster.Post(ctx, url, "application/json", body, headers)
	if err != nil {
		return err
	}
	if code < 200 || code >= 300 {
		return fmt.Errorf("HTTP %d", code)
	}
	return nil
}
