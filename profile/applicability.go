package profile

import (
	"strings"

	"git.commsnet.org/commstech/repository-detective/scanners"
)

// AnnotateScannerResults sets applicability reasons on scanner run results.
func AnnotateScannerResults(results []scanners.RunResult, prof RepoProfile) []scanners.RunResult {
	out := make([]scanners.RunResult, len(results))
	for i, r := range results {
		out[i] = r
		if out[i].ApplicabilityReason != "" {
			continue
		}
		out[i].ApplicabilityReason = inferApplicability(r, prof)
	}
	return out
}

func inferApplicability(r scanners.RunResult, prof RepoProfile) string {
	name := strings.ToLower(strings.TrimSpace(r.Scanner))
	status := r.Status
	detail := strings.ToLower(r.Detail)

	switch status {
	case scanners.StatusDisabled:
		return ApplicabilitySkippedDisabledByPolicy
	case scanners.StatusBinaryMissing:
		return ApplicabilitySkippedToolUnavailable
	}

	switch {
	case strings.Contains(detail, "no go"):
		return ApplicabilitySkippedNoMatchingFiles
	case strings.Contains(detail, "no dockerfile"):
		return ApplicabilitySkippedNoMatchingFiles
	case strings.Contains(detail, "no terraform") || strings.Contains(detail, "no iac"):
		return ApplicabilitySkippedNoMatchingFiles
	case strings.Contains(detail, "security analysis disabled"), strings.Contains(detail, "quality analysis disabled"):
		return ApplicabilitySkippedDisabledByPolicy
	}

	if prof.IsDocsOnlyRepo() && isAppScanner(name) {
		return ApplicabilitySkippedDocsOnlyRepo
	}

	if status == scanners.StatusClean && len(r.Findings) == 0 && detail != "" {
		if strings.Contains(detail, "no ") {
			return ApplicabilitySkippedNoMatchingFiles
		}
	}

	if len(r.Findings) == 0 && (status == scanners.StatusClean || status == scanners.StatusFound) {
		return ApplicabilityApplicable
	}
	return ApplicabilityApplicable
}

func isAppScanner(name string) bool {
	switch name {
	case "govulncheck", "gosec", "staticcheck", "linters":
		return true
	default:
		return false
	}
}

// ScannerFindingCounts returns findings count excluding non-applicable skipped scanners.
func ScannerFindingCounts(results []scanners.RunResult) (applicableFindings int, skippedScanners int) {
	for _, r := range results {
		reason := r.ApplicabilityReason
		if reason == "" {
			reason = ApplicabilityApplicable
		}
		switch reason {
		case ApplicabilitySkippedToolUnavailable, ApplicabilitySkippedDisabledByPolicy,
			ApplicabilitySkippedNoMatchingFiles, ApplicabilitySkippedDocsOnlyRepo,
			ApplicabilitySkippedUnsupportedType, ApplicabilitySkippedGeneratedVendor:
			skippedScanners++
		default:
			applicableFindings += len(r.Findings)
		}
	}
	return applicableFindings, skippedScanners
}
