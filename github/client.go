package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/forge"
	"github.com/sirupsen/logrus"
)

var errNotDirectory = errors.New("path is not a directory")

// Client talks to the GitHub REST API for repository content and listing.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewClient creates a GitHub API client. baseURL defaults to https://api.github.com.
func NewClient(baseURL, token string, logger *logrus.Logger) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	if logger == nil {
		logger = logrus.New()
	}
	return &Client{
		baseURL:    baseURL,
		token:      strings.TrimSpace(token),
		httpClient: &http.Client{Timeout: 120 * time.Second},
		logger:     logger,
	}
}

var _ forge.RepoClient = (*Client)(nil)

func (c *Client) authHeader() string {
	return "Bearer " + c.token
}

func (c *Client) TestConnection(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/user", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github API returned status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*forge.Repository, error) {
	url := fmt.Sprintf("%s/repos/%s/%s", c.baseURL, owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github API returned status %d: %s", resp.StatusCode, string(body))
	}
	var gh struct {
		DefaultBranch string `json:"default_branch"`
		CloneURL      string `json:"clone_url"`
		Size          int    `json:"size"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gh); err != nil {
		return nil, err
	}
	return &forge.Repository{
		DefaultBranch: gh.DefaultBranch,
		CloneURL:      gh.CloneURL,
		Empty:         gh.Size == 0,
	}, nil
}

func (c *Client) GetFileContent(ctx context.Context, owner, repo, ref, filePath string) (string, error) {
	content, err := c.getRepositoryContent(ctx, owner, repo, ref, filePath)
	if err != nil {
		return "", err
	}
	if content.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(content.Content))
		if err != nil {
			return "", fmt.Errorf("decode base64 for %s: %w", filePath, err)
		}
		return string(decoded), nil
	}
	return content.Content, nil
}

func (c *Client) getRepositoryContent(ctx context.Context, owner, repo, ref, path string) (*forge.RepositoryContent, error) {
	body, err := c.fetchContentsResponse(ctx, owner, repo, ref, path)
	if err != nil {
		return nil, err
	}
	items, err := forge.DecodeRepositoryContents(body)
	if err != nil {
		return nil, err
	}
	if len(items) != 1 {
		return nil, fmt.Errorf("expected single file at %s", path)
	}
	return &items[0], nil
}

func (c *Client) ListRepositoryContents(ctx context.Context, owner, repo, ref, dirPath string) ([]forge.RepositoryContent, error) {
	body, err := c.fetchContentsResponse(ctx, owner, repo, ref, dirPath)
	if err != nil {
		return nil, err
	}
	contents, err := forge.DecodeRepositoryContents(body)
	if err != nil {
		return nil, err
	}
	if len(contents) == 1 && contents[0].Type == "file" {
		if strings.TrimSpace(dirPath) == "" {
			return contents, nil
		}
		return nil, fmt.Errorf("%w: %s", errNotDirectory, dirPath)
	}
	return contents, nil
}

func (c *Client) fetchContentsResponse(ctx context.Context, owner, repo, ref, path string) ([]byte, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", c.baseURL, owner, repo, path)
	if ref != "" {
		url += "?ref=" + ref
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("content not found: %s", path)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned status %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (c *Client) ListAllFiles(ctx context.Context, owner, repo, ref, dirPath string) ([]forge.RepositoryContent, error) {
	var allFiles []forge.RepositoryContent
	if ref == "" {
		info, err := c.GetRepository(ctx, owner, repo)
		if err != nil {
			return nil, err
		}
		ref = info.DefaultBranch
	}
	contents, err := c.ListRepositoryContents(ctx, owner, repo, ref, dirPath)
	if err != nil {
		return nil, err
	}
	for _, item := range contents {
		if item.Type == "dir" {
			if repositoryPathSkipped(item.Path) {
				continue
			}
			subFiles, err := c.ListAllFiles(ctx, owner, repo, ref, item.Path)
			if err != nil {
				if errors.Is(err, errNotDirectory) {
					allFiles = append(allFiles, item)
					continue
				}
				c.logger.Debugf("Skipping path %s during file listing: %v", item.Path, err)
				continue
			}
			allFiles = append(allFiles, subFiles...)
		} else {
			allFiles = append(allFiles, item)
		}
	}
	return allFiles, nil
}
