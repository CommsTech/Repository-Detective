package preinstall_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/preinstall"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestComputeRiskScoreLowConfidenceGraphNotBlocker(t *testing.T) {
	findings := []store.AuditFinding{
		{Severity: "high", Source: "graph", Category: "architecture", Confidence: 0.4},
		{Severity: "high", Source: "graph", Category: "architecture", Confidence: 0.35},
	}
	out := preinstall.ComputeRiskScore(findings, nil)
	if out.Recommendation == store.AuditRecommendationDoNotInstall {
		t.Fatalf("low-confidence graph should not block install, got %s score=%d", out.Recommendation, out.Score)
	}
	if out.Score >= 20 {
		t.Fatalf("low-confidence graph capped, got score %d", out.Score)
	}
}

func TestComputeRiskScoreSafeWhenNoFindings(t *testing.T) {
	out := preinstall.ComputeRiskScore(nil, nil)
	if out.Score != 0 || out.Recommendation != store.AuditRecommendationSafe {
		t.Fatalf("expected safe zero score, got %+v", out)
	}
}

func TestComputeRiskScoreHighFindingCautionOrBlock(t *testing.T) {
	findings := []store.AuditFinding{{Severity: "high", Source: "semgrep"}}
	out := preinstall.ComputeRiskScore(findings, nil)
	if out.Score < 20 {
		t.Fatalf("expected score >= 20, got %d", out.Score)
	}
	if out.Recommendation != store.AuditRecommendationCaution && out.Recommendation != store.AuditRecommendationDoNotInstall {
		t.Fatalf("unexpected recommendation: %s", out.Recommendation)
	}
}

func TestComputeRiskScoreSecretMinimumCaution(t *testing.T) {
	findings := []store.AuditFinding{{Severity: "low", Source: "gitleaks", Category: "secret"}}
	out := preinstall.ComputeRiskScore(findings, nil)
	if out.Recommendation != store.AuditRecommendationCaution {
		t.Fatalf("secret finding should force caution minimum, got %s", out.Recommendation)
	}
}

func TestComputeRiskScoreScannerFailures(t *testing.T) {
	scanners := []store.AuditScannerResult{{Scanner: "trivy", Status: "binary_missing"}}
	out := preinstall.ComputeRiskScore(nil, scanners)
	if out.Score < 5 {
		t.Fatalf("scanner failure should add points, got %d", out.Score)
	}
}

func TestComputeRiskScoreDoNotInstall(t *testing.T) {
	findings := []store.AuditFinding{
		{Severity: "critical"},
		{Severity: "critical"},
	}
	out := preinstall.ComputeRiskScore(findings, nil)
	if out.Recommendation != store.AuditRecommendationDoNotInstall {
		t.Fatalf("expected do_not_install, got %s", out.Recommendation)
	}
}

func TestComputeRiskScoreHealthLowFindingsCapped(t *testing.T) {
	var findings []store.AuditFinding
	for i := 0; i < 10; i++ {
		findings = append(findings, store.AuditFinding{Severity: "low", Source: "tech_debt", Category: "tech_debt"})
	}
	out := preinstall.ComputeRiskScore(findings, nil)
	if out.Score >= 20 {
		t.Fatalf("health low findings should not dominate score, got %d", out.Score)
	}
}
