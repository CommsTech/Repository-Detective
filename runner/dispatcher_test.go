package runner_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/runner"
)

func TestSelectRunnerModeNativeDefault(t *testing.T) {
	cfg := runner.Config{Mode: runner.ModeAuto}
	if got := runner.SelectRunnerMode(cfg, runner.JobTypeScanFullRepo); got != runner.ModeNative {
		t.Fatalf("scan job: got %q want native", got)
	}
}

func TestSelectRunnerModeRemediationVerifyGiteaActions(t *testing.T) {
	cfg := runner.Config{Mode: runner.ModeAuto}
	if got := runner.SelectRunnerMode(cfg, runner.JobTypeRemediationVerify); got != runner.ModeGiteaActions {
		t.Fatalf("verify job: got %q want gitea_actions", got)
	}
}

func TestSelectRunnerModeExplicitNative(t *testing.T) {
	cfg := runner.Config{Mode: runner.ModeNative}
	if got := runner.SelectRunnerMode(cfg, runner.JobTypeRemediationVerify); got != runner.ModeNative {
		t.Fatalf("got %q", got)
	}
}
