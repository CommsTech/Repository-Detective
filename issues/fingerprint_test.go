package issues

import (
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/ai"
)

func TestComputeFingerprintStable(t *testing.T) {
	in := FingerprintInput{
		Repository: "owner/repo",
		Category:   "secret",
		Source:     "gitleaks",
		RuleID:     "generic-api-key",
		File:       "config/env.py",
		Line:       12,
	}
	fp1 := ComputeFingerprint(in)
	fp2 := ComputeFingerprint(in)
	if fp1 != fp2 {
		t.Fatalf("expected stable fingerprint, got %q vs %q", fp1, fp2)
	}
	if !strings.HasPrefix(fp1, "rd-") {
		t.Fatalf("expected rd- prefix, got %q", fp1)
	}
}

func TestComputeFingerprintDifferentRuleSameLocation(t *testing.T) {
	base := FingerprintInput{
		Repository: "owner/repo",
		Category:   "security",
		Source:     "semgrep",
		File:       "main.go",
		Line:       42,
	}
	a := ComputeFingerprint(base)
	base.RuleID = "rule-a"
	b := ComputeFingerprint(base)
	base.RuleID = "rule-b"
	c := ComputeFingerprint(base)
	if a == b || a == c || b == c {
		t.Fatal("different rules at same location should not share fingerprint")
	}
}

func TestSanitizeSecretEvidence(t *testing.T) {
	raw := `api_key="AKIAIOSFODNN7EXAMPLE"`
	sanitized := SanitizeSecretEvidence(raw)
	if strings.Contains(sanitized, "AKIAIOSFODNN7EXAMPLE") {
		t.Fatalf("raw secret leaked into sanitized evidence: %q", sanitized)
	}
}

func TestExtractFingerprintFromBody(t *testing.T) {
	body := "## Tracking\n\n- Repository Detective fingerprint: rd-deadbeef\n"
	if got := ExtractFingerprintFromBody(body); got != "rd-deadbeef" {
		t.Fatalf("marker parse: got %q", got)
	}
	if got := ExtractFingerprintFromBody("no marker here"); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestEnrichIssueSetsFingerprintAndLabelsMetadata(t *testing.T) {
	issue := &ai.CodeIssue{
		Title:       "Semgrep finding: rule-x",
		Severity:    "high",
		Category:    "sast",
		Source:      "semgrep",
		RuleID:      "rule-x",
		File:        "src/main.go",
		LineNumber:  10,
		Confidence:  0.9,
		CodeSnippet: "eval(userInput)",
	}
	EnrichIssue("owner/repo", issue, "scan-123")
	if issue.Fingerprint == "" {
		t.Fatal("expected fingerprint")
	}
	if !strings.HasPrefix(issue.Fingerprint, "rd-") {
		t.Fatalf("expected rd- prefix, got %q", issue.Fingerprint)
	}
	if issue.Category != CategorySecurity {
		t.Fatalf("expected normalized category security, got %q", issue.Category)
	}
	if issue.Fixable == "" || issue.RegressionRisk == "" {
		t.Fatal("expected remediation metadata")
	}
}

func TestNormalizeCategoryMappings(t *testing.T) {
	cases := map[string]string{
		"hardcoded_secret":          CategorySecret,
		"dependency_vulnerability":  CategoryDependency,
		"sql_injection":             CategorySecurity,
		"quality":                   CategoryCodeQuality,
		"lint":                      CategoryMaintainability,
	}
	for input, want := range cases {
		source := ""
		if input == "lint" {
			source = "golangci-lint"
		}
		if got := NormalizeCategory(input, source); got != want {
			t.Fatalf("%s: got %q want %q", input, got, want)
		}
	}
}

func TestAIGeneratedRiskWording(t *testing.T) {
	if !strings.Contains(AIGeneratedRiskWording(), "Possible AI-generated") {
		t.Fatal("wording should not overclaim AI authorship")
	}
}
