package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// RepoPermissions mirrors Gitea repository permissions for the authenticated user.
type RepoPermissions struct {
	Admin bool `json:"admin"`
	Push  bool `json:"push"`
	Pull  bool `json:"pull"`
}

// RepositoryWithPermissions is GetRepository plus permission bits when present.
type RepositoryWithPermissions struct {
	Repository
	Permissions RepoPermissions `json:"permissions"`
}

// GetRepositoryPermissions loads repo metadata and permission flags.
func (c *Client) GetRepositoryPermissions(ctx context.Context, owner, repo string) (*RepositoryWithPermissions, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s", c.baseURL, owner, repo)
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
	var out RepositoryWithPermissions
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// HookSummary is a listed webhook without secret material.
type HookSummary struct {
	ID     int64    `json:"id"`
	Type   string   `json:"type"`
	Active bool     `json:"active"`
	Events []string `json:"events"`
	URL    string   `json:"url"` // from config.url when present
}

// ListRepositoryHooks lists webhooks for a repository (secrets not returned).
func (c *Client) ListRepositoryHooks(ctx context.Context, owner, repo string) ([]HookSummary, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/hooks?limit=50", c.baseURL, owner, repo)
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
	var raw []struct {
		ID     int64    `json:"id"`
		Type   string   `json:"type"`
		Active bool     `json:"active"`
		Events []string `json:"events"`
		Config struct {
			URL string `json:"url"`
		} `json:"config"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := make([]HookSummary, 0, len(raw))
	for _, h := range raw {
		out = append(out, HookSummary{
			ID: h.ID, Type: h.Type, Active: h.Active, Events: h.Events, URL: h.Config.URL,
		})
	}
	return out, nil
}

// PermissionMatrix maps Gitea permission bits to RD-013 display states.
type PermissionMatrix struct {
	RepositoryRead      string
	IssuesWrite         string
	CommitStatusWrite   string
	PRCommentWrite      string
	BranchPRRemediation string
	Detail              string
}

func passOrNot(ok bool) string {
	if ok {
		return "PASS"
	}
	return "NOT_GRANTED"
}

// BuildPermissionMatrix converts Gitea pull/push/admin into operator-facing checks.
// Push is used as a proxy for issues/status/PR comment/branch write (Gitea token scopes
// are not always enumerated by the API).
func BuildPermissionMatrix(p RepoPermissions) PermissionMatrix {
	return PermissionMatrix{
		RepositoryRead:      passOrNot(p.Pull || p.Push || p.Admin),
		IssuesWrite:         passOrNot(p.Push || p.Admin),
		CommitStatusWrite:   passOrNot(p.Push || p.Admin),
		PRCommentWrite:      passOrNot(p.Push || p.Admin),
		BranchPRRemediation: passOrNot(p.Push || p.Admin),
		Detail:              fmt.Sprintf("gitea permissions pull=%v push=%v admin=%v", p.Pull, p.Push, p.Admin),
	}
}

// FindHookByURL reports whether any hook matches the expected callback URL.
func FindHookByURL(hooks []HookSummary, wantURL string) (HookSummary, bool) {
	want := strings.TrimRight(strings.TrimSpace(wantURL), "/")
	for _, h := range hooks {
		got := strings.TrimRight(strings.TrimSpace(h.URL), "/")
		if got != "" && strings.EqualFold(got, want) {
			return h, true
		}
	}
	return HookSummary{}, false
}
