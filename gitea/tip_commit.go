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

// ResolveTipCommit returns the tip commit SHA for a branch (or tag) ref.
func (c *Client) ResolveTipCommit(ctx context.Context, owner, repo, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", nil
	}
	if IsCommitSHA(ref) {
		return strings.ToLower(ref), nil
	}

	// Prefer branch endpoint — returns commit.id.
	endpoint := fmt.Sprintf("%s/api/v1/repos/%s/%s/branches/%s",
		c.baseURL, url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(ref))
	sha, err := c.fetchTipSHA(ctx, endpoint, true)
	if err == nil && sha != "" {
		return sha, nil
	}

	// Fallback: git refs API.
	endpoint = fmt.Sprintf("%s/api/v1/repos/%s/%s/git/refs/heads/%s",
		c.baseURL, url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(ref))
	sha, err2 := c.fetchTipSHA(ctx, endpoint, false)
	if err2 == nil && sha != "" {
		return sha, nil
	}
	if err != nil {
		return "", err
	}
	return "", err2
}

func (c *Client) fetchTipSHA(ctx context.Context, endpoint string, branchAPI bool) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tip commit status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if branchAPI {
		var payload struct {
			Commit struct {
				ID string `json:"id"`
			} `json:"commit"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return "", err
		}
		return normalizeCommitSHA(payload.Commit.ID), nil
	}
	// git/refs may return object or array
	var single struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := json.Unmarshal(body, &single); err == nil {
		if sha := normalizeCommitSHA(single.Object.SHA); sha != "" {
			return sha, nil
		}
	}
	var multi []struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := json.Unmarshal(body, &multi); err == nil && len(multi) > 0 {
		return normalizeCommitSHA(multi[0].Object.SHA), nil
	}
	return "", fmt.Errorf("tip commit: unable to parse sha from response")
}

func normalizeCommitSHA(sha string) string {
	sha = strings.ToLower(strings.TrimSpace(sha))
	if !IsCommitSHA(sha) {
		return ""
	}
	return sha
}
