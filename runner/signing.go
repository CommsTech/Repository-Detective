package runner

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SignRequest builds an HMAC-SHA256 signature for runner HTTP requests.
func SignRequest(secret, timestamp, nonce, method, path string, body []byte) string {
	payload := strings.Join([]string{
		timestamp,
		nonce,
		strings.ToUpper(method),
		path,
		string(body),
	}, "\n")
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyRequest validates runner request headers and body signature.
func VerifyRequest(secret, timestamp, nonce, signature, method, path string, body []byte, now time.Time) error {
	if secret == "" {
		return ErrSharedSecretRequired
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64)
	if err != nil {
		return ErrExpiredTimestamp
	}
	reqTime := time.Unix(ts, 0)
	skew := now.Sub(reqTime)
	if skew < 0 {
		skew = -skew
	}
	if skew > time.Duration(MaxClockSkewSeconds)*time.Second {
		return ErrExpiredTimestamp
	}
	if strings.TrimSpace(nonce) == "" {
		return fmt.Errorf("missing runner nonce")
	}
	expected := SignRequest(secret, timestamp, nonce, method, path, body)
	if !hmac.Equal([]byte(expected), []byte(strings.TrimSpace(signature))) {
		return ErrInvalidSignature
	}
	return nil
}

// SignBody signs an arbitrary JSON payload envelope for runner responses.
func SignBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyBodySignature checks a detached body signature.
func VerifyBodySignature(secret string, body []byte, signature string) error {
	if secret == "" {
		return ErrSharedSecretRequired
	}
	expected := SignBody(secret, body)
	if !hmac.Equal([]byte(expected), []byte(strings.TrimSpace(signature))) {
		return ErrInvalidSignature
	}
	return nil
}
