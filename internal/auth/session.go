package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// NewSessionID returns a cryptographically random session identifier.
func NewSessionID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// SignSessionCookie produces a signed cookie value: sessionID.signature.
func SignSessionCookie(secret, sessionID string) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", fmt.Errorf("session secret is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return "", fmt.Errorf("session id is required")
	}
	sig := sign(secret, sessionID)
	return sessionID + "." + sig, nil
}

// ParseSessionCookie verifies the signature and returns the session ID.
func ParseSessionCookie(secret, cookieValue string) (string, bool) {
	if secret == "" || cookieValue == "" {
		return "", false
	}
	parts := strings.SplitN(cookieValue, ".", 2)
	if len(parts) != 2 {
		return "", false
	}
	sessionID, sig := parts[0], parts[1]
	if sessionID == "" || sig == "" {
		return "", false
	}
	expected := sign(secret, sessionID)
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return "", false
	}
	return sessionID, true
}

func sign(secret, sessionID string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write([]byte("rd-session-v1:")); err != nil {
		return ""
	}
	if _, err := mac.Write([]byte(sessionID)); err != nil {
		return ""
	}
	return hex.EncodeToString(mac.Sum(nil))
}

// SessionExpiresAt returns the expiry time for a new session.
func SessionExpiresAt(ttlHours int) time.Time {
	if ttlHours <= 0 {
		ttlHours = 12
	}
	return time.Now().UTC().Add(time.Duration(ttlHours) * time.Hour)
}
