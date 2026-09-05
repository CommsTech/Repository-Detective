package gitea

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// errNotDirectory is returned when the Gitea contents API returns a file for a path
// that was listed as a directory (submodules, gitlinks, or inconsistent tree entries).
var errNotDirectory = errors.New("path is not a directory")

// Client handles communication with Gitea API
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	logger     *logrus.Logger
}

// Token returns the configured API token so callers that must authenticate
// outside the HTTP client (such as git clone over HTTPS) can reuse it.
func (c *Client) Token() string {
	if c == nil {
		return ""
	}
	return c.token
}

// RepositoryContent represents a file or directory in a repository
type RepositoryContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	SHA         string `json:"sha"`
	Size        int64  `json:"size"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	GitURL      string `json:"git_url"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"` // "file", "dir", "symlink"
	Content     string `json:"content,omitempty"`
	Encoding    string `json:"encoding,omitempty"`
	Links       struct {
		Self string `json:"self"`
		Git  string `json:"git"`
		HTML string `json:"html"`
	} `json:"_links"`
}

// Issue represents a Gitea issue
type Issue struct {
	ID          int64        `json:"id"`
	Number      int          `json:"number"`
	User        User         `json:"user"`
	Title       string       `json:"title"`
	Body        string       `json:"body"`
	State       string       `json:"state"`
	Comments    int          `json:"comments"`
	HTMLURL     string       `json:"html_url"`
	Milestone   *Milestone   `json:"milestone,omitempty"`
	Labels      []Label      `json:"labels"`
	Assignee    *User        `json:"assignee,omitempty"`
	Assignees   []User       `json:"assignees,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	ClosedAt    *time.Time   `json:"closed_at,omitempty"`
	DueDate     *time.Time   `json:"due_date,omitempty"`
	PullRequest *PullRequest `json:"pull_request,omitempty"`
}

// CreateIssueRequest represents a request to create an issue
type CreateIssueRequest struct {
	Title     string  `json:"title"`
	Body      string  `json:"body"`
	Assignee  string  `json:"assignee,omitempty"`
	Milestone int64   `json:"milestone,omitempty"`
	Labels    []int64 `json:"labels,omitempty"`
	Closed    bool    `json:"closed,omitempty"`
	DueDate   string  `json:"due_date,omitempty"`
}

// User represents a Gitea user
type User struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	Username  string `json:"username"`
}

// Label represents a Gitea label
type Label struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// Milestone represents a Gitea milestone
type Milestone struct {
	ID           int64      `json:"id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	State        string     `json:"state"`
	DueDate      *time.Time `json:"due_date,omitempty"`
	ClosedAt     *time.Time `json:"closed_at,omitempty"`
	OpenIssues   int        `json:"open_issues"`
	ClosedIssues int        `json:"closed_issues"`
}

// PullRequest represents a Gitea pull request
type PullRequest struct {
	ID         int64      `json:"id"`
	Number     int        `json:"number"`
	State      string     `json:"state"`
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	User       User       `json:"user"`
	HTMLURL    string     `json:"html_url"`
	DiffURL    string     `json:"diff_url"`
	PatchURL   string     `json:"patch_url"`
	Mergeable  bool       `json:"mergeable"`
	Merged     bool       `json:"merged"`
	MergedAt   string     `json:"merged_at,omitempty"`
	MergedBy   *User      `json:"merged_by,omitempty"`
	BaseBranch string     `json:"base_branch"`
	HeadBranch string     `json:"head_branch"`
	Head       PullRequestGitRef `json:"head"`
	BaseRepo   Repository `json:"base_repo"`
	HeadRepo   Repository `json:"head_repo"`
}

// PullRequestGitRef identifies the head commit/branch of a pull request.
type PullRequestGitRef struct {
	SHA string `json:"sha"`
	Ref string `json:"ref"`
}

// Repository represents a Gitea repository
type Repository struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Owner         User   `json:"owner"`
	Private       bool   `json:"private"`
	HTMLURL       string `json:"html_url"`
	CloneURL      string `json:"clone_url"`
	GitURL        string `json:"git_url"`
	SSHURL        string `json:"ssh_url"`
	Description   string `json:"description"`
	Language      string `json:"language"`
	DefaultBranch string `json:"default_branch"`
	Size          int64  `json:"size"`
	Empty         bool   `json:"empty"`
	Fork          bool   `json:"fork"`
	Archived      bool   `json:"archived"`
}

// NewClient creates a new Gitea client
func NewClient(baseURL, token string, logger *logrus.Logger) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: logger,
	}
}

// GetRepositoryContent fetches the content of a file or directory entry.
// Gitea returns a single object for files and a JSON array for directories.
func (c *Client) GetRepositoryContent(ctx context.Context, owner, repo, ref, path string) (*RepositoryContent, error) {
	body, err := c.fetchContentsResponse(ctx, owner, repo, ref, path)
	if err != nil {
		return nil, err
	}

	items, err := decodeRepositoryContents(body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("content not found: %s", path)
	}
	if len(items) > 1 {
		return nil, fmt.Errorf("path is a directory: %s", path)
	}
	return &items[0], nil
}

// GetFileContent fetches the content of a specific file
func (c *Client) GetFileContent(ctx context.Context, owner, repo, ref, filePath string) (string, error) {
	// Issue #10: Shorter timeout for file downloads (30s)
	fileCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	content, err := c.GetRepositoryContent(fileCtx, owner, repo, ref, filePath)
	if err != nil {
		return "", err
	}

	if content.Type != "file" {
		return "", fmt.Errorf("path is not a file: %s", filePath)
	}

	// Decode base64 content when returned by the Gitea API
	if content.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(content.Content))
		if err != nil {
			return "", fmt.Errorf("failed to decode base64 content for %s: %w", filePath, err)
		}
		return string(decoded), nil
	}

	return content.Content, nil
}

// ListRepositoryContents lists the contents of a directory
func (c *Client) ListRepositoryContents(ctx context.Context, owner, repo, ref, dirPath string) ([]RepositoryContent, error) {
	body, err := c.fetchContentsResponse(ctx, owner, repo, ref, dirPath)
	if err != nil {
		return nil, err
	}

	contents, err := decodeRepositoryContents(body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(contents) == 1 && contents[0].Type == "file" {
		// Some repos expose only a single file at the repository root.
		if strings.TrimSpace(dirPath) == "" {
			return contents, nil
		}
		return nil, fmt.Errorf("%w: %s", errNotDirectory, dirPath)
	}

	return contents, nil
}

func (c *Client) fetchContentsResponse(ctx context.Context, owner, repo, ref, path string) ([]byte, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/contents/%s", c.baseURL, owner, repo, path)
	if ref != "" {
		url += "?ref=" + ref
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("content not found: %s", path)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitea API returned status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// decodeRepositoryContents handles Gitea returning either a JSON array (directories)
// or a single object (files).
func decodeRepositoryContents(body []byte) ([]RepositoryContent, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}

	if trimmed[0] == '[' {
		var contents []RepositoryContent
		if err := json.Unmarshal(body, &contents); err != nil {
			return nil, err
		}
		return contents, nil
	}

	var content RepositoryContent
	if err := json.Unmarshal(body, &content); err != nil {
		return nil, err
	}
	return []RepositoryContent{content}, nil
}

// CreateIssue creates a new issue in a repository
func (c *Client) CreateIssue(ctx context.Context, owner, repo string, issueReq *CreateIssueRequest) (*Issue, error) {
	// Issue #10: Medium timeout for issue creation (45s)
	issueCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues", c.baseURL, owner, repo)

	jsonData, err := json.Marshal(issueReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal issue request: %w", err)
	}

	req, err := http.NewRequestWithContext(issueCtx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gitea API returned status %d: %s", resp.StatusCode, string(body))
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}

// GetRepository gets repository information
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s", c.baseURL, owner, repo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gitea API returned status %d: %s", resp.StatusCode, string(body))
	}

	var repository Repository
	if err := json.NewDecoder(resp.Body).Decode(&repository); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &repository, nil
}

// GetPullRequest gets pull request information
func (c *Client) GetPullRequest(ctx context.Context, owner, repo string, prNumber int) (*PullRequest, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls/%d", c.baseURL, owner, repo, prNumber)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gitea API returned status %d: %s", resp.StatusCode, string(body))
	}

	var pullRequest PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&pullRequest); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &pullRequest, nil
}

// GetChangedFiles gets the list of changed files in a pull request
func (c *Client) GetChangedFiles(ctx context.Context, owner, repo string, prNumber int) ([]string, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls/%d/files", c.baseURL, owner, repo, prNumber)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gitea API returned status %d: %s", resp.StatusCode, string(body))
	}

	var files []struct {
		SHA         string `json:"sha"`
		Filename    string `json:"filename"`
		Status      string `json:"status"`
		Additions   int    `json:"additions"`
		Deletions   int    `json:"deletions"`
		Changes     int    `json:"changes"`
		BlobURL     string `json:"blob_url"`
		RawURL      string `json:"raw_url"`
		ContentsURL string `json:"contents_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var changedFiles []string
	for _, file := range files {
		changedFiles = append(changedFiles, file.Filename)
	}

	return changedFiles, nil
}

// repositoryPathSkipped reports paths under common vendor or VCS directories.
func repositoryPathSkipped(path string) bool {
	for _, segment := range strings.Split(strings.Trim(path, "/"), "/") {
		switch strings.ToLower(segment) {
		case "node_modules", "vendor", ".git", ".venv", "venv", "__pycache__", "build", "dist", "target":
			return true
		}
	}
	return false
}

// ListAllFiles recursively lists all files in a repository
func (c *Client) ListAllFiles(ctx context.Context, owner, repo, ref, dirPath string) ([]RepositoryContent, error) {
	var allFiles []RepositoryContent

	// Handle root directory
	if dirPath == "" {
		// Get the repository to find the default branch
		repoInfo, err := c.GetRepository(ctx, owner, repo)
		if err != nil {
			return nil, fmt.Errorf("failed to get repository info: %w", err)
		}
		dirPath = ""
		if ref == "" {
			ref = repoInfo.DefaultBranch
		}
	}

	// Fetch directory contents
	contents, err := c.ListRepositoryContents(ctx, owner, repo, ref, dirPath)
	if err != nil {
		return nil, err
	}

	for _, item := range contents {
		if item.Type == "dir" {
			if repositoryPathSkipped(item.Path) {
				continue
			}
			// Recursively fetch subdirectory
			subFiles, err := c.ListAllFiles(ctx, owner, repo, ref, item.Path)
			if err != nil {
				if errors.Is(err, errNotDirectory) {
					// Tree listed type=dir but contents API returned a file (submodule/gitlink).
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

// TestConnection tests the connection to Gitea
func (c *Client) TestConnection(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/v1/version", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gitea API returned status %d", resp.StatusCode)
	}

	return nil
}
