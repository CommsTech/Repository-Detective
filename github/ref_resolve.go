package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// ResolveRef picks a git ref that exists on GitHub.
func (c *Client) ResolveRef(ctx context.Context, owner, repo, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	candidates := make([]string, 0, 6)
	if ref != "" {
		candidates = append(candidates, ref)
	}
	info, err := c.GetRepository(ctx, owner, repo)
	if err == nil {
		if info.Empty {
			if db := strings.TrimSpace(info.DefaultBranch); db != "" {
				return db, nil
			}
			return "main", nil
		}
		if db := strings.TrimSpace(info.DefaultBranch); db != "" {
			candidates = append(candidates, db)
		}
	}
	candidates = append(candidates, "main", "master")

	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		if c.refExists(ctx, owner, repo, candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no valid ref found for %s/%s", owner, repo)
}

func (c *Client) refExists(ctx context.Context, owner, repo, ref string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return false
	}
	url := fmt.Sprintf("%s/repos/%s/%s/git/ref/heads/%s", c.baseURL, owner, repo, ref)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return true
	}
	_, err = c.fetchContentsResponse(ctx, owner, repo, ref, "")
	return err == nil
}
