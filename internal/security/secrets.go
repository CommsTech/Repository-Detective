package security

import (
	"fmt"
	"strings"
)

// Published example secrets and forge placeholders that must never be used in a
// running deployment. Copied from .env.example / config.env.template so a
// "cp .env.example .env && docker compose up" path cannot expose a privileged
// instance with predictable credentials.
var publishedSecretPlaceholders = []string{
	"change-me-to-a-secure-random-string",
	"replace-with-openssl-rand-hex-32",
	"your-gitea-token",
	"your-gitea-access-token-here",
	"your-webhook-secret",
	"your-github-token",
	"generate-a-long-random-string",
	"another-random-string",
}

var publishedURLPlaceholders = []string{
	"https://git.example.com",
	"http://git.example.com",
	"https://detective.example.com",
	"http://detective.example.com",
	"https://repository-detective.example.com",
	"http://repository-detective.example.com",
}

// IsPublishedSecretPlaceholder reports whether value matches a shipped example secret.
func IsPublishedSecretPlaceholder(value string) bool {
	v := strings.TrimSpace(strings.ToLower(value))
	if v == "" {
		return false
	}
	for _, p := range publishedSecretPlaceholders {
		if v == strings.ToLower(p) {
			return true
		}
	}
	return false
}

// IsPublishedURLPlaceholder reports whether value is a documented example URL.
func IsPublishedURLPlaceholder(value string) bool {
	v := strings.TrimSpace(strings.ToLower(strings.TrimRight(value, "/")))
	if v == "" {
		return false
	}
	for _, p := range publishedURLPlaceholders {
		if v == strings.ToLower(strings.TrimRight(p, "/")) {
			return true
		}
	}
	return false
}

// ValidateInstallSecrets rejects published example credentials and URLs so a
// fresh copy of .env.example cannot start as an externally usable instance.
func ValidateInstallSecrets(apiKey, sessionSecret, webhookSecret, giteaToken, giteaURL, publicURL string, authModeLocal bool) error {
	checks := []struct {
		name  string
		value string
		kind  string
	}{
		{"api_key", apiKey, "secret"},
		{"webhook_secret", webhookSecret, "secret"},
		{"gitea_token", giteaToken, "secret"},
		{"gitea_url", giteaURL, "url"},
		{"public_url", publicURL, "url"},
	}
	if authModeLocal {
		checks = append(checks, struct {
			name  string
			value string
			kind  string
		}{"session_secret", sessionSecret, "secret"})
	}
	for _, c := range checks {
		switch c.kind {
		case "secret":
			if IsPublishedSecretPlaceholder(c.value) {
				return fmt.Errorf("%s uses a published example placeholder — generate a random value before starting (see .env.example comments)", c.name)
			}
		case "url":
			if IsPublishedURLPlaceholder(c.value) {
				return fmt.Errorf("%s uses a published example hostname — set your real URL before starting", c.name)
			}
		}
	}
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("api_key is required (generate with: openssl rand -hex 32)")
	}
	if len(strings.TrimSpace(apiKey)) < 16 {
		return fmt.Errorf("api_key is too short (use at least 16 random characters)")
	}
	if authModeLocal {
		sec := strings.TrimSpace(sessionSecret)
		if sec == "" {
			return fmt.Errorf("auth_mode=local requires session_secret (set REPOSITORY_DETECTIVE_SESSION_SECRET)")
		}
		if len(sec) < 32 {
			return fmt.Errorf("session_secret is too short (use at least 32 random characters; openssl rand -hex 32)")
		}
	}
	return nil
}
