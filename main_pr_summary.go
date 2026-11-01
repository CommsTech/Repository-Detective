package main

import (
	"context"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/gitea"
	"git.commsnet.org/commstech/repository-detective/issues"
	"git.commsnet.org/commstech/repository-detective/store"
)

// giteaPRCommentAPI adapts gitea.Client to issues.PRCommentAPI.
type giteaPRCommentAPI struct {
	client *gitea.Client
}

func (a giteaPRCommentAPI) ListIssueComments(ctx context.Context, owner, repo string, issueNumber int) ([]issues.CommentRef, error) {
	raw, err := a.client.ListIssueComments(ctx, owner, repo, issueNumber)
	if err != nil {
		return nil, err
	}
	out := make([]issues.CommentRef, 0, len(raw))
	for _, c := range raw {
		out = append(out, issues.CommentRef{ID: c.ID, Body: c.Body})
	}
	return out, nil
}

func (a giteaPRCommentAPI) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, body string) (int64, error) {
	return a.client.CreateIssueCommentReturningID(ctx, owner, repo, issueNumber, body)
}

func (a giteaPRCommentAPI) EditIssueComment(ctx context.Context, owner, repo string, commentID int64, body string) error {
	return a.client.EditIssueComment(ctx, owner, repo, commentID, body)
}

func (a giteaPRCommentAPI) DeleteIssueComment(ctx context.Context, owner, repo string, commentID int64) error {
	return a.client.DeleteIssueComment(ctx, owner, repo, commentID)
}

// maybePostPRPolicySummary upserts one compact PR comment summarizing policy/findings.
// Idempotent via <!-- repository-detective-policy-summary --> marker (RD-006A).
func maybePostPRPolicySummary(ctx context.Context, owner, repo string, prNumber int, result *analyzers.AnalysisResult, effective store.EffectiveSettings, eval gitea.CommitStatusEvaluation, repositoryID int64) {
	if prNumber <= 0 || giteaClient == nil {
		return
	}
	if result == nil {
		return
	}
	mode := store.EnforcementModeFromPolicyLevel(effective.PolicyLevel)
	if mode == store.EnforcementObserve && eval.PolicyOutcome == gitea.PolicyOutcomeObservationOnly {
		if len(result.Issues) == 0 && eval.State != gitea.CommitStateError {
			return
		}
	}

	body := issues.RenderPRPolicySummary(issues.PRPolicySummaryInput{
		Outcome:         eval.PolicyOutcome,
		EnforcementMode: mode,
		Description:     eval.Description,
		IssueCount:      len(filterIssuesWithSuppression(repositoryID, result.Issues)),
		ScannerCoverage: formatScannerCoverageLine(result, effective),
		ScanID:          result.ScanID,
		CommitSHA:       result.CommitSHA,
		UIBase:          strings.TrimRight(config.PublicURL, "/"),
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	})
	res := issues.UpsertPRPolicySummary(ctx, giteaPRCommentAPI{client: giteaClient}, owner, repo, prNumber, body)
	if res.Err != nil {
		logger.Warnf("PR policy summary upsert failed for %s/%s#%d action=%s: %v", owner, repo, prNumber, res.Action, res.Err)
		return
	}
	logger.Infof("PR policy summary %s on %s/%s#%d outcome=%s duplicates_removed=%d",
		res.Action, owner, repo, prNumber, eval.PolicyOutcome, res.DuplicatesRemoved)
}

func formatScannerCoverageLine(result *analyzers.AnalysisResult, effective store.EffectiveSettings) string {
	if result == nil {
		return ""
	}
	rows := make([]struct {
		Scanner string
		Status  string
		Detail  string
	}, 0, len(result.ScannerResults))
	for _, sr := range result.ScannerResults {
		rows = append(rows, struct {
			Scanner string
			Status  string
			Detail  string
		}{Scanner: sr.Scanner, Status: string(sr.Status), Detail: sr.Detail})
	}
	sum := store.BuildScannerCoverageSummary(effective.ScanProfile, effective, rows)
	return store.FormatCoverageRatio(sum)
}
