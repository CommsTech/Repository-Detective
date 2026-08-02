package scanners_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/scanners"
)

func TestParseHadolintToleratesStderrNoise(t *testing.T) {
	raw := []byte("WARN progress\n[{\"code\":\"DL3008\",\"message\":\"Pin versions\",\"line\":1,\"file\":\"Dockerfile\",\"level\":\"warning\"}]\n")
	parsed, err := scanners.ParseHadolintOutputForTest(raw, t.TempDir(), scanners.Config{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(parsed.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(parsed.Findings))
	}
}

func TestParseStaticcheckToleratesStderrNoise(t *testing.T) {
	raw := []byte("staticcheck: loading...\n{\"code\":\"SA4006\",\"message\":\"unused\",\"location\":{\"file\":\"/tmp/x.go\",\"line\":3,\"column\":2}}\n")
	parsed, err := scanners.ParseStaticcheckOutputForTest(raw, "/tmp", scanners.Config{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(parsed.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(parsed.Findings))
	}
}

func TestParseTrivyToleratesStderrNoise(t *testing.T) {
	raw := []byte("WARN db\n{\"Results\":[]}\n")
	findings, err := scanners.ParseTrivyOutputForTest(raw, t.TempDir())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(findings))
	}
}
