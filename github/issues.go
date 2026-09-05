package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Issue is a GitHub issue (not a pull request).
type Issue struct {
	Number      int    `json:"number"`
	HTMLURL     string `json:"html_url"`
	Body        string `json:"body"`
	PullRequest *struct {
		URL string `json:"url"`
	} `json:"pull_request,omitempty"`
}

// ListIssuesOptions filters repository issues.
type ListIssuesOptions struct {
	State  string
	Labels []string
	Limit  int
	Page   int
}

// CreateIssueRequest creates a GitHub issue.
type CreateIssueRequest struct {
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Labels []string `json:"labels,omitempty"`
}

// ListIssues returns open issues matching labels (excludes pull requests).
func (c *Client) ListIssues(ctx context.Context, owner, repo string, opts ListIssuesOptions) ([]Issue, error) {
	if opts.Limit <= 0 {
		opts.Limit = 100
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.State == "" {
		opts.State = "open"
	}

	query := url.Values{}
	query.Set("state", opts.State)
	query.Set("per_page", fmt.Sprintf("%d", opts.Limit))
	query.Set("page", fmt.Sprintf("%d", opts.Page))
	if len(opts.Labels) > 0 {
		query.Set("labels", strings.Join(opts.Labels, ","))
	}

	endpoint := fmt.Sprintf("%s/repos/%s/%s/issues?%s", c.baseURL, owner, repo, query.Encode())
	body, err := c.doJSON(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var raw []Issue
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode issues: %w", err)
	}
	out := make([]Issue, 0, len(raw))
	for _, issue := range raw {
		if issue.PullRequest != nil {
			continue
		}
		out = append(out, issue)
	}
	return out, nil
}

// CreateIssue opens a new GitHub issue.
func (c *Client) CreateIssue(ctx context.Context, owner, repo string, req *CreateIssueRequest) (*Issue, error) {
	if req == nil {
		return nil, fmt.Errorf("create issue request is nil")
	}
	endpoint := fmt.Sprintf("%s/repos/%s/%s/issues", c.baseURL, owner, repo)
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	body, err := c.doJSON(ctx, http.MethodPost, endpoint, payload)
	if err != nil {
		return nil, err
	}
	var issue Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("decode created issue: %w", err)
	}
	return &issue, nil
}

// CreateIssueComment posts a comment on an issue.
func (c *Client) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, body string) error {
	endpoint := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", c.baseURL, owner, repo, issueNumber)
	payload, err := json.Marshal(map[string]string{"body": body})
	if err != nil {
		return err
	}
	_, err = c.doJSON(ctx, http.MethodPost, endpoint, payload)
	return err
}

// AddIssueLabels attaches labels by name to an issue.
func (c *Client) AddIssueLabels(ctx context.Context, owner, repo string, issueNumber int, labels []string) error {
	if len(labels) == 0 {
		return nil
	}
	endpoint := fmt.Sprintf("%s/repos/%s/%s/issues/%d/labels", c.baseURL, owner, repo, issueNumber)
	payload, err := json.Marshal(labels)
	if err != nil {
		return err
	}
	_, err = c.doJSON(ctx, http.MethodPost, endpoint, payload)
	return err
}

// EnsureRepositoryLabels creates missing labels so CreateIssue can reference them.
func (c *Client) EnsureRepositoryLabels(ctx context.Context, owner, repo string, names []string) error {
	if len(names) == 0 {
		return nil
	}
	existing, err := c.ListRepositoryLabels(ctx, owner, repo)
	if err != nil {
		return err
	}
	byName := make(map[string]struct{}, len(existing))
	for _, label := range existing {
		byName[strings.ToLower(label.Name)] = struct{}{}
	}
	for _, name := range names {
		if name == "" {
			continue
		}
		if _, ok := byName[strings.ToLower(name)]; ok {
			continue
		}
		if err := c.CreateRepositoryLabel(ctx, owner, repo, name); err != nil {
			c.logger.Warnf("Failed to create GitHub label %s: %v", name, err)
		}
	}
	return nil
}

type repositoryLabel struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// ListRepositoryLabels lists labels defined on a repository.
func (c *Client) ListRepositoryLabels(ctx context.Context, owner, repo string) ([]repositoryLabel, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s/labels?per_page=100", c.baseURL, owner, repo)
	body, err := c.doJSON(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	var labels []repositoryLabel
	if err := json.Unmarshal(body, &labels); err != nil {
		return nil, err
	}
	return labels, nil
}

// CreateRepositoryLabel adds a label to a repository.
func (c *Client) CreateRepositoryLabel(ctx context.Context, owner, repo, name string) error {
	endpoint := fmt.Sprintf("%s/repos/%s/%s/labels", c.baseURL, owner, repo)
	payload, err := json.Marshal(map[string]string{
		"name":  name,
		"color": defaultLabelColor(name),
	})
	if err != nil {
		return err
	}
	_, err = c.doJSON(ctx, http.MethodPost, endpoint, payload)
	return err
}

func defaultLabelColor(name string) string {
	switch {
	case strings.Contains(strings.ToLower(name), "critical"), strings.Contains(strings.ToLower(name), "high"):
		return "d73a4a"
	case strings.Contains(strings.ToLower(name), "security"):
		return "ee0701"
	case strings.Contains(strings.ToLower(name), "medium"):
		return "fbca04"
	default:
		return "0e8a16"
	}
}

func (c *Client) doJSON(ctx context.Context, method, endpoint string, payload []byte) ([]byte, error) {
	var bodyReader io.Reader
	if len(payload) > 0 {
		bodyReader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Accept", "application/vnd.github+json")
	if len(payload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github API %s %s returned %d: %s", method, endpoint, resp.StatusCode, string(body))
	}
	return body, nil
}
