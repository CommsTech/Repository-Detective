package store

import (
	"fmt"
	"strings"
)

// ScannerRole is how a scanner participates in a profile's policy evaluation.
const (
	ScannerRoleRequired      = "REQUIRED"
	ScannerRoleOptional      = "OPTIONAL"
	ScannerRoleInformational = "INFORMATIONAL"
)

// ScannerCoverageState is the normalized coverage classification for one scanner run.
const (
	ScannerCoverageSuccess         = "SUCCESS"
	ScannerCoverageFailed          = "FAILED"
	ScannerCoverageTimeout         = "TIMEOUT"
	ScannerCoverageUnavailable     = "UNAVAILABLE"
	ScannerCoverageSkippedByPolicy = "SKIPPED_BY_POLICY"
	ScannerCoverageNotApplicable   = "NOT_APPLICABLE"
)

// ScannerCoverageRow is one scanner's role + coverage state for a scan.
type ScannerCoverageRow struct {
	Scanner string
	Role    string
	State   string
	Detail  string
}

// ScannerCoverageSummary aggregates required/optional completion for policy evaluation.
type ScannerCoverageSummary struct {
	RequiredTotal     int
	RequiredCompleted int
	RequiredIncomplete []string
	OptionalTotal     int
	OptionalCompleted int
	Rows              []ScannerCoverageRow
}

// RequiredScannersForProfile returns scanners that must complete for POLICY_MET.
// Custom profiles treat every enabled scanner as required.
func RequiredScannersForProfile(profile string, e EffectiveSettings) []string {
	profile = NormalizeScanProfile(profile)
	switch profile {
	case ScanProfileLight:
		return filterEnabled([]string{"gitleaks", "trivy"}, e)
	case ScanProfileStandard, ScanProfileDeep:
		return EnabledScannersList(e)
	default:
		return EnabledScannersList(e)
	}
}

// OptionalScannersForProfile returns enabled scanners that are not required.
func OptionalScannersForProfile(profile string, e EffectiveSettings) []string {
	required := map[string]struct{}{}
	for _, name := range RequiredScannersForProfile(profile, e) {
		required[name] = struct{}{}
	}
	var out []string
	for _, name := range EnabledScannersList(e) {
		if _, ok := required[name]; !ok {
			out = append(out, name)
		}
	}
	return out
}

func filterEnabled(names []string, e EffectiveSettings) []string {
	enabled := map[string]struct{}{}
	for _, name := range EnabledScannersList(e) {
		enabled[name] = struct{}{}
	}
	var out []string
	for _, name := range names {
		if _, ok := enabled[name]; ok {
			out = append(out, name)
		}
	}
	return out
}

// ClassifyScannerCoverageStatus maps a raw scanner status string to coverage state.
func ClassifyScannerCoverageStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "clean", "found", "success":
		return ScannerCoverageSuccess
	case "failed", "parse_failed":
		return ScannerCoverageFailed
	case "timed_out", "timeout":
		return ScannerCoverageTimeout
	case "binary_missing", "scanner_unavailable":
		return ScannerCoverageUnavailable
	case "disabled":
		return ScannerCoverageSkippedByPolicy
	case "no_supported_manifest":
		return ScannerCoverageNotApplicable
	default:
		if status == "" {
			return ScannerCoverageUnavailable
		}
		return ScannerCoverageFailed
	}
}

// CoverageStateBlocksPolicyMet reports whether a coverage state prevents POLICY_MET for a required scanner.
func CoverageStateBlocksPolicyMet(state string) bool {
	switch state {
	case ScannerCoverageFailed, ScannerCoverageTimeout, ScannerCoverageUnavailable:
		return true
	default:
		return false
	}
}

// BuildScannerCoverageSummary classifies scanner results against the profile required set.
func BuildScannerCoverageSummary(profile string, e EffectiveSettings, results []struct {
	Scanner string
	Status  string
	Detail  string
}) ScannerCoverageSummary {
	requiredSet := map[string]struct{}{}
	for _, name := range RequiredScannersForProfile(profile, e) {
		requiredSet[name] = struct{}{}
	}
	optionalSet := map[string]struct{}{}
	for _, name := range OptionalScannersForProfile(profile, e) {
		optionalSet[name] = struct{}{}
	}

	sum := ScannerCoverageSummary{
		RequiredTotal: len(requiredSet),
		OptionalTotal: len(optionalSet),
	}
	seenRequired := map[string]bool{}

	for _, result := range results {
		name := strings.ToLower(strings.TrimSpace(result.Scanner))
		state := ClassifyScannerCoverageStatus(result.Status)
		role := ScannerRoleInformational
		if _, ok := requiredSet[name]; ok {
			role = ScannerRoleRequired
			seenRequired[name] = true
		} else if _, ok := optionalSet[name]; ok {
			role = ScannerRoleOptional
		}
		row := ScannerCoverageRow{Scanner: name, Role: role, State: state, Detail: result.Detail}
		sum.Rows = append(sum.Rows, row)
		switch role {
		case ScannerRoleRequired:
			if CoverageStateBlocksPolicyMet(state) {
				sum.RequiredIncomplete = append(sum.RequiredIncomplete, fmt.Sprintf("%s (%s)", name, state))
			} else if state == ScannerCoverageSuccess || state == ScannerCoverageNotApplicable {
				sum.RequiredCompleted++
			}
		case ScannerRoleOptional:
			if state == ScannerCoverageSuccess || state == ScannerCoverageNotApplicable || state == ScannerCoverageSkippedByPolicy {
				sum.OptionalCompleted++
			}
		}
	}

	// Required scanners with no result row are incomplete.
	for name := range requiredSet {
		if !seenRequired[name] {
			sum.RequiredIncomplete = append(sum.RequiredIncomplete, fmt.Sprintf("%s (UNAVAILABLE)", name))
		}
	}
	return sum
}

// FormatCoverageRatio returns e.g. "6/8 required analyzers completed".
func FormatCoverageRatio(sum ScannerCoverageSummary) string {
	completed := sum.RequiredTotal - len(sum.RequiredIncomplete)
	if completed < 0 {
		completed = 0
	}
	if sum.RequiredCompleted > completed {
		completed = sum.RequiredCompleted
	}
	return fmt.Sprintf("%d/%d required analyzers completed", completed, sum.RequiredTotal)
}
