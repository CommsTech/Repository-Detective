package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var commitSHARe = regexp.MustCompile(`(?i)^[0-9a-f]{40}$`)

// ResolveTipCommit returns the tip commit SHA for a branch (or tag) ref.
func (c *Client) ResolveTipCommit(ctx context.Context, owner, repo, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", nil
	}
	if commitSHARe.MatchString(ref) {
		return strings.ToLower(ref), nil
	}

	endpoint := fmt.Sprintf("%s/repos/%s/%s/branches/%s",
		strings.TrimRight(c.baseURL, "/"), url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(ref))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tip commit status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	sha := strings.ToLower(strings.TrimSpace(payload.Commit.SHA))
	if !commitSHARe.MatchString(sha) {
		return "", fmt.Errorf("tip commit: invalid sha %q", payload.Commit.SHA)
	}
	return sha, nil
}
