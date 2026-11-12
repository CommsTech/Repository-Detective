package gitea

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"git.commsnet.org/commstech/repository-detective/internal/httpbody"
)

// IssueLabelsOption is the request body for adding/replacing issue labels.
// Gitea expects {"labels": [<id or name>, ...]} — not a bare array.
type IssueLabelsOption struct {
	Labels []any `json:"labels"`
}

// AddIssueLabels attaches labels to an existing issue by ID or name and then
// verifies the request actually landed.
//
// Some Gitea versions/authorization setups accept unknown label names or drop
// labels the token cannot resolve while returning 200 — a silent no-op. This
// client therefore treats the post-attach state as authoritative:
//
//  1. Read the issue's current labels and skip anything already present.
//  2. Attach the remainder.
//  3. Read the resulting label set back and check every requested label —
//     by numeric ID when the request was an ID, by case-insensitive name
//     otherwise — is now present.
//  4. When name-typed labels are missing, ensure they exist on the repository
//     (creating them via ResolveLabelIDs) and retry the attach once.
//  5. Any label still missing afterwards returns an error naming it; the
//     caller's forge write is never reported successful without proof.
func (c *Client) AddIssueLabels(ctx context.Context, owner, repo string, issueNumber int, labels []any) ([]Label, error) {
	if len(labels) == 0 {
		return nil, nil
	}

	labels, err := c.filterLabelsNotOnIssue(ctx, owner, repo, issueNumber, labels)
	if err != nil {
		return nil, err
	}
	if len(labels) == 0 {
		return c.GetIssueLabels(ctx, owner, repo, issueNumber)
	}

	attached, err := c.attachIssueLabels(ctx, owner, repo, issueNumber, labels)
	if err != nil {
		return nil, err
	}

	missing, ok := verifyRequestedLabels(labels, attached)
	if ok {
		return attached, nil
	}

	// The attach was silently dropped (unknown names, unreadable token, older
	// Gitea). Ensure name-typed labels exist on the repository, then retry once
	// with the resolved IDs before failing.
	names := make([]string, 0, len(missing))
	var numeric []any
	for _, want := range missing {
		name, isName := labelName(want)
		if _, isID := asLabelID(want); !isID && isName {
			names = append(names, name)
		} else if isName {
			numeric = append(numeric, want)
		}
	}
	if len(names) > 0 {
		ids, err := c.ResolveLabelIDs(ctx, owner, repo, names)
		if err != nil {
			return attached, fmt.Errorf("labels not applied on issue #%d and ensure failed: %w (missing: %s)", issueNumber, err, missingLabelText(missing))
		}
		for _, id := range ids {
			numeric = append(numeric, id)
		}
	}
	if len(numeric) > 0 {
		if _, err := c.attachIssueLabels(ctx, owner, repo, issueNumber, numeric); err != nil {
			return attached, fmt.Errorf("labels missing on issue #%d and attach retry failed: %w (missing: %s)", issueNumber, err, missingLabelText(missing))
		}
	}
	final, err := c.GetIssueLabels(ctx, owner, repo, issueNumber)
	if err != nil {
		return attached, fmt.Errorf("labels not applied on issue #%d and verification failed: %w (missing: %s)", issueNumber, err, missingLabelText(missing))
	}
	if _, ok := verifyRequestedLabels(labels, final); !ok {
		return final, fmt.Errorf("gitea accepted the request but did not apply labels on issue #%d (missing: %s)", issueNumber, missingLabelText(missing))
	}
	return final, nil
}

// attachIssueLabels performs the raw POST attach and resolves the resulting
// label set, falling back to GET when the response body is empty or not a
// label array (some Gitea versions behave that way).
func (c *Client) attachIssueLabels(ctx context.Context, owner, repo string, issueNumber int, labels []any) ([]Label, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/labels", c.baseURL, owner, repo, issueNumber)
	payload, err := json.Marshal(IssueLabelsOption{Labels: labels})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpbody.Close(resp.Body)

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("gitea API returned status %d: %s", resp.StatusCode, string(body))
	}

	var attached []Label
	if len(bytes.TrimSpace(body)) == 0 || bytes.Equal(bytes.TrimSpace(body), []byte("[]")) {
		return c.GetIssueLabels(ctx, owner, repo, issueNumber)
	}

	if err := json.Unmarshal(body, &attached); err != nil {
		// Some Gitea versions return non-array payloads; fall back to GET.
		return c.GetIssueLabels(ctx, owner, repo, issueNumber)
	}
	if len(attached) == 0 {
		return c.GetIssueLabels(ctx, owner, repo, issueNumber)
	}
	return attached, nil
}

// GetIssueLabels lists labels on an issue.
func (c *Client) GetIssueLabels(ctx context.Context, owner, repo string, issueNumber int) ([]Label, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/labels", c.baseURL, owner, repo, issueNumber)

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
	defer httpbody.Close(resp.Body)

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

func (c *Client) filterLabelsNotOnIssue(ctx context.Context, owner, repo string, issueNumber int, labels []any) ([]any, error) {
	existing, err := c.GetIssueLabels(ctx, owner, repo, issueNumber)
	if err != nil {
		// If we cannot read current labels, still attempt attach (create path).
		return labels, nil
	}
	present := make(map[string]struct{}, len(existing))
	for _, label := range existing {
		present[strings.ToLower(strings.TrimSpace(label.Name))] = struct{}{}
	}
	filtered := make([]any, 0, len(labels))
	for _, label := range labels {
		name, ok := labelName(label)
		if !ok {
			filtered = append(filtered, label)
			continue
		}
		if _, found := present[strings.ToLower(name)]; found {
			continue
		}
		filtered = append(filtered, label)
	}
	return filtered, nil
}

// verifyRequestedLabels checks that every requested label (by numeric ID when
// the request was numeric, else by case-insensitive name) is present in got.
// Elements of got are Label values (the wire type); it returns the missing
// request values and whether everything was found.
func verifyRequestedLabels(requested []any, got []Label) ([]any, bool) {
	present := make(map[labelKey]struct{}, len(got))
	for _, g := range got {
		present[labelKey{id: g.ID, name: strings.ToLower(strings.TrimSpace(g.Name))}] = struct{}{}
	}
	var missing []any
	for _, r := range requested {
		id, isID := asLabelID(r)
		name, _ := labelName(r)
		found := false
		for k := range present {
			if isID && k.id == id {
				found = true
				break
			}
			if !isID && name != "" && k.name == strings.ToLower(name) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, r)
		}
	}
	return missing, len(missing) == 0
}

type labelKey struct {
	id   int64
	name string
}

// asLabelID converts a numeric label request to its ID. Non-numeric requests
// return (0, false).
func asLabelID(label any) (int64, bool) {
	switch v := label.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

// missingLabelText renders missing request values for error messages.
func missingLabelText(missing []any) string {
	parts := make([]string, 0, len(missing))
	for _, m := range missing {
		if name, ok := labelName(m); ok {
			if _, isID := asLabelID(m); isID {
				parts = append(parts, fmt.Sprintf("#%s", name))
				continue
			}
			parts = append(parts, name)
		} else {
			parts = append(parts, fmt.Sprintf("%v", m))
		}
	}
	return strings.Join(parts, ", ")
}

// labelName reports the canonical string identity of a label request value.
func labelName(label any) (string, bool) {
	switch v := label.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return "", false
		}
		return v, true
	case int:
		return strconv.Itoa(v), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case float64:
		return strconv.FormatInt(int64(v), 10), true
	default:
		return "", false
	}
}

// CreateIssueComment adds a comment to an issue.
func (c *Client) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, body string) error {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/comments", c.baseURL, owner, repo, issueNumber)
	payload, err := json.Marshal(map[string]string{"body": body})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer httpbody.Close(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gitea API returned status %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}
