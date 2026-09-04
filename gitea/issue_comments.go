package gitea

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// IssueComment is a Gitea issue/PR comment.
type IssueComment struct {
	ID      int64  `json:"id"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
	User    User   `json:"user"`
}

// ListIssueComments returns comments on an issue or pull request (PRs share the issue comment API).
func (c *Client) ListIssueComments(ctx context.Context, owner, repo string, issueNumber int) ([]IssueComment, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/comments?limit=100", c.baseURL, owner, repo, issueNumber)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gitea API returned status %d: %s", resp.StatusCode, string(raw))
	}
	var comments []IssueComment
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		return nil, err
	}
	return comments, nil
}

// EditIssueComment updates an existing issue/PR comment by ID.
func (c *Client) EditIssueComment(ctx context.Context, owner, repo string, commentID int64, body string) error {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/comments/%d", c.baseURL, owner, repo, commentID)
	payload, err := json.Marshal(map[string]string{"body": body})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gitea API returned status %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

// DeleteIssueComment deletes an issue/PR comment by ID.
func (c *Client) DeleteIssueComment(ctx context.Context, owner, repo string, commentID int64) error {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/comments/%d", c.baseURL, owner, repo, commentID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gitea API returned status %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

// CreateIssueCommentReturningID creates a comment and returns its ID when the API includes it.
func (c *Client) CreateIssueCommentReturningID(ctx context.Context, owner, repo string, issueNumber int, body string) (int64, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/comments", c.baseURL, owner, repo, issueNumber)
	payload, err := json.Marshal(map[string]string{"body": body})
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("gitea API returned status %d: %s", resp.StatusCode, string(raw))
	}
	var created IssueComment
	if err := json.Unmarshal(raw, &created); err != nil {
		return 0, nil // created successfully; ID optional
	}
	if created.ID != 0 {
		return created.ID, nil
	}
	// Some Gitea versions nest id differently — try map parse.
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err == nil {
		if id, ok := asInt64(generic["id"]); ok {
			return id, nil
		}
	}
	_ = strconv.Itoa(0)
	return 0, nil
}

func asInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	default:
		return 0, false
	}
}
