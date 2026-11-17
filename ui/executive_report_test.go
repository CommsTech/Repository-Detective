package ui

import (
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/notify"
	"git.commsnet.org/commstech/repository-detective/operator"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestBuildFleetExecutiveSummaryRecommendation(t *testing.T) {
	summary := store.DashboardSummary{
		OpenFindingsCount:      10,
		OpenFindingsBySeverity: map[string]int{"critical": 1, "high": 2},
		TotalRepositories:      3,
		RecentScans:            []store.ScanWithRepo{{}},
	}
	ex := buildFleetExecutiveSummary(summary)
	if ex.Recommendation != "blocked" {
		t.Fatalf("expected blocked recommendation, got %q", ex.Recommendation)
	}
	if ex.RecommendationLabel == "" {
		t.Fatal("expected recommendation label")
	}
}

func TestBuildRepoExecutiveSummaryNoOpenFindings(t *testing.T) {
	repo := store.Repository{FullName: "org/app"}
	ex := buildRepoExecutiveSummary(
		repo,
		map[string]int{},
		map[string]int{},
		map[string]int{"actionable": 0, "review": 0},
		nil,
		nil,
		nil,
		store.EffectiveSettings{PolicyLevel: "standard", WorkspaceMode: "archive"},
		"standard",
	)
	if ex.Recommendation != "caution" {
		t.Fatalf("expected caution without scan evidence, got %q", ex.Recommendation)
	}
	if !strings.Contains(ex.ConfidenceLevel, "no scan evidence") {
		t.Fatalf("expected missing scan evidence warning, got %q", ex.ConfidenceLevel)
	}
}

func TestBuildCapabilityStatusesRemediationPRDisabled(t *testing.T) {
	readiness := operator.Readiness{
		Features: operator.FeatureFlags{
			RemediationPREnabled:      false,
			RemediationPlannerEnabled: true,
		},
	}
	statuses := buildCapabilityStatuses(readiness, notify.DefaultConfig(), PlatformContext{}, "/ui")
	var pr *CapabilityStatus
	for i := range statuses {
		if statuses[i].Name == "Remediation PR" {
			pr = &statuses[i]
			break
		}
	}
	if pr == nil {
		t.Fatal("missing remediation PR capability")
	}
	if pr.State != "disabled" {
		t.Fatalf("expected disabled, got %q", pr.State)
	}
	if pr.Reason == "" || pr.SettingsURL == "" {
		t.Fatal("expected reason and settings link")
	}
	if !strings.Contains(pr.SettingsURL, "/configure#remediation-pr") {
		t.Fatalf("expected configure anchor, got %q", pr.SettingsURL)
	}
}

func TestRepoRecommendationRequiresActualScannerEvidence(t *testing.T) {
	scans := []store.Scan{{Status: "completed"}}
	rec, _ := repoRecommendationWithEvidence(0, 0, 0, scans, nil)
	if rec != "caution" {
		t.Fatalf("no evidence must be caution, got %q", rec)
	}
	rec, _ = repoRecommendationWithEvidence(0, 0, 0, scans, []store.ScannerResultRecord{{ScannerName: "trivy", Status: "failed"}})
	if rec != "caution" {
		t.Fatalf("failed scanner must be caution, got %q", rec)
	}
	rec, _ = repoRecommendationWithEvidence(1, 0, 1, scans, nil)
	if rec != "blocked" {
		t.Fatalf("critical finding should remain blocked, got %q", rec)
	}
}
