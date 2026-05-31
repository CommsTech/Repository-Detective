package issues

import (
	"context"

	"git.commsnet.org/commstech/bugbot/gitea"
)

// ExistingIssueMatch describes a prior Bugbot issue located by fingerprint.
type ExistingIssueMatch struct {
	IssueNumber int
	IssueURL    string
	Body        string
}

// FindIssueByFingerprint searches open labeled issues for a fingerprint marker.
func FindIssueByFingerprint(ctx context.Context, client *gitea.Client, owner, repo, fingerprint string) (*ExistingIssueMatch, error) {
	if client == nil || fingerprint == "" {
		return nil, nil
	}

	seen := make(map[int]struct{})
	for _, baseLabel := range IssueLookupBaseLabels() {
		issues, err := client.ListIssues(ctx, owner, repo, gitea.ListIssuesOptions{
			State:  "open",
			Labels: []string{baseLabel},
			Limit:  100,
		})
		if err != nil {
			return nil, err
		}

		for _, issue := range issues {
			if _, ok := seen[issue.Number]; ok {
				continue
			}
			seen[issue.Number] = struct{}{}
			if ExtractFingerprintFromBody(issue.Body) == fingerprint {
				return &ExistingIssueMatch{
					IssueNumber: issue.Number,
					IssueURL:    issue.HTMLURL,
					Body:        issue.Body,
				}, nil
			}
		}
	}
	return nil, nil
}

// ReportNotReproduced comments on open labeled issues absent from the current scan.
func ReportNotReproduced(ctx context.Context, client *gitea.Client, owner, repo, scanID string, seenFingerprints map[string]struct{}) error {
	if client == nil || scanID == "" || len(seenFingerprints) == 0 {
		return nil
	}

	seenIssues := make(map[int]struct{})
	for _, baseLabel := range IssueLookupBaseLabels() {
		issues, err := client.ListIssues(ctx, owner, repo, gitea.ListIssuesOptions{
			State:  "open",
			Labels: []string{baseLabel},
			Limit:  100,
		})
		if err != nil {
			return err
		}

		for _, issue := range issues {
			if _, ok := seenIssues[issue.Number]; ok {
				continue
			}
			seenIssues[issue.Number] = struct{}{}

			fp := ExtractFingerprintFromBody(issue.Body)
			if fp == "" {
				continue
			}
			if _, ok := seenFingerprints[fp]; ok {
				continue
			}
			comment := NotReproducedCommentBody(scanID)
			if err := client.CreateIssueComment(ctx, owner, repo, issue.Number, comment); err != nil {
				return err
			}
			_, _ = client.AddIssueLabels(ctx, owner, repo, issue.Number, ExpandLifecycleLabel(LifecycleNotReproduced))
		}
	}
	return nil
}
