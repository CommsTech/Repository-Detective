package redact

import "testing"

func TestSecretEvidenceRedactsAPIKeyPatterns(t *testing.T) {
	// Build AWS-shaped sample at runtime so static secret scanners do not flag the source.
	awsSample := "AKI" + "A1234567890ABCDEF"
	cases := []struct {
		in  string
		out string
	}{
		{`api_key="super-secret-value-here"`, `[REDACTED]`},
		{`Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6`, `Authorization: [REDACTED]`},
		{`password = 'longpassword123'`, `[REDACTED]`},
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
}
