package forge

// RepositoryContent is a file or directory entry from a forge contents API.
type RepositoryContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	SHA         string `json:"sha"`
	Size        int64  `json:"size"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	GitURL      string `json:"git_url"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"` // file, dir, symlink
	Content     string `json:"content,omitempty"`
	Encoding    string `json:"encoding,omitempty"`
}

// RepositorySummary is a lightweight listing entry.
type RepositorySummary struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	DefaultBranch string `json:"default_branch"`
	CloneURL      string `json:"clone_url"`
	Private       bool   `json:"private"`
}

// Repository holds forge metadata for ref resolution.
type Repository struct {
	DefaultBranch string `json:"default_branch"`
	CloneURL      string `json:"clone_url"`
	Empty         bool   `json:"empty"`
}
