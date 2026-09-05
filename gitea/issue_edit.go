package gitea

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// EditIssueRequest updates issue fields.
type EditIssueRequest struct {
	State string `json:"state,omitempty"`
}

// EditIssue updates an existing issue.
func (c *Client) EditIssue(ctx context.Context, owner, repo string, issueNumber int, req *EditIssueRequest) (*Issue, error) {
	if req == nil {
		return nil, fmt.Errorf("edit issue request required")
	}
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d", c.baseURL, owner, repo, issueNumber)
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "token "+c.token)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("gitea API returned status %d: %s", resp.StatusCode, string(body))
	}
	var issue Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

// CloseIssue closes a Gitea issue.
func (c *Client) CloseIssue(ctx context.Context, owner, repo string, issueNumber int) error {
	_, err := c.EditIssue(ctx, owner, repo, issueNumber, &EditIssueRequest{State: "closed"})
	return err
}
