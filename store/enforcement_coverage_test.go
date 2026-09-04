package store_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestEnforcementModeMapping(t *testing.T) {
	if store.EnforcementModeFromPolicyLevel(store.PolicyMonitorOnly) != store.EnforcementObserve {
		t.Fatal("monitor_only → observe")
	}
	if store.EnforcementModeFromPolicyLevel(store.PolicyIssueOnly) != store.EnforcementWarn {
		t.Fatal("issue_only → warn")
	}
	if store.EnforcementModeFromPolicyLevel(store.PolicyGatePR) != store.EnforcementEnforce {
		t.Fatal("gate_pr → enforce")
	}
	if store.PolicyLevelFromEnforcementMode(store.EnforcementObserve) != store.PolicyMonitorOnly {
		t.Fatal("observe → monitor_only")
	}
}

func TestRequiredScannersLight(t *testing.T) {
	e := store.ProfileDefaults(store.ScanProfileLight)
	e.ScanProfile = store.ScanProfileLight
	req := store.RequiredScannersForProfile(store.ScanProfileLight, e)
	if len(req) != 2 {
		t.Fatalf("light required=%v", req)
	}
}

func TestRequiredMissingBlocksCoverage(t *testing.T) {
	e := store.ProfileDefaults(store.ScanProfileLight)
	e.ScanProfile = store.ScanProfileLight
	sum := store.BuildScannerCoverageSummary(store.ScanProfileLight, e, []struct {
		Scanner string
		Status  string
		Detail  string
	}{
		{Scanner: "gitleaks", Status: "clean"},
		{Scanner: "trivy", Status: "binary_missing"},
	})
	if len(sum.RequiredIncomplete) == 0 {
		t.Fatal("expected incomplete required scanner")
	}
}
