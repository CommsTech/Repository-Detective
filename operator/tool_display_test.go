package operator

import "testing"

func TestToolStatusDisplay(t *testing.T) {
	cases := []struct {
		tool     ToolStatus
		install  string
		optional bool
		impact   string
		version  string
	}{
		{
			tool:    ToolStatus{Name: "git", Configured: true, Available: true, Version: "git version 2.45.4", LastChecked: "now"},
			install: "available", optional: false, impact: "none", version: "git version 2.45.4",
		},
		{
			tool:    ToolStatus{Name: "trivy", Configured: true, Available: false, LastChecked: "now"},
			install: "missing", optional: false, impact: "degraded", version: "—",
		},
		{
			tool:    ToolStatus{Name: "hadolint", Configured: false, Available: false, LastChecked: "now"},
			install: "disabled", optional: true, impact: "inactive", version: "—",
		},
		{
			tool:    ToolStatus{Name: "semgrep", Configured: true, Available: true, LastChecked: "now"},
			install: "available", optional: false, impact: "none", version: "unknown",
		},
	}
	for _, tc := range cases {
		if got := tc.tool.InstallState(); got != tc.install {
			t.Fatalf("%s InstallState: got %q want %q", tc.tool.Name, got, tc.install)
		}
		if got := tc.tool.IsOptional(); got != tc.optional {
			t.Fatalf("%s IsOptional: got %v want %v", tc.tool.Name, got, tc.optional)
		}
		if got := tc.tool.CoverageImpact(); got != tc.impact {
			t.Fatalf("%s CoverageImpact: got %q want %q", tc.tool.Name, got, tc.impact)
		}
		if got := tc.tool.VersionDisplay(); got != tc.version {
			t.Fatalf("%s VersionDisplay: got %q want %q", tc.tool.Name, got, tc.version)
		}
	}
}

func TestTrivyRemediationHint(t *testing.T) {
	tool := ToolStatus{Name: "trivy", Configured: true, Available: false, LastChecked: "now"}
	hint := tool.RemediationHint()
	if hint == "" {
		t.Fatal("expected trivy remediation hint")
	}
}
