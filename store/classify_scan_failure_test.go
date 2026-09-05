package store

import "testing"

func TestClassifyScanFailureBuckets(t *testing.T) {
	cases := map[string]string{
		"scan did not complete (stale — reaped on startup)":                 "stale_reaped",
		"scan interrupted by process restart — requeued manually":          "stale_reaped",
		"prepare failed: failed to resolve files: no valid ref found for x": "invalid_ref",
		"unable to verify refs for o/r: connection reset":                  "forge_unavailable",
		"clone failed: 401 unauthorized":                                   "clone_auth",
		"context deadline exceeded":                                        "timeout",
		"workspace prepare failed: content not found":                      "prepare",
		"scanner parse_failed for trivy":                                   "scanner",
		"": "unknown",
	}
	for in, want := range cases {
		if got := ClassifyScanFailure(in); got != want {
			t.Fatalf("ClassifyScanFailure(%q)=%q want %q", in, got, want)
		}
	}
	if !IsNoiseScanFailure("scan did not complete (stale — reaped on startup)") {
		t.Fatal("stale reaped should be noise")
	}
	if IsNoiseScanFailure("no valid ref found for owner/repo") {
		t.Fatal("invalid_ref should remain actionable")
	}
}
