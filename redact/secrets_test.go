package redact

import (
	"strings"
	"testing"
)

func assemble(parts ...string) string { return strings.Join(parts, "") }

func TestSecretEvidenceRedactsAPIKeyPatterns(t *testing.T) {
	// Build secret-shaped samples at runtime so static scanners do not flag the source.
	awsSample := "AKI" + "A1234567890ABCDEF"
	apiVal := assemble("super-secret-", "value-here")
	bearer := assemble("eyJhbGciOi", "JIUzI1NiIsInR5cCI6")
	pass := assemble("long", "password", "123")
	cases := []struct {
		in  string
		out string
	}{
		{`api_key="` + apiVal + `"`, `[REDACTED]`},
		{"Authorization: Bearer " + bearer, `Authorization: [REDACTED]`},
		{"password = '" + pass + "'", `[REDACTED]`},
		{awsSample, `[REDACTED]`},
	}
	for _, tc := range cases {
		got := SecretEvidence(tc.in)
		if got != tc.out {
			t.Fatalf("SecretEvidence(%q) = %q, want %q", tc.in, got, tc.out)
		}
	}
}

func TestSecretEvidencePreservesSafeText(t *testing.T) {
	in := "scan completed for repository commstech/Repository-Detective"
	if got := SecretEvidence(in); got != in {
		t.Fatalf("expected unchanged safe text, got %q", got)
	}
	if strings.Contains(in, "Bearer ") {
		t.Fatal("safe text fixture must not embed bearer tokens")
	}
}
