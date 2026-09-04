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

func TestRedactAccessLogQueryAPIKey(t *testing.T) {
	raw := `/ui/scans/abc?api_key=should-not-appear-in-logs`
	out := RedactAccessLogLine(raw)
	if contains(out, "should-not-appear") {
		t.Fatalf("query api_key leaked: %q", out)
	}
}

func TestRedactBearerToken(t *testing.T) {
	raw := "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.aaa.bbb"
	out := RedactSecrets(raw)
	if contains(out, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9") {
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
