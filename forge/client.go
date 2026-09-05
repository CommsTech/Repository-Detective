package forge

import "context"

// RepoClient lists repositories and reads source for analysis.
type RepoClient interface {
	TestConnection(ctx context.Context) error
	GetFileContent(ctx context.Context, owner, repo, ref, filePath string) (string, error)
	ListAllFiles(ctx context.Context, owner, repo, ref, dirPath string) ([]RepositoryContent, error)
	ResolveRef(ctx context.Context, owner, repo, ref string) (string, error)
	GetRepository(ctx context.Context, owner, repo string) (*Repository, error)
	DownloadRepositoryArchive(ctx context.Context, owner, repo, ref string, maxBytes int64) (path string, cleanup func(), written int64, err error)
	ListAllUserRepositories(ctx context.Context) ([]RepositorySummary, error)
	ListAllOrgRepositories(ctx context.Context, org string) ([]RepositorySummary, error)
}
