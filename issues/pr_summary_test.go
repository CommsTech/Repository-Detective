package issues

import (
	"strings"
	"testing"
)

func TestRenderPRPolicySummary(t *testing.T) {
	body := RenderPRPolicySummary(PRPolicySummaryInput{
		Outcome:         "ACTION_REQUIRED",
		EnforcementMode: "warn",
		Description:     "Action required — owner policy conditions violated",
		IssueCount:      2,
		ScannerCoverage: "8/8 required analyzers completed",
		ScanID:          "scan-1",
		UIBase:          "https://rd.example.com",
	})
	for _, part := range []string{
		"repository-detective-policy-summary",
		"ACTION_REQUIRED",
		"8/8 required analyzers completed",
		"owner-configured repository policy",
		"/ui/scans/scan-1",
	} {
		if !strings.Contains(body, part) {
			t.Fatalf("missing %q in body:\n%s", part, body)
		}
	}
}
