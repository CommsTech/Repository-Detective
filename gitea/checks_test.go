package gitea

import "testing"

func TestEvaluateCommitStatusSuccess(t *testing.T) {
	eval := EvaluateCommitStatus(nil, nil, ChecksConfig{
		FailOn:                 "high",
		WarnOn:                 "medium",
		IncludeScannerFailures: true,
	})
	if eval.State != CommitStateSuccess {
		t.Fatalf("expected success, got %q", eval.State)
	}
	if eval.PolicyOutcome != "POLICY_MET" {
		t.Fatalf("expected POLICY_MET, got %q desc=%q", eval.PolicyOutcome, eval.Description)
	}
}

func TestEvaluateCommitStatusFailureOnHighFinding(t *testing.T) {
	eval := EvaluateCommitStatus([]string{"high", "medium"}, nil, ChecksConfig{
		FailOn: "high",
		WarnOn: "medium",
	})
	if eval.State != CommitStateFailure {
		t.Fatalf("expected failure, got %q", eval.State)
	}
	if eval.PolicyOutcome != "ACTION_REQUIRED" {
		t.Fatalf("expected ACTION_REQUIRED, got %q", eval.PolicyOutcome)
	}
}

func TestEvaluateCommitStatusWarningOnMediumFinding(t *testing.T) {
	eval := EvaluateCommitStatus([]string{"medium"}, nil, ChecksConfig{
		FailOn: "high",
		WarnOn: "medium",
	})
	if eval.State != CommitStateWarning {
		t.Fatalf("expected warning, got %q", eval.State)
	}
	if MapGiteaCommitState(eval.State) != CommitStateFailure {
		t.Fatalf("expected warning mapped to failure for Gitea")
	}
}

func TestEvaluateCommitStatusScannerFailureIncluded(t *testing.T) {
	eval := EvaluateCommitStatus(nil, []ScannerResultSummary{
		{Scanner: "trivy", Status: "failed", Required: true},
	}, ChecksConfig{
		IncludeScannerFailures: true,
	})
	if eval.State != CommitStateError {
		t.Fatalf("expected error, got %q", eval.State)
	}
	if eval.PolicyOutcome != "EVALUATION_INCOMPLETE" {
		t.Fatalf("expected EVALUATION_INCOMPLETE, got %q", eval.PolicyOutcome)
	}
}

func TestEvaluateCommitStatusScannerFailureIgnored(t *testing.T) {
	eval := EvaluateCommitStatus(nil, []ScannerResultSummary{
		{Scanner: "semgrep", Status: "timed_out", Required: false},
	}, ChecksConfig{
		IncludeScannerFailures: false,
	})
	if eval.State != CommitStateSuccess {
		t.Fatalf("expected success when optional scanner failures ignored, got %q", eval.State)
	}
}

func TestEvaluateCommitStatusRequiredBinaryMissingIncomplete(t *testing.T) {
	eval := EvaluateCommitStatus(nil, []ScannerResultSummary{
		{Scanner: "gitleaks", Status: "binary_missing", Required: true},
	}, ChecksConfig{
		IncludeScannerFailures: false,
	})
	if eval.PolicyOutcome != "EVALUATION_INCOMPLETE" {
		t.Fatalf("required missing binary should be incomplete, got %q %q", eval.PolicyOutcome, eval.Description)
	}
}

func TestEvaluateCommitStatusOptionalBinaryMissingOK(t *testing.T) {
	eval := EvaluateCommitStatus(nil, []ScannerResultSummary{
		{Scanner: "gitleaks", Status: "binary_missing", Required: false},
		{Scanner: "semgrep", Status: "disabled", Required: false},
	}, ChecksConfig{
		IncludeScannerFailures: true,
	})
	if eval.State != CommitStateSuccess {
		t.Fatalf("expected success for optional missing binaries, got %q", eval.State)
	}
}

func TestEvaluateCommitStatusCriticalCountsAsFailure(t *testing.T) {
	eval := EvaluateCommitStatus([]string{"critical"}, nil, ChecksConfig{
		FailOn: "high",
		WarnOn: "medium",
	})
	if eval.State != CommitStateFailure {
		t.Fatalf("expected failure for critical finding, got %q", eval.State)
	}
}

func TestIsCommitSHA(t *testing.T) {
	if !IsCommitSHA("abc1234567890") {
		t.Fatal("expected valid SHA")
	}
	if IsCommitSHA("main") {
		t.Fatal("branch name should not be treated as SHA")
	}
}

func TestPendingCommitStatusEvaluation(t *testing.T) {
	eval := PendingCommitStatusEvaluation()
	if eval.State != CommitStatePending {
		t.Fatalf("unexpected pending eval: %+v", eval)
	}
}

func TestObserveModeNeverBlocks(t *testing.T) {
	eval := EvaluateCommitStatusForPolicy([]string{"critical"}, nil, ChecksConfig{FailOn: "high"}, "monitor_only", "high")
	if eval.State == CommitStateFailure {
		t.Fatalf("observe must not fail, got %s", eval.State)
	}
	if eval.PolicyOutcome != "OBSERVATION_ONLY" {
		t.Fatalf("expected OBSERVATION_ONLY, got %q", eval.PolicyOutcome)
	}
}
