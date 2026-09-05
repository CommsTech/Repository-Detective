package closure

import (
	"context"
	"testing"

	"git.commsnet.org/commstech/repository-detective/gitea"
)

type stubPR struct {
	pr *gitea.PullRequest
	err error
}

func (s stubPR) GetPullRequest(ctx context.Context, owner, repo string, prNumber int) (*gitea.PullRequest, error) {
	return s.pr, s.err
}

func TestDetectMergeOpenPR(t *testing.T) {
	info, err := DetectMerge(context.Background(), stubPR{pr: &gitea.PullRequest{Merged: false}}, "o", "r", 1)
	if err != nil || info.Merged {
		t.Fatal("open PR should not be merged")
	}
}

func TestDetectMergeMergedPR(t *testing.T) {
	info, err := DetectMerge(context.Background(), stubPR{pr: &gitea.PullRequest{Merged: true, Head: gitea.PullRequestGitRef{SHA: "deadbeef"}}}, "o", "r", 1)
	if err != nil || !info.Merged || info.MergeCommitSHA != "deadbeef" {
		t.Fatalf("merged PR not detected: %+v err=%v", info, err)
	}
}

func TestDetectMergeAPIFailure(t *testing.T) {
	_, err := DetectMerge(context.Background(), stubPR{err: context.DeadlineExceeded}, "o", "r", 1)
	if err == nil {
		t.Fatal("expected error")
	}
}
