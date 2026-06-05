package redact

import "testing"

func TestSecretEvidenceRedactsAPIKeyPatterns(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{`api_key="super-secret-value-here"`, `[REDACTED]`},
		{`Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6`, `Authorization: [REDACTED]`},
		{`password = 'longpassword123'`, `[REDACTED]`},
		{`AKIA1234567890ABCDEF`, `[REDACTED]`},
	}
	for _, tc := range cases {
		got := SecretEvidence(tc.in)
		if got != tc.out {
			t.Fatalf("SecretEvidence(%q) = %q, want %q", tc.in, got, tc.out)
		}
	}
}

func TestSecretEvidencePreservesSafeText(t *testing.T) {
	in := "scan completed for repository commstech/Bugbot"
	if got := SecretEvidence(in); got != in {
		t.Fatalf("expected unchanged safe text, got %q", got)
	}
}
