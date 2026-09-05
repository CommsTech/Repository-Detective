package closure_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/closure"
)

func TestScannerForSourceHealthFamily(t *testing.T) {
	cases := map[string]string{
		"reliability":       "health",
		"tech_debt":         "health",
		"maintainability":   "health",
		"test_gap":          "health",
		"performance":       "health",
		"ai_generated_risk": "health",
	}
	for source, want := range cases {
		if got := closure.ScannerForSource(source); got != want {
			t.Fatalf("ScannerForSource(%q) = %q, want %q", source, got, want)
		}
	}
}

func TestScannerForSourceStaticAndExternal(t *testing.T) {
	cases := map[string]string{
		"static":        "static",
		"hadolint":      "hadolint",
		"checkov":       "checkov",
		"semgrep":       "semgrep",
		"golangci-lint": "staticcheck",
	}
	for source, want := range cases {
		if got := closure.ScannerForSource(source); got != want {
			t.Fatalf("ScannerForSource(%q) = %q, want %q", source, got, want)
		}
	}
}
