package gitea

import (
	"context"

	"github.com/sirupsen/logrus"
)

// StatusReporter posts optional Gitea commit statuses for scans.
type StatusReporter struct {
	client  *Client
	cfg     ChecksConfig
	enabled bool
	logger  *logrus.Logger
}

// NewStatusReporter creates a commit status reporter.
func NewStatusReporter(client *Client, enabled bool, cfg ChecksConfig, logger *logrus.Logger) *StatusReporter {
	return &StatusReporter{
		client:  client,
		cfg:     cfg,
		enabled: enabled,
		logger:  logger,
	}
}

// ReportPending posts a pending commit status when SHA is available.
func (r *StatusReporter) ReportPending(ctx context.Context, owner, repo, sha string) {
	if !r.enabled {
		return
	}
	if !IsCommitSHA(sha) {
		r.logger.Infof("Repository-Detective scan skipped status: no commit SHA")
		return
	}
	r.post(ctx, owner, repo, sha, PendingCommitStatusEvaluation())
}

// ReportFinal posts the final commit status from issue severities and scanner results.
func (r *StatusReporter) ReportFinal(ctx context.Context, owner, repo, sha string, severities []string, scannerResults []ScannerResultSummary, analysisFailed bool) {
	r.ReportFinalWithPolicy(ctx, owner, repo, sha, severities, scannerResults, analysisFailed, "issue_only", "")
}

// ReportFinalWithPolicy posts commit status using per-repo policy settings and returns the evaluation.
func (r *StatusReporter) ReportFinalWithPolicy(ctx context.Context, owner, repo, sha string, severities []string, scannerResults []ScannerResultSummary, analysisFailed bool, policyLevel, severityGate string) CommitStatusEvaluation {
	if !r.enabled {
		return CommitStatusEvaluation{}
	}
	if !IsCommitSHA(sha) {
		r.logger.Infof("Repository-Detective scan skipped status: no commit SHA")
		return CommitStatusEvaluation{}
	}

	var eval CommitStatusEvaluation
	if analysisFailed {
		eval = AnalysisFailedCommitStatusEvaluation()
	} else {
		eval = EvaluateCommitStatusForPolicy(severities, scannerResults, r.cfg, policyLevel, severityGate)
		if IsRemediationPolicyLevel(policyLevel) {
			r.logger.Infof("Policy level %q uses gate_pr status behavior until remediation PRs land (Phase 9+)", policyLevel)
		}
	}
	r.post(ctx, owner, repo, sha, eval)
	return eval
}

func (r *StatusReporter) post(ctx context.Context, owner, repo, sha string, eval CommitStatusEvaluation) {
	status := &CommitStatus{
		State:       eval.State,
		TargetURL:   r.cfg.TargetURL,
		Description: eval.Description,
		Context:     r.cfg.Context,
	}
	if err := r.client.CreateCommitStatus(ctx, owner, repo, sha, status); err != nil {
		r.logger.Warnf("Failed to post Gitea commit status (%s): %v", eval.State, err)
		return
	}
	r.logger.Infof("Posted Gitea commit status state=%s context=%q", MapGiteaCommitState(eval.State), r.cfg.Context)
}
