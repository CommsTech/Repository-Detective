package store

import (
	"testing"

	"git.commsnet.org/commstech/bugbot/operator"
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
	for _, r := range merged.Rollups {
		switch r.Name {
		case "hadolint":
			if r.StatusLabel != "disabled, not required" {
				t.Fatalf("hadolint: %q", r.StatusLabel)
			}
		case "trivy":
			if r.AffectedRepos != 12 || r.RecommendedFix == "" {
				t.Fatalf("trivy rollup: repos=%d fix=%q", r.AffectedRepos, r.RecommendedFix)
			}
		}
	}
}
