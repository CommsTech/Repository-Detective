package gitea

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// ErrArchiveTooLarge is returned when a repository archive exceeds the configured limit.
var ErrArchiveTooLarge = fmt.Errorf("repository archive exceeds size limit")

// DownloadRepositoryArchive streams a repo zipball to a temp file with a size cap.
// Tries the GitHub-compatible zipball endpoint first, then the legacy archive endpoint.
func (c *Client) DownloadRepositoryArchive(ctx context.Context, owner, repo, ref string, maxBytes int64) (path string, cleanup func(), written int64, err error) {
	if maxBytes <= 0 {
		maxBytes = 500 * 1024 * 1024
	}

	refPath := encodeArchiveRef(ref)
	candidates := []string{
		fmt.Sprintf("%s/api/v1/repos/%s/%s/zipball/%s", c.baseURL, owner, repo, refPath),
		fmt.Sprintf("%s/api/v1/repos/%s/%s/archive/%s.zip", c.baseURL, owner, repo, refPath),
	}

	var lastErr error
	for _, archiveURL := range candidates {
		path, cleanup, written, err = c.downloadArchiveURL(ctx, archiveURL, maxBytes)
		if err == nil {
			return path, cleanup, written, nil
		}
		lastErr = err
		if !isNotFoundArchiveError(err) {
			return "", nil, 0, err
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("archive download failed")
	}
	return "", nil, 0, lastErr
}

func encodeArchiveRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ref
	}
	parts := strings.Split(ref, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func (c *Client) downloadArchiveURL(ctx context.Context, archiveURL string, maxBytes int64) (string, func(), int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, archiveURL, nil)
	if err != nil {
		return "", nil, 0, err
	}
	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", nil, 0, fmt.Errorf("archive not found (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", nil, 0, fmt.Errorf("gitea archive API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	tmp, err := os.CreateTemp("", "rd-archive-*.zip")
	if err != nil {
		return "", nil, 0, err
	}
	cleanup := func() {
		_ = os.Remove(tmp.Name())
	}

	limited := io.LimitReader(resp.Body, maxBytes+1)
	written, err := io.Copy(tmp, limited)
	if err != nil {
		cleanup()
		return "", nil, 0, err
	}
	if written > maxBytes {
		cleanup()
		return "", nil, 0, fmt.Errorf("%w (%d bytes > %d bytes)", ErrArchiveTooLarge, written, maxBytes)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return "", nil, 0, err
	}

	return tmp.Name(), cleanup, written, nil
}

func isNotFoundArchiveError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "status 404") || strings.Contains(msg, "archive not found")
}
