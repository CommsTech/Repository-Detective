package closure

import (
	"testing"
)

func TestVerifiedWhenAbsentAndScannerClean(t *testing.T) {
	result := Verify(VerifyInput{
		Evidence: Evidence{Fingerprint: "fp-1", OriginalSource: "staticcheck", Status: StatusPendingRescan, MergeCommitSHA: "abc"},
		Scan: ScanContext{
			ScanID: "scan-2", FingerprintsSeen: map[string]struct{}{},
			ScannerResults: map[string]string{"staticcheck": "clean"},
		},
		RequireScanner: true,
		PRMerged:       true,
	})
	if result.Status != StatusVerified {
		t.Fatalf("expected verified, got %s", result.Status)
	}
}

func TestStillPresentWhenFingerprintSeen(t *testing.T) {
	result := Verify(VerifyInput{
		Evidence: Evidence{Fingerprint: "fp-1", OriginalSource: "staticcheck", MergeCommitSHA: "abc"},
		Scan: ScanContext{
			ScanID: "scan-2", FingerprintsSeen: map[string]struct{}{"fp-1": {}},
			ScannerResults: map[string]string{"staticcheck": "clean"},
		},
		RequireScanner: true,
		PRMerged:       true,
	})
	if result.Status != StatusStillPresent {
		t.Fatalf("expected still_present, got %s", result.Status)
	}
}

func TestBlockedWhenScannerFailed(t *testing.T) {
	result := Verify(VerifyInput{
		Evidence: Evidence{Fingerprint: "fp-1", OriginalSource: "staticcheck", MergeCommitSHA: "abc"},
		Scan: ScanContext{
			ScanID: "scan-2", FingerprintsSeen: map[string]struct{}{},
			ScannerResults: map[string]string{"staticcheck": "failed"},
		},
		RequireScanner: true,
		PRMerged:       true,
	})
	if result.Status != StatusBlocked {
		t.Fatalf("expected blocked, got %s", result.Status)
	}
}

func TestBlockedWhenScannerMissing(t *testing.T) {
	result := Verify(VerifyInput{
		Evidence: Evidence{Fingerprint: "fp-1", OriginalSource: "staticcheck", MergeCommitSHA: "abc"},
		Scan: ScanContext{
			ScanID: "scan-2", FingerprintsSeen: map[string]struct{}{},
			ScannerResults: map[string]string{},
		},
		RequireScanner: true,
		PRMerged:       true,
	})
	if result.Status != StatusBlocked {
		t.Fatalf("expected blocked, got %s", result.Status)
	}
}

func TestPendingWhenNoScanAfterMerge(t *testing.T) {
	result := Verify(VerifyInput{
		Evidence: Evidence{Fingerprint: "fp-1", MergeCommitSHA: "abc", Status: StatusPendingRescan},
		Scan:     ScanContext{},
		PRMerged: true,
	})
	if result.Status != StatusPendingRescan {
		t.Fatalf("expected pending_rescan, got %s", result.Status)
	}
}

func TestScannerSucceeded(t *testing.T) {
	if !ScannerSucceeded("clean", true) || !ScannerSucceeded("found", true) {
		t.Fatal("clean/found should succeed")
	}
	if ScannerSucceeded("failed", true) || ScannerSucceeded("binary_missing", true) {
		t.Fatal("failed statuses should not succeed")
	}
}
