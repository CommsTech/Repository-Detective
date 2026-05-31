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
	if eval.Description != "Bugbot scan passed with no findings" {
		t.Fatalf("unexpected description %q", eval.Description)
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
	if eval.Description != "Bugbot found 1 high, 1 medium findings" {
		t.Fatalf("unexpected description %q", eval.Description)
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
		{Scanner: "trivy", Status: "failed"},
	}, ChecksConfig{
		IncludeScannerFailures: true,
	})
	if eval.State != CommitStateError {
		t.Fatalf("expected error, got %q", eval.State)
	}
}

func TestEvaluateCommitStatusScannerFailureIgnored(t *testing.T) {
	eval := EvaluateCommitStatus(nil, []ScannerResultSummary{
		{Scanner: "semgrep", Status: "timed_out"},
	}, ChecksConfig{
		IncludeScannerFailures: false,
	})
	if eval.State != CommitStateSuccess {
		t.Fatalf("expected success when scanner failures ignored, got %q", eval.State)
	}
}

func TestEvaluateCommitStatusBinaryMissingNotBad(t *testing.T) {
	eval := EvaluateCommitStatus(nil, []ScannerResultSummary{
		{Scanner: "gitleaks", Status: "binary_missing"},
		{Scanner: "semgrep", Status: "disabled"},
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
	if eval.Description != "Bugbot found 1 critical findings" {
		t.Fatalf("unexpected description %q", eval.Description)
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
	if eval.State != CommitStatePending || eval.Description != "Bugbot scan started" {
		t.Fatalf("unexpected pending eval: %+v", eval)
	}
}
