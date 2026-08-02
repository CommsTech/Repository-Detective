package openclaw_test

import (
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/openclaw"
)

func TestSecretValueRedacted(t *testing.T) {
	out, _ := openclaw.RedactText("password=supersecret123", true)
	if strings.Contains(out, "supersecret") {
		t.Fatalf("not redacted: %q", out)
	}
}

func TestTokenInURLRedacted(t *testing.T) {
	out, _ := openclaw.RedactText("https://oauth2:abc123@git.example.com/repo.git", true)
	if strings.Contains(out, "abc123") {
		t.Fatalf("token not redacted: %q", out)
	}
}

func TestPrivateKeyRedacted(t *testing.T) {
	// Construct PEM-shaped sample at runtime to avoid static secret scanner hits on source.
	key := "-----BEGIN " + "PRIVATE KEY-----\nMIIE\n-----END " + "PRIVATE KEY-----"
	out, _ := openclaw.RedactText(key, true)
	if strings.Contains(out, "BEGIN PRIVATE") {
		t.Fatalf("private key not redacted: %q", out)
	}
}

func TestAWSKeyRedactedInPacket(t *testing.T) {
	awsSample := "AKI" + "AIOSFODNN7EXAMPLE"
	pkt := &openclaw.ReviewPacket{
		Findings: []openclaw.FindingInput{{
			Fingerprint: "fp1", EvidenceRedacted: "key " + awsSample + " in config",
		}},
	}
	cfg := openclaw.DefaultConfig()
	n, err := openclaw.RedactPacket(pkt, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("expected redaction count > 0")
	}
	if strings.Contains(pkt.Findings[0].EvidenceRedacted, "AKIA") {
		t.Fatalf("aws key not redacted: %q", pkt.Findings[0].EvidenceRedacted)
	}
}

func TestLongEvidenceTruncated(t *testing.T) {
	long := strings.Repeat("x", 5000)
	out, _ := openclaw.RedactText(long, true)
	if len(out) > 2100 {
		t.Fatalf("expected truncation, len=%d", len(out))
	}
}
