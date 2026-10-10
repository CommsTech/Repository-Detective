package closure

import (
	"context"
	"time"

	"git.commsnet.org/commstech/repository-detective/gitea"
)

// PRClient queries pull request state.
type PRClient interface {
	GetPullRequest(ctx context.Context, owner, repo string, prNumber int) (*gitea.PullRequest, error)
}

// DetectMerge checks whether a Gitea pull request has been merged.
func DetectMerge(ctx context.Context, client PRClient, owner, repo string, prNumber int) (MergeInfo, error) {
	if client == nil || prNumber <= 0 {
		return MergeInfo{}, nil
	}
	pr, err := client.GetPullRequest(ctx, owner, repo, prNumber)
	if err != nil {
		return MergeInfo{}, err
	}
	info := MergeInfo{Merged: pr.Merged}
	if pr.Head.SHA != "" {
		info.MergeCommitSHA = pr.Head.SHA
	}
	if pr.MergedAt != "" {
		if t, err := time.Parse(time.RFC3339, pr.MergedAt); err == nil {
			info.MergedAt = t
		}
	}
	if info.Merged && info.MergedAt.IsZero() {
		info.MergedAt = time.Now().UTC()
	}
	return info, nil
}
