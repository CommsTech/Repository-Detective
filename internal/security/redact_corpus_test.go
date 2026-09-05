package security_test

import (
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/internal/security"
)

// assemble builds synthetic fixtures from parts so the source file never contains a
// contiguous secret-shaped string that trip public-mirror push protection.
func assemble(parts ...string) string { return strings.Join(parts, "") }

// Fake-only corpus for RD-034 regression. Values are synthetic and not real credentials.
func TestSanitizeDiagnosticCorpus(t *testing.T) {
	slack := assemble("xoxb-", "123456789012-", "123456789012-", "abcdefghijklmnopqrstuvwx")
	ghpat := assemble("ghp_", "ABCDEFGHIJKLMNOPQRSTUVWXYZ", "0123456789")
	aws := assemble("AKIA", "IOSFODNN7EXAMPLE")
	bearer := assemble("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", ".", "aaa", ".", "bbb")
	openai := assemble("sk-", "abcdefghijklmnopqrstuvwxyz", "0123456789ABCD")

	fixtures := []struct {
		name string
		raw  string
		leak string
	}{
		{
			name: "github_pat",
			raw:  "clone failed token=" + ghpat + " leaked",
			leak: ghpat,
		},
		{
			name: "aws_access_key",
			raw:  "aws_access_key_id=" + aws + " region=us-east-1",
			leak: aws,
		},
		{
			name: "slack_bot",
			raw:  "gitleaks: " + slack,
			leak: slack,
		},
		{
			name: "bearer",
			raw:  "Authorization: Bearer " + bearer,
			leak: bearer,
		},
		{
			name: "password_in_url",
			raw:  "fatal: https://deploy:" + assemble("Super", "Secret", "Pass99") + "@git.example.com/org/repo.git",
			leak: assemble("Super", "Secret", "Pass99"),
		},
		{
			name: "webhook_secret",
			raw:  `webhook_secret="` + assemble("whsec_", "test_not_real_", "1234567890") + `"`,
			leak: assemble("whsec_", "test_not_real_", "1234567890"),
		},
		{
			name: "ai_api_key",
			raw:  "OPENAI_API_KEY=" + openai,
			leak: openai,
		},
		{
			name: "rd_env_token",
			raw:  "REPOSITORY_DETECTIVE_GITEA_TOKEN=" + assemble("gitea_token_", "not_real_", "abcdef"),
			leak: assemble("gitea_token_", "not_real_", "abcdef"),
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
