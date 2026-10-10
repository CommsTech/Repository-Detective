package middleware_test

import (
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/internal/security"
)

func TestRedactAccessLogLineQueryAPIKey(t *testing.T) {
	raw := `/ui/scans/abc/graph?api_key=super-secret-key-value`
	out := security.RedactAccessLogLine(raw)
	if strings.Contains(out, "super-secret") {
		t.Fatalf("api key leaked in log line: %q", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Fatalf("expected redaction marker, got %q", out)
	}
}
