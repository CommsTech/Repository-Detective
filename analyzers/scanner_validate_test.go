package analyzers

import "testing"

func TestClassifyScannerRunStatus(t *testing.T) {
	if got := ClassifyScannerRunStatus(1, "", "binary not found", false); got != "scanner_unavailable" {
		t.Fatalf("got %q", got)
	}
	if got := ClassifyScannerRunStatus(0, "not json", "", false); got != "parse_failed" {
		t.Fatalf("got %q", got)
	}
	if got := ClassifyScannerRunStatus(0, `{"results":[]}`, "", false); got != "success" {
		t.Fatalf("got %q", got)
	}
	if got := ClassifyScannerRunStatus(0, "", "", true); got != "timed_out" {
		t.Fatalf("got %q", got)
	}
}
