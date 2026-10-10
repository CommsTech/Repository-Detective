package security

import (
	"strings"
	"testing"
)

// assemble joins parts at runtime so the source file never contains a contiguous
// secret-shaped literal that GitGuardian / GitHub secret scanning will flag.
func assemble(parts ...string) string { return strings.Join(parts, "") }

func TestRedactSecrets(t *testing.T) {
	secret := assemble("super", "secret", "12345")
	aws := assemble("AKIA", "IOSFODNN7EXAMPLE")
	raw := assemble(`api_key="`, secret, `" and `, aws)
	out := RedactSecrets(raw)
	if out == raw {
		t.Fatalf("expected redaction, got %q", out)
	}
	if contains(out, secret) || contains(out, aws) {
		t.Fatalf("secret leaked: %q", out)
	}
}

func TestRedactAccessLogQueryAPIKey(t *testing.T) {
	leak := assemble("should-not-", "appear-in-", "logs")
	raw := "/ui/scans/abc?api_key=" + leak
	out := RedactAccessLogLine(raw)
	if contains(out, leak) || contains(out, "should-not-appear") {
		t.Fatalf("query api_key leaked: %q", out)
	}
}

func TestRedactBearerToken(t *testing.T) {
	header := assemble("eyJhbGciOi", "JIUzI1NiIsInR5cCI6", "IkpXVCJ9")
	raw := "Authorization: Bearer " + header + ".aaa.bbb"
	out := RedactSecrets(raw)
	if contains(out, header) {
		t.Fatalf("bearer token leaked: %q", out)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
