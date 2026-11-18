package patcher

import (
	"git.commsnet.org/commstech/repository-detective/remediation"
	"strings"
	"testing"
)

func TestAllValidationCommandsMustBeAllowed(t *testing.T) {
	cfg := enabledPRConfig()
	cfg.UseRunnerVerification = false // explicitly opt into documented local Class-B execution for this eligibility test
	plan := EligiblePlan()
	plan.ValidationCommands = []string{"go vet ./...", "curl https://example.invalid"}
	got := CheckPREligibility(plan, connectedRepo(), cfg)
	if got.Eligible || got.Checklist["validation"] {
		t.Fatalf("unsupported second command was skipped: %+v", got)
	}
	if !strings.Contains(strings.Join(got.BlockedReasons, " "), "every validation command") {
		t.Fatalf("missing reason: %+v", got)
	}
	plan.ValidationCommands = []string{"go vet ./...", "go test ./..."}
	got = CheckPREligibility(plan, connectedRepo(), cfg)
	if !got.Eligible {
		t.Fatalf("expected allowlisted set to pass under explicit local mode: %+v", got.BlockedReasons)
	}
}

func TestRunnerVerificationCannotSilentlyRunLocally(t *testing.T) {
	cfg := enabledPRConfig()
	cfg.UseRunnerVerification = true // explicit: this is the blocked-by-default boundary
	got := CheckPREligibility(EligiblePlan(), connectedRepo(), cfg)
	if got.Eligible || got.Checklist["runner_verification"] {
		t.Fatalf("claimed unavailable runner verification: %+v", got)
	}
	if !strings.Contains(strings.Join(got.BlockedReasons, " "), "runner-based PR validation") {
		t.Fatal("missing runner boundary explanation")
	}
}

func TestTerminalPlansCannotBeResurrectedWhenApprovalOptional(t *testing.T) {
	cfg := enabledPRConfig()
	cfg.UseRunnerVerification = false
	cfg.RequireApproval = false
	for _, status := range []string{remediation.StatusRejected, remediation.StatusSuperseded} {
		plan := EligiblePlan()
		plan.Status = status
		got := CheckPREligibility(plan, connectedRepo(), cfg)
		if got.Eligible || got.Checklist["terminal_status"] {
			t.Fatalf("terminal plan %q was eligible", status)
		}
	}
}
