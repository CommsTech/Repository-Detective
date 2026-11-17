package scanners

import (
	"strings"
	"testing"
	"time"
)

func TestGolangciRunArgsForVersion(t *testing.T) {
	v1 := golangciRunArgsForVersion("golangci-lint has version 1.55.2 built with go1.21.3")
	if len(v1) < 4 || v1[2] != "--out-format" {
		t.Fatalf("v1 args = %#v", v1)
	}
	v2 := strings.Join(golangciRunArgsForVersion("golangci-lint has version 2.14.0 built with go1.26.0"), " ")
	if !strings.Contains(v2, "--output.json.path=stdout") {
		t.Fatalf("v2 args = %s", v2)
	}
}

func TestGolangciConfigRejected(t *testing.T) {
	if !golangciConfigRejected([]byte("Error: can't load config: unsupported version of the configuration: \"\""), nil) {
		t.Fatal("expected v1 config text to be rejected")
	}
	if golangciConfigRejected([]byte(`{"Issues":[]}`), nil) {
		t.Fatal("json report is not a config rejection")
	}
	if golangciTimeout(180) != 10*time.Minute {
		t.Fatal("short scanner cap should be raised for golangci-lint")
	}
}
