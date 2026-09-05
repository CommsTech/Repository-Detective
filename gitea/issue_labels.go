package gitea

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// IssueLabelsOption is the request body for adding/replacing issue labels.
// Gitea expects {"labels": [<id or name>, ...]} — not a bare array.
type IssueLabelsOption struct {
	Labels []any `json:"labels"`
}

// AddIssueLabels attaches labels to an existing issue by ID or name.
// Returns the labels now on the issue (Gitea may return an empty slice on older versions
// even when labels were applied — callers should verify via GetIssueLabels when needed).
func (c *Client) AddIssueLabels(ctx context.Context, owner, repo string, issueNumber int, labels []any) ([]Label, error) {
	if len(labels) == 0 {
		return nil, nil
	}

	labels, err := c.filterLabelsNotOnIssue(ctx, owner, repo, issueNumber, labels)
	if err != nil {
		return nil, err
	}
	if len(labels) == 0 {
		return c.GetIssueLabels(ctx, owner, repo, issueNumber)
	}

	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/labels", c.baseURL, owner, repo, issueNumber)
	payload, err := json.Marshal(IssueLabelsOption{Labels: labels})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("gitea API returned status %d: %s", resp.StatusCode, string(body))
	}

	var attached []Label
	if len(bytes.TrimSpace(body)) == 0 || bytes.Equal(bytes.TrimSpace(body), []byte("[]")) {
		return c.GetIssueLabels(ctx, owner, repo, issueNumber)
	}

	if err := json.Unmarshal(body, &attached); err != nil {
		// Some Gitea versions return non-array payloads; fall back to GET.
		return c.GetIssueLabels(ctx, owner, repo, issueNumber)
	}
	if len(attached) == 0 {
		return c.GetIssueLabels(ctx, owner, repo, issueNumber)
	}
	return attached, nil
}

// GetIssueLabels lists labels on an issue.
func (c *Client) GetIssueLabels(ctx context.Context, owner, repo string, issueNumber int) ([]Label, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/labels", c.baseURL, owner, repo, issueNumber)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gitea API returned status %d: %s", resp.StatusCode, string(body))
	}

	var labels []Label
	if err := json.NewDecoder(resp.Body).Decode(&labels); err != nil {
		return nil, err
	}
	return labels, nil
}

func (c *Client) filterLabelsNotOnIssue(ctx context.Context, owner, repo string, issueNumber int, labels []any) ([]any, error) {
	existing, err := c.GetIssueLabels(ctx, owner, repo, issueNumber)
	if err != nil {
		// If we cannot read current labels, still attempt attach (create path).
		return labels, nil
	}
	present := make(map[string]struct{}, len(existing))
	for _, label := range existing {
		present[strings.ToLower(strings.TrimSpace(label.Name))] = struct{}{}
	}
	filtered := make([]any, 0, len(labels))
	for _, label := range labels {
		name, ok := labelName(label)
		if !ok {
			filtered = append(filtered, label)
			continue
		}
		if _, found := present[strings.ToLower(name)]; found {
			continue
		}
		filtered = append(filtered, label)
	}
	return filtered, nil
}

func labelName(label any) (string, bool) {
	switch v := label.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return "", false
		}
		return v, true
	case int:
		return strconv.Itoa(v), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case float64:
		return strconv.FormatInt(int64(v), 10), true
	default:
		return "", false
	}
}

// CreateIssueComment adds a comment to an issue.
func (c *Client) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, body string) error {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/comments", c.baseURL, owner, repo, issueNumber)
	payload, err := json.Marshal(map[string]string{"body": body})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
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

	if resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gitea API returned status %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}
