package store_test

import (
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func coverageRows(pairs ...string) []struct {
	Scanner string
	Status  string
	Detail  string
} {
	var rows []struct {
		Scanner string
		Status  string
		Detail  string
	}
	for i := 0; i+1 < len(pairs); i += 2 {
		rows = append(rows, struct {
			Scanner string
			Status  string
			Detail  string
		}{Scanner: pairs[i], Status: pairs[i+1]})
	}
	return rows
}

func TestLightRequiredSurvivesDisable(t *testing.T) {
	e := store.ProfileDefaults(store.ScanProfileLight)
	e.ScanProfile = store.ScanProfileLight
	e.EnableGitleaks = false
	e.EnableTrivy = true
	req := store.RequiredScannersForProfile(store.ScanProfileLight, e)
	if len(req) != 2 {
		t.Fatalf("required must stay {gitleaks,trivy}, got %v", req)
	}
	sum := store.BuildScannerCoverageSummary(store.ScanProfileLight, e, coverageRows(
		"trivy", "clean",
		// gitleaks omitted — disabled
	))
	if store.RequiredEvidenceSatisfied(sum) {
		t.Fatalf("disabled required scanner must not satisfy evidence: %+v", sum)
	}
	if !containsIncomplete(sum, "gitleaks") {
		t.Fatalf("expected gitleaks incomplete: %v", sum.RequiredIncomplete)
	}
}

func TestLightBothDisabledIncomplete(t *testing.T) {
	e := store.ProfileDefaults(store.ScanProfileLight)
	e.ScanProfile = store.ScanProfileLight
	e.EnableGitleaks = false
	e.EnableTrivy = false
	sum := store.BuildScannerCoverageSummary(store.ScanProfileLight, e, nil)
	if sum.RequiredTotal != 2 || store.RequiredEvidenceSatisfied(sum) {
		t.Fatalf("both disabled: %+v", sum)
	}
}

func TestRequiredBinaryMissingTimeoutSkipped(t *testing.T) {
	e := store.ProfileDefaults(store.ScanProfileLight)
	e.ScanProfile = store.ScanProfileLight
	cases := []struct {
		status string
	}{
		{"binary_missing"},
		{"timed_out"},
		{"disabled"},
		{"scanner_unavailable"},
		{"failed"},
	}
	for _, tc := range cases {
		sum := store.BuildScannerCoverageSummary(store.ScanProfileLight, e, coverageRows(
			"gitleaks", tc.status,
			"trivy", "clean",
		))
		if store.RequiredEvidenceSatisfied(sum) {
			t.Fatalf("status %s must block POLICY_MET evidence", tc.status)
		}
	}
}

func TestRequiredNotApplicableCountsComplete(t *testing.T) {
	e := store.ProfileDefaults(store.ScanProfileLight)
	e.ScanProfile = store.ScanProfileLight
	sum := store.BuildScannerCoverageSummary(store.ScanProfileLight, e, coverageRows(
		"gitleaks", "clean",
		"trivy", "no_supported_manifest",
	))
	if !store.RequiredEvidenceSatisfied(sum) {
		t.Fatalf("legitimate NOT_APPLICABLE should complete: %+v", sum)
	}
}

func TestOptionalDisabledOK(t *testing.T) {
	e := store.ProfileDefaults(store.ScanProfileStandard)
	e.ScanProfile = store.ScanProfileStandard
	e.EnableHadolint = false
	req := store.RequiredScannersForProfile(store.ScanProfileStandard, e)
	for _, name := range req {
		if name == "hadolint" {
			t.Fatal("disabled optional must not enter required via enable list alone when not in declared min — wait, standard unions enabled; hadolint disabled so not in required")
		}
	}
}

func TestAllRequiredHealthySatisfies(t *testing.T) {
	e := store.ProfileDefaults(store.ScanProfileLight)
	e.ScanProfile = store.ScanProfileLight
	sum := store.BuildScannerCoverageSummary(store.ScanProfileLight, e, coverageRows(
		"gitleaks", "clean",
		"trivy", "found",
	))
	if !store.RequiredEvidenceSatisfied(sum) {
		t.Fatalf("healthy required should satisfy: %+v", sum)
	}
	ratio := store.FormatCoverageRatio(sum)
	if !strings.Contains(ratio, "2/2") {
		t.Fatalf("ratio=%q", ratio)
	}
}

func TestCustomEmptyRequiredNeverSatisfied(t *testing.T) {
	e := store.DefaultGlobalSettings()
	eff := store.EffectiveFromGlobalSnapshot(e)
	eff.ScanProfile = store.ScanProfileCustom
	eff.EnableTrivy = false
	eff.EnableGrype = false
	eff.EnableGitleaks = false
	eff.EnableSemgrep = false
	eff.EnableLinters = false
	eff.EnableGovulncheck = false
	eff.EnableGosec = false
	eff.EnableStaticcheck = false
	eff.EnableHadolint = false
	eff.EnableCheckov = false
	sum := store.BuildScannerCoverageSummary(store.ScanProfileCustom, eff, nil)
	if sum.RequiredTotal != 0 {
		t.Fatalf("custom with nothing enabled: required=%d", sum.RequiredTotal)
	}
	if store.RequiredEvidenceSatisfied(sum) {
		t.Fatal("0/0 must not satisfy evidence for custom")
	}
}

func TestSkippedByPolicyBlocksRequired(t *testing.T) {
	if !store.CoverageStateBlocksPolicyMet(store.ScannerCoverageSkippedByPolicy) {
		t.Fatal("SKIPPED_BY_POLICY must block required completion")
	}
}

func containsIncomplete(sum store.ScannerCoverageSummary, name string) bool {
	for _, item := range sum.RequiredIncomplete {
		if strings.HasPrefix(item, name) {
			return true
		}
	}
	return false
}
