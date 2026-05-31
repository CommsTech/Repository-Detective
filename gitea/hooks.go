package gitea

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// RepositorySummary is a lightweight repo listing entry.
type RepositorySummary struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Owner       User   `json:"owner"`
	Private     bool   `json:"private"`
	HTMLURL     string `json:"html_url"`
	Description string `json:"description"`
}

// HookConfig is used to create repository webhooks.
type HookConfig struct {
	Type   string `json:"type"`
	Config struct {
		URL         string `json:"url"`
		ContentType string `json:"content_type"`
		Secret      string `json:"secret"`
	} `json:"config"`
	Events []string `json:"events"`
	Active bool     `json:"active"`
}

// ListUserRepositories lists repositories visible to the authenticated user.
func (c *Client) ListUserRepositories(ctx context.Context, limit int) ([]RepositorySummary, error) {
	if limit <= 0 {
		limit = 50
	}
	url := fmt.Sprintf("%s/api/v1/user/repos?limit=%d", c.baseURL, limit)
	return c.listRepositories(ctx, url)
}

// ListOrgRepositories lists repositories for an organization.
func (c *Client) ListOrgRepositories(ctx context.Context, org string, limit int) ([]RepositorySummary, error) {
	if limit <= 0 {
		limit = 50
	}
	url := fmt.Sprintf("%s/api/v1/orgs/%s/repos?limit=%d", c.baseURL, org, limit)
	return c.listRepositories(ctx, url)
}

func (c *Client) listRepositories(ctx context.Context, url string) ([]RepositorySummary, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

	var repos []RepositorySummary
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}
	return repos, nil
}

// CreateRepositoryHook registers a webhook on a repository.
func (c *Client) CreateRepositoryHook(ctx context.Context, owner, repo string, hook *HookConfig) error {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/hooks", c.baseURL, owner, repo)
	payload, err := json.Marshal(hook)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gitea API returned status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ListRepositoryLabels returns labels defined on a repository.
func (c *Client) ListRepositoryLabels(ctx context.Context, owner, repo string) ([]Label, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/labels?limit=100", c.baseURL, owner, repo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

// CreateRepositoryLabel creates a label on a repository.
func (c *Client) CreateRepositoryLabel(ctx context.Context, owner, repo, name, color string) (*Label, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/labels", c.baseURL, owner, repo)
	payload, _ := json.Marshal(map[string]string{
		"name":  name,
		"color": color,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gitea API returned status %d: %s", resp.StatusCode, string(body))
	}

	var label Label
	if err := json.NewDecoder(resp.Body).Decode(&label); err != nil {
		return nil, err
	}
	return &label, nil
}

// ResolveLabelIDs returns label IDs for the given names, creating missing labels when needed.
func (c *Client) ResolveLabelIDs(ctx context.Context, owner, repo string, names []string) ([]int64, error) {
	if len(names) == 0 {
		return nil, nil
	}

	existing, err := c.ListRepositoryLabels(ctx, owner, repo)
	if err != nil {
		return nil, err
	}

	byName := make(map[string]int64, len(existing))
	for _, label := range existing {
		byName[strings.ToLower(label.Name)] = label.ID
	}

	var ids []int64
	for _, name := range names {
		if id, ok := byName[strings.ToLower(name)]; ok {
			ids = append(ids, id)
			continue
		}
		created, err := c.CreateRepositoryLabel(ctx, owner, repo, name, "5319e7")
		if err != nil {
			c.logger.Warnf("Failed to create label %s: %v", name, err)
			continue
		}
		ids = append(ids, created.ID)
	}
	return ids, nil
}
