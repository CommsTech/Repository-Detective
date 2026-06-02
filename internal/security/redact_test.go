package security

import "testing"

func TestRedactSecrets(t *testing.T) {
	raw := `api_key="supersecret12345" and AKIAIOSFODNN7EXAMPLE`
	out := RedactSecrets(raw)
	if out == raw {
		t.Fatalf("expected redaction, got %q", out)
	}
	if contains(out, "supersecret") || contains(out, "AKIAIOSFODNN7EXAMPLE") {
		t.Fatalf("secret leaked: %q", out)
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
