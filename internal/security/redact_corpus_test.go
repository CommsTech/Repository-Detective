package security_test

import (
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/internal/security"
)

// Fake-only corpus for RD-034 regression. Values are synthetic and not real credentials.
func TestSanitizeDiagnosticCorpus(t *testing.T) {
	fixtures := []struct {
		name string
		raw  string
		leak string
	}{
		{
			name: "github_pat",
			raw:  "clone failed token=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 leaked",
			leak: "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
		},
		{
			name: "aws_access_key",
			raw:  "aws_access_key_id=AKIAIOSFODNN7EXAMPLE region=us-east-1",
			leak: "AKIAIOSFODNN7EXAMPLE",
		},
		{
			name: "slack_bot",
			raw:  "gitleaks: xoxb-REDACTED-TEST-FIXTURE-NOT-A-REAL-TOKEN",
			leak: "xoxb-REDACTED-TEST-FIXTURE-NOT-A-REAL-TOKEN",
		},
		{
			name: "bearer",
			raw:  "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.aaa.bbb",
			leak: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.aaa.bbb",
		},
		{
			name: "password_in_url",
			raw:  "fatal: https://deploy:SuperSecretPass99@git.example.com/org/repo.git",
			leak: "SuperSecretPass99",
		},
		{
			name: "webhook_secret",
			raw:  `webhook_secret="whsec_test_not_real_1234567890"`,
			leak: "whsec_test_not_real_1234567890",
		},
		{
			name: "ai_api_key",
			raw:  "OPENAI_API_KEY=sk-abcdefghijklmnopqrstuvwxyz0123456789ABCD",
			leak: "sk-abcdefghijklmnopqrstuvwxyz0123456789ABCD",
		},
		{
			name: "rd_env_token",
			raw:  "REPOSITORY_DETECTIVE_GITEA_TOKEN=gitea_token_not_real_abcdef",
			leak: "gitea_token_not_real_abcdef",
		},
	}

	for _, tc := range fixtures {
		t.Run(tc.name, func(t *testing.T) {
			out := security.SanitizeDiagnostic(tc.raw, 2000)
			if strings.Contains(out, tc.leak) {
				t.Fatalf("leak %q still present in %q", tc.leak, out)
			}
			if !strings.Contains(out, "[REDACTED]") {
				t.Fatalf("expected [REDACTED] marker in %q", out)
			}
		})
	}
}

func TestMinimizeSensitivePathKeepsOperationalTail(t *testing.T) {
	unix := security.MinimizeSensitivePath("/home/alice/projects/repo/main.go")
	if strings.Contains(unix, "alice") {
		t.Fatalf("username leaked: %q", unix)
	}
	if !strings.Contains(unix, "projects/repo/main.go") {
		t.Fatalf("path tail lost: %q", unix)
	}
	win := security.MinimizeSensitivePath(`C:\Users\Bob\src\app\main.go`)
	if strings.Contains(win, "Bob") {
		t.Fatalf("windows username leaked: %q", win)
	}
}

func TestSanitizeDiagnosticTruncates(t *testing.T) {
	raw := strings.Repeat("a", 5000)
	out := security.SanitizeDiagnostic(raw, 100)
	if len(out) > 120 {
		t.Fatalf("expected truncation, got len=%d", len(out))
	}
	if !strings.Contains(out, "[truncated]") {
		t.Fatalf("missing truncated marker: %q", out)
	}
}
