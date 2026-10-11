package runner_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/runner"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestSignAndVerifyRequest(t *testing.T) {
	secret := "test-runner-secret-key"
	body := []byte(`{"runner_id":"worker-1"}`)
	ts := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	nonce := "nonce-abc"
	path := "/api/v1/runner/jobs/claim"
	sig := runner.SignRequest(secret, ts, nonce, "POST", path, body)
	if err := runner.VerifyRequest(secret, ts, nonce, sig, "POST", path, body, time.Now().UTC()); err != nil {
		t.Fatalf("expected valid signature: %v", err)
	}
	if err := runner.VerifyRequest(secret, ts, nonce, "bad", "POST", path, body, time.Now().UTC()); err == nil {
		t.Fatal("expected invalid signature rejection")
	}
}

func TestExpiredTimestampRejected(t *testing.T) {
	secret := "test-runner-secret-key"
	old := strconv.FormatInt(time.Now().UTC().Add(-10*time.Minute).Unix(), 10)
	path := "/api/v1/runner/jobs/claim"
	sig := runner.SignRequest(secret, old, "n1", "POST", path, nil)
	if err := runner.VerifyRequest(secret, old, "n1", sig, "POST", path, nil, time.Now().UTC()); err == nil {
		t.Fatal("expected expired timestamp error")
	}
}

func TestShouldDelegatePolicy(t *testing.T) {
	cfg := runner.Config{DelegationEnabled: true, Mode: runner.ModeGiteaActions, SharedSecret: "x"}
	effective := store.EffectiveSettings{RunnerPolicy: "gitea_actions"}
	if runner.ShouldDelegate(cfg, effective, store.TriggerScheduled) != runner.DecisionDelegate {
		t.Fatal("expected delegate for scheduled scan")
	}
	if runner.ShouldDelegate(cfg, effective, store.TriggerPush) != runner.DecisionCore {
		t.Fatal("webhook push should stay on core in phase 12")
	}
	cfg.Mode = runner.ModeCore
	if runner.ShouldDelegate(cfg, effective, store.TriggerScheduled) != runner.DecisionCore {
		t.Fatal("global core mode disables delegation")
	}
}

func TestShouldDelegateNativeMode(t *testing.T) {
	cfg := runner.Config{DelegationEnabled: true, Mode: runner.ModeNative, SharedSecret: "x"}
	effective := store.EffectiveSettings{RunnerPolicy: runner.ModeAuto}
	if runner.ShouldDelegate(cfg, effective, store.TriggerManual) != runner.DecisionDelegate {
		t.Fatal("expected delegate for native mode with auto repo policy")
	}
}

func TestValidateResultRejectsMismatch(t *testing.T) {
	job := runner.JobView{JobID: "j1", ScanID: "scan-a"}
	result := runner.JobResult{JobID: "j1", ScanID: "scan-b", Status: runner.JobStatusCompleted}
	if err := runner.ValidateResultBasic(job, result, 1024*1024); err == nil {
		t.Fatal("expected scan id mismatch")
	}
}

func TestValidateResultRejectsForbiddenAction(t *testing.T) {
	job := runner.JobView{JobID: "j1", ScanID: "scan-a"}
	result := runner.JobResult{JobID: "j1", ScanID: "scan-a", ForbiddenAction: "issue_create"}
	if err := runner.ValidateResultBasic(job, result, 1024*1024); err == nil {
		t.Fatal("expected forbidden action rejection")
	}
}

func TestValidateResultRejectsOversized(t *testing.T) {
	job := runner.JobView{JobID: "j1", ScanID: "scan-a"}
	result := runner.JobResult{
		JobID: "j1", ScanID: "scan-a", Status: runner.JobStatusCompleted,
		Findings: make([]runner.FindingResult, 2000),
	}
	for i := range result.Findings {
		result.Findings[i] = runner.FindingResult{
			Fingerprint: "fp", Category: "security", Severity: "high", Confidence: 0.9,
			Source: "test", RuleID: "R", File: "a.go", Line: 1,
			Title: "finding", Description: strings.Repeat("x", 512),
		}
	}
	if err := runner.ValidateResultBasic(job, result, 4096); err == nil {
		t.Fatal("expected oversized result rejection")
	} else if !errors.Is(err, runner.ErrResultTooLarge) {
		t.Fatalf("expected ErrResultTooLarge, got %v", err)
	}
}

func TestValidateResultRejectsJobIDMismatch(t *testing.T) {
	job := runner.JobView{JobID: "j1", ScanID: "scan-a"}
	result := runner.JobResult{JobID: "j2", ScanID: "scan-a", Status: runner.JobStatusCompleted}
	err := runner.ValidateResultBasic(job, result, 1024*1024)
	if err == nil {
		t.Fatal("expected job id mismatch rejection")
	}
	if !errors.Is(err, runner.ErrUnknownJob) {
		t.Fatalf("expected ErrUnknownJob, got %v", err)
	}
}

func TestBuildJobSpecExposesForbiddenTasks(t *testing.T) {
	repo := store.Repository{ForgeType: store.ForgeTypeGitea, Owner: "o", Name: "r", FullName: "o/r", CloneURL: "https://git.example/o/r.git"}
	spec := runner.BuildJobSpec(runner.Config{CallbackBaseURL: "https://core.example"}, "j1", repo, "scan-1", "main", "", analyzers.PolicySnapshot{})
	for _, forbidden := range runner.ForbiddenTasks {
		found := false
		for _, task := range spec.ForbiddenTasks {
			if task == forbidden {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected forbidden task %q in spec", forbidden)
		}
	}
	for _, allowed := range runner.AllowedTasks {
		found := false
		for _, task := range spec.AllowedTasks {
			if task == allowed {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected allowed task %q in spec", allowed)
		}
	}
}

func TestShouldFallbackToCore(t *testing.T) {
	if !runner.ShouldFallbackToCore(store.EffectiveSettings{RunnerPolicy: "auto"}) {
		t.Fatal("auto policy should allow core fallback")
	}
	if runner.ShouldFallbackToCore(store.EffectiveSettings{RunnerPolicy: "gitea_actions"}) {
		t.Fatal("gitea_actions policy should not fall back silently")
	}
}

func TestStartupValidRequiresSecret(t *testing.T) {
	cfg := runner.Config{DelegationEnabled: true, Mode: runner.ModeGiteaActions}
	if err := cfg.StartupValid(); err == nil {
		t.Fatal("expected secret required when delegation enabled")
	}
}
