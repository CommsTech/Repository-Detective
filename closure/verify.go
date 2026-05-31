package closure

import "fmt"

// VerifyInput bundles evidence and scan data for closure verification.
type VerifyInput struct {
	Evidence           Evidence
	Scan               ScanContext
	RequireScanner     bool
	PRMerged           bool
}

// VerifyResult is the outcome of closure verification.
type VerifyResult struct {
	Status             string
	Reason             string
	Blockers           []string
	FingerprintPresent bool
	ScannerStatus      string
}

// Verify evaluates closure evidence against a completed scan.
func Verify(in VerifyInput) VerifyResult {
	if !in.PRMerged && in.Evidence.Status == StatusPendingRescan && in.Evidence.PatchAttemptID == "" {
		return VerifyResult{
			Status:   StatusPendingRescan,
			Reason:   "remediation PR not merged yet",
			Blockers: []string{"PR not merged"},
		}
	}
	if !in.PRMerged && in.Evidence.PatchAttemptID != "" && in.Evidence.MergeCommitSHA == "" {
		return VerifyResult{
			Status:   StatusPendingRescan,
			Reason:   "waiting for remediation PR merge",
			Blockers: []string{"PR not merged"},
		}
	}
	if in.Scan.ScanID == "" {
		return VerifyResult{
			Status:   StatusPendingRescan,
			Reason:   "no verification scan after merge",
			Blockers: []string{"no scan after merge"},
		}
	}

	scannerName := ScannerForSource(in.Evidence.OriginalSource)
	scannerStatus := in.Scan.ScannerResults[scannerName]
	_, present := in.Scan.FingerprintsSeen[in.Evidence.Fingerprint]

	if present {
		return VerifyResult{
			Status:             StatusStillPresent,
			Reason:             "fingerprint still present after remediation",
			FingerprintPresent: true,
			ScannerStatus:      scannerStatus,
		}
	}

	if in.RequireScanner {
		if ScannerMissing(scannerStatus) {
			return VerifyResult{
				Status:        StatusBlocked,
				Reason:        fmt.Sprintf("original scanner %q did not run in verification scan", scannerName),
				Blockers:      []string{"scanner missing: " + scannerName},
				ScannerStatus: scannerStatus,
			}
		}
		if !ScannerSucceeded(scannerStatus, true) {
			return VerifyResult{
				Status:        StatusBlocked,
				Reason:        fmt.Sprintf("original scanner %q status %q is not successful", scannerName, scannerStatus),
				Blockers:      []string{"scanner failed: " + scannerName + " (" + scannerStatus + ")"},
				ScannerStatus: scannerStatus,
			}
		}
	}

	return VerifyResult{
		Status:             StatusVerified,
		Reason:             "fingerprint absent and scanner evidence satisfied",
		FingerprintPresent: false,
		ScannerStatus:      scannerStatus,
	}
}
