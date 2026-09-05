package runner_test

import (
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/runner"
)

func TestWorkerConfigValidateRequiresSecret(t *testing.T) {
	cfg := runner.WorkerConfig{CoreURL: "http://127.0.0.1:8081"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing secret error")
	}
}

func TestWorkerConfigNormalizedDefaults(t *testing.T) {
	cfg := runner.WorkerConfig{CoreURL: "http://core", SharedSecret: "secret"}.Normalized()
	if cfg.MaxConcurrent != 1 {
		t.Fatalf("max concurrent: %d", cfg.MaxConcurrent)
	}
	if len(cfg.AllowedJobTypes) == 0 {
		t.Fatal("expected default allowed job types")
	}
}

func TestJobTypeAllowed(t *testing.T) {
	if !runner.JobTypeAllowed([]string{"graph", "sbom"}, runner.JobTypeGraph) {
		t.Fatal("graph should be allowed")
	}
	if runner.JobTypeAllowed([]string{"graph"}, runner.JobTypeSBOM) {
		t.Fatal("sbom should be blocked")
	}
}

func TestRedactLogLine(t *testing.T) {
	msg := runner.RedactLogLine("failed token REPOSITORY_DETECTIVE_GITEA_TOKEN=super-secret-value")
	if strings.Contains(msg, "super-secret-value") {
		t.Fatalf("secret leaked: %q", msg)
	}
}
