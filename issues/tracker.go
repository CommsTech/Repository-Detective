package issues

import (
	"context"
)

// ExistingIssueMatch describes a prior Repository Detective issue located by fingerprint.
type ExistingIssueMatch struct {
	IssueNumber int
	IssueURL    string
	Body        string
}

// ListAllOpenLabeledIssues paginates through all open issues with product base labels.
func ListAllOpenLabeledIssues(ctx context.Context, forge IssueForge, owner, repo string) ([]ForgeIssue, error) {
	if forge == nil {
		return nil, nil
	}
	seen := make(map[int]struct{})
	var all []ForgeIssue
	for _, baseLabel := range IssueLookupBaseLabels() {
		page := 1
		for {
			batch, err := forge.ListOpenLabeledIssues(ctx, owner, repo, []string{baseLabel}, 50, page)
			if err != nil {
				return nil, err
			}
			if len(batch) == 0 {
				break
			}
			for _, issue := range batch {
				if _, ok := seen[issue.Number]; ok {
					continue
				}
				seen[issue.Number] = struct{}{}
				all = append(all, issue)
			}
			if len(batch) < 50 {
				break
			}
			page++
		}
	}
	return all, nil
}

// ListAllOpenIssues paginates all open repository issues.
func ListAllOpenIssues(ctx context.Context, forge IssueForge, owner, repo string) ([]ForgeIssue, error) {
	if forge == nil {
		return nil, nil
	}
	var all []ForgeIssue
	page := 1
	for {
		batch, err := forge.ListOpenIssues(ctx, owner, repo, 50, page)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
		if len(batch) < 50 {
			break
		}
		page++
	}
	return all, nil
}

// FindIssueByFingerprint searches open issues for a fingerprint marker.
func FindIssueByFingerprint(ctx context.Context, forge IssueForge, owner, repo, fingerprint string) (*ExistingIssueMatch, error) {
	if forge == nil || fingerprint == "" {
		return nil, nil
	}

	issues, err := ListAllOpenLabeledIssues(ctx, forge, owner, repo)
	if err != nil {
		return nil, err
	}
	if match := matchFingerprintInIssues(issues, fingerprint); match != nil {
		return match, nil
	}

	allOpen, err := ListAllOpenIssues(ctx, forge, owner, repo)
	if err != nil {
		return nil, err
	}
	return matchFingerprintInIssues(allOpen, fingerprint), nil
}

func matchFingerprintInIssues(issues []ForgeIssue, fingerprint string) *ExistingIssueMatch {
	for _, issue := range issues {
		if ExtractFingerprintFromBody(issue.Body) == fingerprint {
			return &ExistingIssueMatch{
				IssueNumber: issue.Number,
				IssueURL:    issue.HTMLURL,
				Body:        issue.Body,
			}
		}
	}
	return nil
}

// ReportNotReproduced comments on open labeled issues absent from the current scan.
func ReportNotReproduced(ctx context.Context, forge IssueForge, owner, repo, scanID string, seenFingerprints map[string]struct{}) error {
	if forge == nil || scanID == "" || len(seenFingerprints) == 0 {
		return nil
	}

	issues, err := ListAllOpenLabeledIssues(ctx, forge, owner, repo)
	if err != nil {
		return err
	}
	for _, issue := range issues {
		fp := ExtractFingerprintFromBody(issue.Body)
		if fp == "" {
			continue
		}
		if _, ok := seenFingerprints[fp]; ok {
			continue
		}
		comment := NotReproducedCommentBody(scanID)
		if err := forge.CreateIssueComment(ctx, owner, repo, issue.Number, comment); err != nil {
			return err
		}
		if err := forge.AddIssueLabels(ctx, owner, repo, issue.Number, ExpandLifecycleLabels(LifecycleNotReproduced)); err != nil {
			return err
		}
	}
	return nil
}
