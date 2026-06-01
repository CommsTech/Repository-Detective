package gitea

import (
	"context"
	"fmt"
	"strings"
)

// ResolveRef picks a git ref that exists on the remote repository.
// Tries the requested ref, the repo default branch, then common fallbacks (main, master).
func (c *Client) ResolveRef(ctx context.Context, owner, repo, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	candidates := make([]string, 0, 6)
	if ref != "" {
		candidates = append(candidates, ref)
	}

	if info, err := c.GetRepository(ctx, owner, repo); err == nil {
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
	_, err := c.fetchContentsResponse(ctx, owner, repo, ref, "")
	return err == nil
}
