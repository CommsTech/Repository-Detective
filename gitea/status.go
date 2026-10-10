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

// Commit status states understood by Repository-Detective before Gitea compatibility mapping.
const (
	CommitStatePending = "pending"
	CommitStateSuccess = "success"
	CommitStateFailure = "failure"
	CommitStateWarning = "warning"
	CommitStateError   = "error"
)

// CommitStatus is the payload for Gitea commit status creation.
type CommitStatus struct {
	State       string `json:"state"`
	TargetURL   string `json:"target_url"`
	Description string `json:"description"`
	Context     string `json:"context"`
}

// CreateCommitStatus posts a commit status to Gitea.
// POST /api/v1/repos/{owner}/{repo}/statuses/{sha}
func (c *Client) CreateCommitStatus(ctx context.Context, owner, repo, sha string, status *CommitStatus) error {
	if status == nil {
		return fmt.Errorf("commit status payload is nil")
	}

	payload := *status
	payload.State = MapGiteaCommitState(payload.State)
	payload.Description = truncateStatusDescription(payload.Description)

	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/statuses/%s", c.baseURL, owner, repo, sha)
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal commit status: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gitea API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// MapGiteaCommitState maps logical Repository-Detective states to Gitea-compatible API states.
// Gitea does not support "warning"; map it to "failure" with warning wording in description.
func MapGiteaCommitState(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case CommitStatePending, CommitStateSuccess, CommitStateFailure, CommitStateError:
		return strings.ToLower(strings.TrimSpace(state))
	case CommitStateWarning:
		return CommitStateFailure
	default:
		return CommitStateError
	}
}

// IsCommitSHA reports whether value looks like a git commit hash.
func IsCommitSHA(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 7 || len(value) > 40 {
		return false
	}
	for _, r := range value {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			continue
		}
		return false
	}
	return true
}

const maxStatusDescriptionLen = 140

func truncateStatusDescription(description string) string {
	description = strings.TrimSpace(description)
	if len(description) <= maxStatusDescriptionLen {
		return description
	}
	return description[:maxStatusDescriptionLen-3] + "..."
}
