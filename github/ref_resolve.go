package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ResolveRef picks a git ref that exists on GitHub.
// Tries the requested ref, live default branch, main/master, then branch listing.
func (c *Client) ResolveRef(ctx context.Context, owner, repo, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	candidates := make([]string, 0, 12)
	if ref != "" {
		candidates = append(candidates, ref)
	}

	var repoErr error
	info, err := c.GetRepository(ctx, owner, repo)
	if err != nil {
		repoErr = err
	} else {
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

	if branches, berr := c.listBranchNames(ctx, owner, repo); berr == nil {
		candidates = append(candidates, branches...)
	}

	seen := make(map[string]struct{}, len(candidates))
	tried := make([]string, 0, len(candidates))
	var lastProbeErr error
	definitiveProbes := 0
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		tried = append(tried, candidate)
		ok, probeErr := c.refExists(ctx, owner, repo, candidate)
		if probeErr != nil {
			lastProbeErr = probeErr
			if ctx.Err() != nil {
				return "", fmt.Errorf("unable to verify refs for %s/%s (context canceled while probing %q): %w", owner, repo, candidate, ctx.Err())
			}
			continue
		}
		definitiveProbes++
		if ok {
			return candidate, nil
		}
	}

	if definitiveProbes == 0 {
		if lastProbeErr != nil {
			return "", fmt.Errorf("unable to verify refs for %s/%s: %w", owner, repo, lastProbeErr)
		}
		if repoErr != nil {
			return "", fmt.Errorf("unable to verify refs for %s/%s: %w", owner, repo, repoErr)
		}
	}

	detail := fmt.Sprintf("tried %s", strings.Join(tried, ","))
	if repoErr != nil {
		detail += fmt.Sprintf("; get repository: %v", repoErr)
	}
	if lastProbeErr != nil {
		detail += fmt.Sprintf("; last probe: %v", lastProbeErr)
	}
	return "", fmt.Errorf("no valid ref found for %s/%s (%s)", owner, repo, detail)
}

func (c *Client) listBranchNames(ctx context.Context, owner, repo string) ([]string, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s/branches?per_page=50",
		c.baseURL, url.PathEscape(owner), url.PathEscape(repo))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
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
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("list branches status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(payload))
	for _, b := range payload {
		if name := strings.TrimSpace(b.Name); name != "" {
			out = append(out, name)
		}
	}
	return out, nil
}

func (c *Client) refExists(ctx context.Context, owner, repo, ref string) (bool, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return false, nil
	}
	endpoint := fmt.Sprintf("%s/repos/%s/%s/git/ref/heads/%s",
		c.baseURL, url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(ref))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	_, err = c.fetchContentsResponse(ctx, owner, repo, ref, "")
	if err == nil {
		return true, nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "not found") ||
		strings.Contains(err.Error(), "status 404") {
		return false, nil
	}
	return false, err
}
