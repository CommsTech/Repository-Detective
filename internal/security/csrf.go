package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// CSRFToken derives a stable CSRF token from the operator API key secret and client key material.
// This is a homelab-friendly mitigation when UI forms authenticate via query-string API keys.
func CSRFToken(apiSecret, clientKey string) string {
	if apiSecret == "" || clientKey == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(apiSecret))
	if _, err := mac.Write([]byte("rd-csrf-v1:")); err != nil {
		return ""
	}
	if _, err := mac.Write([]byte(clientKey)); err != nil {
		return ""
	}
	sum := mac.Sum(nil)
	return hex.EncodeToString(sum[:16])
}

// ValidCSRFToken performs constant-time comparison of CSRF tokens.
func ValidCSRFToken(apiSecret, clientKey, token string) bool {
	expected := CSRFToken(apiSecret, clientKey)
	if expected == "" || token == "" {
		return false
	}
	return hmac.Equal([]byte(token), []byte(expected))
}

// SessionCSRFToken derives a CSRF token for browser sessions in local auth mode.
// Use sessionID "bootstrap" and userID 0 for login/bootstrap forms before a session exists.
func SessionCSRFToken(sessionSecret, sessionID string, userID int64) string {
	if sessionSecret == "" || sessionID == "" || userID < 0 {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(sessionSecret))
	if _, err := mac.Write([]byte("rd-csrf-session-v1:")); err != nil {
		return ""
	}
	if _, err := mac.Write([]byte(sessionID)); err != nil {
		return ""
	}
	if _, err := mac.Write([]byte(fmt.Sprintf(":%d", userID))); err != nil {
		return ""
	}
	sum := mac.Sum(nil)
	return hex.EncodeToString(sum[:16])
}

// ValidSessionCSRFToken performs constant-time comparison of session CSRF tokens.
func ValidSessionCSRFToken(sessionSecret, sessionID string, userID int64, token string) bool {
	if userID < 0 {
		return false
	}
	expected := SessionCSRFToken(sessionSecret, sessionID, userID)
	if expected == "" || token == "" {
		return false
	}
	return hmac.Equal([]byte(token), []byte(expected))
}
