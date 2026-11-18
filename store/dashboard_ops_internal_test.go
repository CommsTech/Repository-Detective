package store

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/operator"
)

func TestMergeScannerRollupsLabels(t *testing.T) {
	db := map[string]scannerDBRollup{
		"trivy": {MissingScans: 39, MissingRepos: 12},
	}
	tools := []operator.ToolStatus{
		{Name: "trivy", Configured: true, Available: false},
		{Name: "hadolint", Configured: false, Available: false},
		{Name: "grype", Configured: true, Available: true},
	}
	merged := MergeScannerRollups(db, tools)
	if merged.UniqueMissingTools != 1 {
		t.Fatalf("unique missing = %d", merged.UniqueMissingTools)
	}
	if merged.ConfiguredMissingRuntime != 0 || merged.DegradedCoverage {
		t.Fatalf("trivy bypassed by grype: missing=%d degraded=%v", merged.ConfiguredMissingRuntime, merged.DegradedCoverage)
	}
	for _, r := range merged.Rollups {
		switch r.Name {
		case "hadolint":
			if r.StatusLabel != "optional, inactive" {
				t.Fatalf("hadolint: %q", r.StatusLabel)
			}
			if !r.Optional || r.CoverageImpact != "inactive" {
				t.Fatalf("hadolint optional/inactive: optional=%v impact=%q", r.Optional, r.CoverageImpact)
			}
		case "trivy":
			if !r.Optional || r.StatusLabel != "bypassed (grype active)" {
				t.Fatalf("trivy rollup: optional=%v label=%q", r.Optional, r.StatusLabel)
			}
			if r.RecommendedFix == "" {
				t.Fatal("trivy bypass should include recommended fix text")
			}
		}
	}
}
