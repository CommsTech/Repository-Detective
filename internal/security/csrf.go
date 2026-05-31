package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// CSRFToken derives a stable CSRF token from the operator API key secret and client key material.
// This is a homelab-friendly mitigation when UI forms authenticate via query-string API keys.
func CSRFToken(apiSecret, clientKey string) string {
	if apiSecret == "" || clientKey == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(apiSecret))
	_, _ = mac.Write([]byte("bugbot-csrf-v1:"))
	_, _ = mac.Write([]byte(clientKey))
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
