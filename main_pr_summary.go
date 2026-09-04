package main

import (
	"context"
	"strings"

	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/gitea"
	"git.commsnet.org/commstech/repository-detective/issues"
	"git.commsnet.org/commstech/repository-detective/store"
)

// maybePostPRPolicySummary posts one compact PR comment summarizing policy/findings.
// It does not create per-finding inline review comments — issues remain canonical.
func maybePostPRPolicySummary(ctx context.Context, owner, repo string, prNumber int, result *analyzers.AnalysisResult, effective store.EffectiveSettings, eval gitea.CommitStatusEvaluation, repositoryID int64) {
	if prNumber <= 0 || giteaClient == nil {
		return
	}
	if result == nil {
		return
	}
	// Skip Observe mode noise unless findings or incomplete evaluation exist.
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
	})
	if err := giteaClient.CreateIssueComment(ctx, owner, repo, prNumber, body); err != nil {
		logger.Warnf("PR policy summary comment failed for %s/%s#%d: %v", owner, repo, prNumber, err)
		return
	}
	logger.Infof("Posted compact PR policy summary on %s/%s#%d outcome=%s", owner, repo, prNumber, eval.PolicyOutcome)
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
