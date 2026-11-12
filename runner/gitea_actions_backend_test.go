package runner_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/runner"
)

func TestGiteaActionsBackendDisabledByDefault(t *testing.T) {
	cfg := runner.GiteaActionsBackendConfig{}.Normalized()
	if cfg.Enabled {
		t.Fatal("expected disabled by default")
	}
}

func TestBuildWorkflowTriggerRequest(t *testing.T) {
	cfg := runner.GiteaActionsBackendConfig{Enabled: true}
	req := runner.BuildWorkflowTrigger(cfg, "o", "r", "main", "scan123")
	if req.Workflow != "repository-detective-verify.yml" {
		t.Fatalf("workflow: %q", req.Workflow)
	}
	if req.Inputs["scan_id"] != "scan123" {
		t.Fatalf("inputs: %v", req.Inputs)
	}
}
