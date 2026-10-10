package auth

import "testing"

func TestSessionCookieSignAndParse(t *testing.T) {
	secret := "test-session-secret-value"
	id, err := NewSessionID()
	if err != nil {
		t.Fatalf("new session id: %v", err)
	}
	signed, err := SignSessionCookie(secret, id)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	parsed, ok := ParseSessionCookie(secret, signed)
	if !ok || parsed != id {
		t.Fatalf("parse failed: ok=%v parsed=%q want=%q", ok, parsed, id)
	}
	if _, ok := ParseSessionCookie(secret, id+".tampered"); ok {
		t.Fatal("expected tampered cookie to fail")
	}
}
