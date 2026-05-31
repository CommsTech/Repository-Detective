package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ListIssuesOptions filters repository issues.
type ListIssuesOptions struct {
	State  string
	Labels []string
	Limit  int
	Page   int
}

// ListIssues returns repository issues matching the provided filters.
func (c *Client) ListIssues(ctx context.Context, owner, repo string, opts ListIssuesOptions) ([]Issue, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.State == "" {
		opts.State = "open"
	}

	query := url.Values{}
	query.Set("state", opts.State)
	query.Set("limit", fmt.Sprintf("%d", opts.Limit))
	query.Set("page", fmt.Sprintf("%d", opts.Page))
	if len(opts.Labels) > 0 {
		query.Set("labels", strings.Join(opts.Labels, ","))
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues?%s", c.baseURL, owner, repo, query.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitea API returned status %d: %s", resp.StatusCode, string(body))
	}

	var issues []Issue
	if err := json.Unmarshal(body, &issues); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return issues, nil
}
