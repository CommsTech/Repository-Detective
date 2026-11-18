package security

import "testing"

func TestIsPublishedSecretPlaceholder(t *testing.T) {
	if !IsPublishedSecretPlaceholder("change-me-to-a-secure-random-string") {
		t.Fatal("expected published API key placeholder")
	}
	if !IsPublishedSecretPlaceholder("replace-with-openssl-rand-hex-32") {
		t.Fatal("expected published session secret placeholder")
	}
	if IsPublishedSecretPlaceholder("a-real-random-operator-secret-value") {
		t.Fatal("real secret must not match placeholder list")
	}
}

func TestValidateInstallSecretsRejectsPlaceholders(t *testing.T) {
	err := ValidateInstallSecrets(
		"change-me-to-a-secure-random-string",
		"replace-with-openssl-rand-hex-32",
		"your-webhook-secret",
		"your-gitea-token",
		"https://git.example.com",
		"https://detective.example.com",
		true,
	)
	if err == nil {
		t.Fatal("expected placeholder rejection")
	}
}

func TestValidateInstallSecretsAcceptsStrongValues(t *testing.T) {
	err := ValidateInstallSecrets(
		"0123456789abcdef0123456789abcdef",
		"0123456789abcdef0123456789abcdef",
		"webhook-secret-that-is-unique",
		"gitea-pat-that-is-unique-enough",
		"https://git.commsnet.org",
		"http://127.0.0.1:8081",
		true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
