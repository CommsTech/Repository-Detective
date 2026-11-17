package scanners_test

import (
	"strings"
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

func TestParseShellcheckFlatArrayFormat(t *testing.T) {
	raw := []byte(`[{"file":"/tmp/x.sh","line":1,"column":1,"level":"warning","code":2155,"message":"Declare and assign separately"}]`)
	findings, err := scanners.ParseShellcheckOutputForTest(raw, "/tmp", scanners.Config{})
	if err != nil {
		t.Fatalf("parse flat shellcheck: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Reference != "SC2155" {
		t.Fatalf("reference = %q", findings[0].Reference)
	}
}

func TestParseShellcheckNestedArrayFormat(t *testing.T) {
	raw := []byte(`[[{"file":"/tmp/x.sh","line":2,"column":1,"level":"error","code":2086,"message":"Double quote"}]]`)
	findings, err := scanners.ParseShellcheckOutputForTest(raw, "/tmp", scanners.Config{})
	if err != nil {
		t.Fatalf("parse nested shellcheck: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
}

func TestParseStaticcheckCompileError(t *testing.T) {
	raw := []byte(`{"code":"compile","severity":"error","location":{"file":"/tmp/x.go","line":1},"message":"module requires at least go1.25.0, but Staticcheck was built with go1.23.12"}`)
	_, err := scanners.ParseStaticcheckOutputForTest(raw, "/tmp", scanners.Config{})
	if err == nil {
		t.Fatal("expected compile error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "requires at least go") {
		t.Fatalf("unexpected error: %v", err)
	}
}
