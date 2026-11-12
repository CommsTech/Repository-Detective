package store_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/learning"
	"git.commsnet.org/commstech/repository-detective/store"
)

// TestProtectedRuleIDForLearningConformance pins the hand-maintained store-side
// mirror (protectedRuleIDForLearning) to the authoritative
// learning.IsProtectedRuleID across a representative matrix, AND pins each
// expected verdict itself. Conformance alone cannot catch two sides drifting
// identically; the explicit expected column catches a coordinated mistake in
// either security gate.
func TestProtectedRuleIDForLearningConformance(t *testing.T) {
	cases := []struct {
		source   string
		ruleID   string
		expected bool // the verdict both implementations must agree on — and we require
		comment  string
	}{
		// Protected sources: true.
		{"gitleaks", "generic-api-key", true, "gitleaks source protected"},
		{"Gitleaks-History", "historical-secret", true, "case-insensitive source match"},
		{"trivy", "CVE-2026-1234", true, "trivy source protected"},
		{"grype", "GHSA-xxxx-yyyy", true, "grype source protected"},
		{"govulncheck", "GO-2026-0001", true, "govulncheck source protected"},
		{"semgrep", "html.security.plaintext-http-link", true, "semgrep source protected"},

		// gosec: informational tooling allowed through, everything else protected.
		{"gosec", "G104", false, "gosec informational allowed"},
		{"gosec", "G304", false, "gosec informational allowed"},
		{"gosec", "G204", false, "gosec informational allowed"},
		{"gosec", "G401", true, "gosec crypto weakness protected"},
		{"gosec", "G501", true, "gosec blocked import protected"},

		// checkov: secret rules protected, other rules not.
		{"checkov", "CKV_SECRET_6", true, "checkov secret prefix protected"},
		{"checkov", "generic-SECRET-rule", true, "checkov secret substring protected"},
		{"checkov", "CKV_DOCKER_2", false, "checkov non-secret not protected"},

		// staticcheck unused code.
		{"staticcheck", "U1000", true, "staticcheck unused protected"},
		{"staticcheck", "U1001", true, "staticcheck family protected"},
		{"go vet", "U1000", true, "rule prefix U1 protected regardless of source"},

		// Rule-ID prefixes: true.
		{"static", "CVE-2026-5678", true, "CVE prefix protected"},
		{"static", "GHSA-abcd-efgh-ijkl", true, "GHSA prefix protected"},
		{"static", "TRIVY-001", true, "trivy rule prefix protected"},
		{"static", "GRYPE-002", true, "grype rule prefix protected"},
		{"static", "GITLEAKS-aws-key", true, "gitleaks rule prefix protected"},
		{"static", "SEC-HARDCODED-SECRET", true, "SEC prefix protected"},
		{"static", "G201", true, "sql injection concat protected"},
		{"static", "G202", true, "sql injection tautology protected"},
		{"static", "G703", true, "path traversal protected"},

		// Rule-ID markers: true.
		{"static", "LEAKED-PASSWORD", true, "password marker protected"},
		{"static", "API-TOKEN-IN-URL", true, "token marker protected"},
		{"static", "CLIENT-CREDENTIAL", true, "credential marker protected"},
		{"static", "RSA-PRIVATE-KEY", true, "private-key marker protected (hyphen)"},
		{"static", "PRIVATE_KEY_LINE", true, "private-key marker protected (underscore)"},

		// Unprotected: false.
		{"static", "OPT-NESTED-LOOP", false, "performance rule not protected"},
		{"static", "QUAL-DEBUG", false, "quality rule not protected"},
		{"static", "HEALTH-COMMENT-BLOCK", false, "health rule not protected"},
		{"tech_debt", "HEALTH-TECH-MARKER", false, "tech marker not protected"},
		{"", "", false, "empty input not protected"},
	}

	for _, tc := range cases {
		storeVerdict := store.ProtectedRuleIDForLearning(tc.source, tc.ruleID)
		learningVerdict := learning.IsProtectedRuleID(tc.source, tc.ruleID)
		if storeVerdict != learningVerdict {
			t.Errorf("divergence for (%q, %q) [%s]: store=%v learning=%v",
				tc.source, tc.ruleID, tc.comment, storeVerdict, learningVerdict)
		}
		if storeVerdict != tc.expected {
			t.Errorf("store verdict for (%q, %q) [%s]: got %v, want %v",
				tc.source, tc.ruleID, tc.comment, storeVerdict, tc.expected)
		}
	}
}
