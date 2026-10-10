package operator

import "testing"

func TestPickVersionLinePrefersStdoutAndSkipsUpgradeNoise(t *testing.T) {
	got := pickVersionLine("1.76.0\n", "\nA new version of Semgrep is available. See https://semgrep.dev/docs/upgrading\n")
	if got != "1.76.0" {
		t.Fatalf("got %q", got)
	}
	got = pickVersionLine("Go: go1.23.12\nScanner: govulncheck@v1.1.3\n", "")
	if got != "Scanner: govulncheck@v1.1.3" {
		t.Fatalf("got %q", got)
	}
}
