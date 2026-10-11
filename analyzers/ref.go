package analyzers

import (
	"strings"

	"git.commsnet.org/commstech/repository-detective/gitea"
)

// looksLikeCommitSHA reports whether ref appears to be a git commit hash.
func looksLikeCommitSHA(ref string) bool {
	ref = strings.TrimSpace(ref)
	if len(ref) < 7 || len(ref) > 40 {
		return false
	}
	for _, r := range ref {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			continue
		}
		return false
	}
	return true
}

// PullRequestRef returns the best archive/content ref and whether it is commit-pinned.
func PullRequestRef(pr *gitea.PullRequest) (ref string, commitPinned bool) {
	if pr == nil {
		return "", false
	}
	if sha := strings.TrimSpace(pr.Head.SHA); sha != "" {
		return sha, true
	}
	return strings.TrimSpace(pr.HeadBranch), false
}
