package store

import (
	"testing"

	"git.commsnet.org/commstech/bugbot/operator"
)

func TestMergeScannerRollupsDegradedCoverage(t *testing.T) {
	tools := []operator.ToolStatus{
		{Name: "git", Configured: true, Available: true, Version: "git version 2.45.4", LastChecked: "t"},
		{Name: "trivy", Configured: true, Available: false, LastChecked: "t"},
		{Name: "hadolint", Configured: false, Available: false, LastChecked: "t"},
	}
	summary := MergeScannerRollups(nil, tools)
	if summary.ConfiguredMissingRuntime != 1 {
		t.Fatalf("expected 1 configured missing runtime, got %d", summary.ConfiguredMissingRuntime)
	}
	if !summary.DegradedCoverage {
		t.Fatal("expected degraded coverage flag")
	}
	var trivy *ScannerPlatformRollup
	for i := range summary.Rollups {
		if summary.Rollups[i].Name == "trivy" {
			trivy = &summary.Rollups[i]
			break
		}
	}
	if trivy == nil || trivy.CoverageImpact != "degraded" {
		t.Fatal("expected trivy degraded impact")
	}
	if trivy.Optional {
		t.Fatal("trivy should not be optional when configured")
	}
}
