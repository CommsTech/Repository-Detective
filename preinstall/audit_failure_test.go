package preinstall_test

import (
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/preinstall"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestFailedCloneShowsRiskUnavailable(t *testing.T) {
	req := store.AuditRequest{AuditID: "a1", Status: store.AuditStatusQueued}
	preinstall.ApplyAuditFailure(&req, preinstall.FailureStageClone, "git clone failed: git operation failed", preinstall.SandboxMeta{SandboxID: "s1"})
	if req.RiskScore != preinstall.RiskScoreUnavailable {
		t.Fatalf("risk score: got %d want unavailable sentinel", req.RiskScore)
	}
	if req.Recommendation != store.AuditRecommendationAuditFailed {
		t.Fatalf("recommendation: %q", req.Recommendation)
	}
	if preinstall.RiskScoreDisplay(req) != "unavailable" {
		t.Fatalf("display: %q", preinstall.RiskScoreDisplay(req))
	}
	if preinstall.RecommendationDisplay(req) != "audit failed" {
		t.Fatalf("rec display: %q", preinstall.RecommendationDisplay(req))
	}
}

func TestFailedScannerSetupShowsRiskUnavailable(t *testing.T) {
	req := store.AuditRequest{}
	preinstall.ApplyAuditFailure(&req, preinstall.FailureStageScannerSetup, "trivy not installed", preinstall.SandboxMeta{})
	if preinstall.RiskScoreDisplay(req) != "unavailable" {
		t.Fatal("expected unavailable risk display")
	}
}

func TestCompletedSafeAuditCanShowLowScore(t *testing.T) {
	req := store.AuditRequest{
		Status:         store.AuditStatusCompleted,
		RiskScore:      0,
		Recommendation: store.AuditRecommendationSafe,
	}
	if preinstall.RiskScoreDisplay(req) != "0 / 100" {
		t.Fatalf("display: %q", preinstall.RiskScoreDisplay(req))
	}
	if preinstall.RecommendationDisplay(req) != "safe" {
		t.Fatalf("rec: %q", preinstall.RecommendationDisplay(req))
	}
}

func TestFailedAuditSummaryHasZeroIssues(t *testing.T) {
	req := store.AuditRequest{}
	preinstall.ApplyAuditFailure(&req, preinstall.FailureStageClone, "clone failed", preinstall.SandboxMeta{})
	body := string(req.SummaryJSON)
	if !strings.Contains(body, `"issues_created":0`) || !strings.Contains(body, `"prs_created":0`) {
		t.Fatalf("summary: %s", body)
	}
}

func TestSanitizeFailureMessageRedactsSecrets(t *testing.T) {
	msg := preinstall.SanitizeFailureMessage("token REPOSITORY_DETECTIVE_GITEA_TOKEN=super-secret-value failed")
	if strings.Contains(msg, "super-secret-value") {
		t.Fatalf("secret leaked: %q", msg)
	}
}

func TestClassifyFailureStageClone(t *testing.T) {
	if preinstall.ClassifyFailureStage("git clone failed") != preinstall.FailureStageClone {
		t.Fatal("expected clone stage")
	}
}
