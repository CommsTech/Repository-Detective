package gitea

import (
	"context"

	"git.commsnet.org/commstech/repository-detective/forge"
)

// ForgeClient adapts the Gitea client to forge.RepoClient.
type ForgeClient struct {
	*Client
}

var _ forge.RepoClient = (*ForgeClient)(nil)

// AsForgeClient returns a forge.RepoClient view of this Gitea client.
func (c *Client) AsForgeClient() forge.RepoClient {
	if c == nil {
		return nil
	}
	return &ForgeClient{Client: c}
}

func (a *ForgeClient) GetRepository(ctx context.Context, owner, repo string) (*forge.Repository, error) {
	r, err := a.Client.GetRepository(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	return &forge.Repository{
		DefaultBranch: r.DefaultBranch,
		CloneURL:      r.CloneURL,
		Empty:         r.Empty,
	}, nil
}

func (a *ForgeClient) GetFileContent(ctx context.Context, owner, repo, ref, filePath string) (string, error) {
	return a.Client.GetFileContent(ctx, owner, repo, ref, filePath)
}

func (a *ForgeClient) ListAllFiles(ctx context.Context, owner, repo, ref, dirPath string) ([]forge.RepositoryContent, error) {
	files, err := a.Client.ListAllFiles(ctx, owner, repo, ref, dirPath)
	if err != nil {
		return nil, err
	}
	out := make([]forge.RepositoryContent, len(files))
	for i, f := range files {
		out[i] = forge.RepositoryContent{
			Name: f.Name, Path: f.Path, SHA: f.SHA, Size: f.Size,
			URL: f.URL, HTMLURL: f.HTMLURL, GitURL: f.GitURL,
			DownloadURL: f.DownloadURL, Type: f.Type,
			Content: f.Content, Encoding: f.Encoding,
		}
	}
	return out, nil
}

func (a *ForgeClient) ResolveRef(ctx context.Context, owner, repo, ref string) (string, error) {
	return a.Client.ResolveRef(ctx, owner, repo, ref)
}

func (a *ForgeClient) DownloadRepositoryArchive(ctx context.Context, owner, repo, ref string, maxBytes int64) (string, func(), int64, error) {
	return a.Client.DownloadRepositoryArchive(ctx, owner, repo, ref, maxBytes)
}

func (a *ForgeClient) TestConnection(ctx context.Context) error {
	return a.Client.TestConnection(ctx)
}

func (a *ForgeClient) ListAllUserRepositories(ctx context.Context) ([]forge.RepositorySummary, error) {
	repos, err := a.Client.ListAllUserRepositories(ctx)
	if err != nil {
		return nil, err
	}
	return toForgeSummaries(repos), nil
}

func (a *ForgeClient) ListAllOrgRepositories(ctx context.Context, org string) ([]forge.RepositorySummary, error) {
	repos, err := a.Client.ListAllOrgRepositories(ctx, org)
	if err != nil {
		return nil, err
	}
	return toForgeSummaries(repos), nil
}

func toForgeSummaries(repos []RepositorySummary) []forge.RepositorySummary {
	out := make([]forge.RepositorySummary, len(repos))
	for i, r := range repos {
		out[i] = forge.RepositorySummary{
			ID:            r.ID,
			Name:          r.Name,
			FullName:      r.FullName,
			DefaultBranch: r.DefaultBranch,
			CloneURL:      r.HTMLURL,
			Private:       r.Private,
		}
	}
	return out
}
