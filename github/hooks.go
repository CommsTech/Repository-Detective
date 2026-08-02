package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"git.commsnet.org/commstech/repository-detective/forge"
)

const defaultRepoPageSize = 100

// ListAllUserRepositories lists repositories visible to the authenticated user.
func (c *Client) ListAllUserRepositories(ctx context.Context) ([]forge.RepositorySummary, error) {
	return c.listAllRepositories(ctx, c.baseURL+"/user/repos?affiliation=owner,organization_member")
}

// ListAllOrgRepositories lists repositories for a GitHub organization.
func (c *Client) ListAllOrgRepositories(ctx context.Context, org string) ([]forge.RepositorySummary, error) {
	org = strings.TrimSpace(org)
	if org == "" {
		return nil, fmt.Errorf("organization name is required")
	}
	return c.listAllRepositories(ctx, fmt.Sprintf("%s/orgs/%s/repos", c.baseURL, org))
}

func (c *Client) listAllRepositories(ctx context.Context, basePath string) ([]forge.RepositorySummary, error) {
	var all []forge.RepositorySummary
	seen := make(map[int64]struct{})
	sep := "?"
	if strings.Contains(basePath, "?") {
		sep = "&"
	}
	for page := 1; ; page++ {
		url := fmt.Sprintf("%s%sper_page=%d&page=%d", basePath, sep, defaultRepoPageSize, page)
		pageRepos, err := c.listRepositories(ctx, url)
		if err != nil {
			return nil, err
		}
		if len(pageRepos) == 0 {
			break
		}
		for _, repo := range pageRepos {
			if _, ok := seen[repo.ID]; ok {
				continue
			}
			seen[repo.ID] = struct{}{}
			all = append(all, repo)
		}
		if len(pageRepos) < defaultRepoPageSize {
			break
		}
	}
	return all, nil
}

func (c *Client) listRepositories(ctx context.Context, url string) ([]forge.RepositorySummary, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github API returned status %d: %s", resp.StatusCode, string(body))
	}

	var repos []forge.RepositorySummary
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}
	return repos, nil
}
