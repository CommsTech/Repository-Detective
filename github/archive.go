package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// ErrArchiveTooLarge is returned when a repository archive exceeds the configured limit.
var ErrArchiveTooLarge = fmt.Errorf("repository archive exceeds size limit")

// DownloadRepositoryArchive streams a zipball to a temp file with a size cap.
func (c *Client) DownloadRepositoryArchive(ctx context.Context, owner, repo, ref string, maxBytes int64) (path string, cleanup func(), written int64, err error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		ref = "HEAD"
	}
	url := fmt.Sprintf("%s/repos/%s/%s/zipball/%s", c.baseURL, owner, repo, ref)
	return c.downloadArchiveURL(ctx, url, maxBytes)
}

func (c *Client) downloadArchiveURL(ctx context.Context, archiveURL string, maxBytes int64) (string, func(), int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, archiveURL, nil)
	if err != nil {
		return "", nil, 0, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return "", nil, 0, fmt.Errorf("archive not found (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", nil, 0, fmt.Errorf("github archive API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	tmp, err := os.CreateTemp("", "repository-detective-github-archive-*.zip")
	if err != nil {
		return "", nil, 0, err
	}
	cleanupFn := func() { _ = os.Remove(tmp.Name()) }

	limited := io.LimitReader(resp.Body, maxBytes+1)
	writtenBytes, err := io.Copy(tmp, limited)
	if err != nil {
		cleanupFn()
		return "", nil, 0, err
	}
	if writtenBytes > maxBytes {
		cleanupFn()
		return "", nil, 0, ErrArchiveTooLarge
	}
	if err := tmp.Close(); err != nil {
		cleanupFn()
		return "", nil, 0, err
	}
	return tmp.Name(), cleanupFn, writtenBytes, nil
}
