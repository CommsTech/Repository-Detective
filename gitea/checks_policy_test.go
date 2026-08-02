package gitea_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/gitea"
)

func TestEvaluateCommitStatusForPolicyIssueOnly(t *testing.T) {
	cfg := gitea.ChecksConfig{FailOn: "high", WarnOn: "medium"}
	eval := gitea.EvaluateCommitStatusForPolicy([]string{"high"}, nil, cfg, "issue_only", "high")
	if eval.State == gitea.CommitStateFailure {
		t.Fatalf("issue_only should not fail status, got %s", eval.State)
	}
}

func TestEvaluateCommitStatusForPolicyGatePR(t *testing.T) {
	cfg := gitea.ChecksConfig{FailOn: "high", WarnOn: "medium"}
	eval := gitea.EvaluateCommitStatusForPolicy([]string{"high"}, nil, cfg, "gate_pr", "high")
	if eval.State != gitea.CommitStateFailure {
		t.Fatalf("gate_pr should fail on high, got %s", eval.State)
	}
}

func TestEvaluateCommitStatusForPolicyMonitorOnly(t *testing.T) {
	cfg := gitea.ChecksConfig{FailOn: "high"}
	eval := gitea.EvaluateCommitStatusForPolicy([]string{"critical"}, nil, cfg, "monitor_only", "high")
	if eval.State == gitea.CommitStateFailure {
		t.Fatalf("monitor_only should not fail, got %s", eval.State)
	}
}
