package security

import "testing"

func TestSessionCSRFToken(t *testing.T) {
	secret := "csrf-secret"
	token := SessionCSRFToken(secret, "sess-1", 42)
	if token == "" {
		t.Fatal("expected token")
	}
	if !ValidSessionCSRFToken(secret, "sess-1", 42, token) {
		t.Fatal("expected valid session csrf")
	}
	if ValidSessionCSRFToken(secret, "sess-1", 42, "bad") {
		t.Fatal("expected invalid token to fail")
	}
	if ValidSessionCSRFToken(secret, "sess-2", 42, token) {
		t.Fatal("expected different session to fail")
	}
}
