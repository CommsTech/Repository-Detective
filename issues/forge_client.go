package issues

import (
	"context"
	"fmt"

	"git.commsnet.org/commstech/repository-detective/gitea"
	"git.commsnet.org/commstech/repository-detective/github"
)

// ForgeIssue is a normalized issue from Gitea or GitHub.
type ForgeIssue struct {
	Number  int
	HTMLURL string
	Body    string
}

// IssueForge creates and updates issues on a code hosting platform.
type IssueForge interface {
	ListOpenLabeledIssues(ctx context.Context, owner, repo string, labels []string, limit, page int) ([]ForgeIssue, error)
	ListOpenIssues(ctx context.Context, owner, repo string, limit, page int) ([]ForgeIssue, error)
	CreateIssue(ctx context.Context, owner, repo, title, body string, labelNames []string) (*ForgeIssue, error)
	CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, body string) error
	AddIssueLabels(ctx context.Context, owner, repo string, issueNumber int, labelNames []string) error
}

// GiteaForge adapts the Gitea API client.
type GiteaForge struct {
	Client *gitea.Client
}

func (f *GiteaForge) ListOpenLabeledIssues(ctx context.Context, owner, repo string, labels []string, limit, page int) ([]ForgeIssue, error) {
	if f == nil || f.Client == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	if page <= 0 {
		page = 1
	}
	issues, err := f.Client.ListIssues(ctx, owner, repo, gitea.ListIssuesOptions{
		State:  "open",
		Labels: labels,
		Limit:  limit,
		Page:   page,
	})
	if err != nil {
		return nil, err
	}
	out := make([]ForgeIssue, 0, len(issues))
	for _, issue := range issues {
		out = append(out, ForgeIssue{
			Number:  issue.Number,
			HTMLURL: issue.HTMLURL,
			Body:    issue.Body,
		})
	}
	return out, nil
}

func (f *GiteaForge) ListOpenIssues(ctx context.Context, owner, repo string, limit, page int) ([]ForgeIssue, error) {
	return f.ListOpenLabeledIssues(ctx, owner, repo, nil, limit, page)
}

func (f *GiteaForge) CreateIssue(ctx context.Context, owner, repo, title, body string, labelNames []string) (*ForgeIssue, error) {
	labelIDs, err := f.Client.ResolveLabelIDs(ctx, owner, repo, labelNames)
	if err != nil {
		return nil, err
	}
	created, err := f.Client.CreateIssue(ctx, owner, repo, &gitea.CreateIssueRequest{
		Title:  title,
		Body:   body,
		Labels: labelIDs,
	})
	if err != nil {
		return nil, err
	}
	if len(labelIDs) == 0 && len(labelNames) > 0 {
		payload := make([]any, 0, len(labelNames))
		for _, name := range labelNames {
			payload = append(payload, name)
		}
		if _, err := f.Client.AddIssueLabels(ctx, owner, repo, created.Number, payload); err != nil {
			return &ForgeIssue{Number: created.Number, HTMLURL: created.HTMLURL}, fmt.Errorf("attach labels to issue #%d: %w", created.Number, err)
		}
	}
	return &ForgeIssue{Number: created.Number, HTMLURL: created.HTMLURL}, nil
}

func (f *GiteaForge) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, body string) error {
	return f.Client.CreateIssueComment(ctx, owner, repo, issueNumber, body)
}

func (f *GiteaForge) AddIssueLabels(ctx context.Context, owner, repo string, issueNumber int, labelNames []string) error {
	if len(labelNames) == 0 {
		return nil
	}
	payload := make([]any, len(labelNames))
	for i, name := range labelNames {
		payload[i] = name
	}
	_, err := f.Client.AddIssueLabels(ctx, owner, repo, issueNumber, payload)
	return err
}

// GitHubForge adapts the GitHub API client.
type GitHubForge struct {
	Client *github.Client
}

func (f *GitHubForge) ListOpenLabeledIssues(ctx context.Context, owner, repo string, labels []string, limit, page int) ([]ForgeIssue, error) {
	if f == nil || f.Client == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	if page <= 0 {
		page = 1
	}
	issues, err := f.Client.ListIssues(ctx, owner, repo, github.ListIssuesOptions{
		State:  "open",
		Labels: labels,
		Limit:  limit,
		Page:   page,
	})
	if err != nil {
		return nil, err
	}
	out := make([]ForgeIssue, 0, len(issues))
	for _, issue := range issues {
		out = append(out, ForgeIssue{
			Number:  issue.Number,
			HTMLURL: issue.HTMLURL,
			Body:    issue.Body,
		})
	}
	return out, nil
}

func (f *GitHubForge) ListOpenIssues(ctx context.Context, owner, repo string, limit, page int) ([]ForgeIssue, error) {
	return f.ListOpenLabeledIssues(ctx, owner, repo, nil, limit, page)
}

func (f *GitHubForge) CreateIssue(ctx context.Context, owner, repo, title, body string, labelNames []string) (*ForgeIssue, error) {
	if err := f.Client.EnsureRepositoryLabels(ctx, owner, repo, labelNames); err != nil {
		return nil, fmt.Errorf("ensure labels: %w", err)
	}
	created, err := f.Client.CreateIssue(ctx, owner, repo, &github.CreateIssueRequest{
		Title:  title,
		Body:   body,
		Labels: labelNames,
	})
	if err != nil {
		return nil, err
	}
	return &ForgeIssue{Number: created.Number, HTMLURL: created.HTMLURL}, nil
}

func (f *GitHubForge) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, body string) error {
	return f.Client.CreateIssueComment(ctx, owner, repo, issueNumber, body)
}

func (f *GitHubForge) AddIssueLabels(ctx context.Context, owner, repo string, issueNumber int, labelNames []string) error {
	return f.Client.AddIssueLabels(ctx, owner, repo, issueNumber, labelNames)
}
