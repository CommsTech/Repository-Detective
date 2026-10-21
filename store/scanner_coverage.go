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
	RequiredTotal       int
	RequiredCompleted   int
	RequiredIncomplete  []string
	OptionalTotal       int
	OptionalCompleted   int
	Rows                []ScannerCoverageRow
	ZeroRequiredAllowed bool // true only for explicit observation/no-required profiles (none today)
}

// ProfileDeclaredRequiredScanners returns the fixed REQUIRED set for a profile.
// Disabling a scanner never removes it from this set (RD-012A).
func ProfileDeclaredRequiredScanners(profile string) []string {
	switch NormalizeScanProfile(profile) {
	case ScanProfileLight:
		return []string{"gitleaks", "trivy"}
	case ScanProfileStandard, ScanProfileDeep:
		// Minimum non-empty required analysis contract for Standard/Deep.
		// Additional operator-enabled scanners are also required (see RequiredScannersForProfile).
		return []string{"gitleaks", "trivy", "grype", "semgrep"}
	default:
		return nil
	}
}

// RequiredScannersForProfile returns scanners that must satisfy the evidence contract for POLICY_MET.
//
// Rules:
//   - Light: fixed {gitleaks, trivy} even if operator-disabled
//   - Standard/Deep: declared minimum UNION all currently enabled scanners
//   - Custom: all enabled scanners; empty set is incomplete (never silent 0/0 POLICY_MET)
func RequiredScannersForProfile(profile string, e EffectiveSettings) []string {
	profile = NormalizeScanProfile(profile)
	declared := ProfileDeclaredRequiredScanners(profile)
	enabled := EnabledScannersList(e)

	switch profile {
	case ScanProfileLight:
		return uniqueStrings(declared)
	case ScanProfileStandard, ScanProfileDeep:
		return uniqueStrings(append(append([]string{}, declared...), enabled...))
	default:
		return uniqueStrings(enabled)
	}
}

// OptionalScannersForProfile returns enabled scanners that are not profile-required.
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

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// ScannerEnabledInSettings reports whether a named scanner is operator-enabled.
func ScannerEnabledInSettings(name string, e EffectiveSettings) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, enabled := range EnabledScannersList(e) {
		if enabled == name {
			return true
		}
	}
	return false
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

// CoverageStateBlocksPolicyMet reports whether a coverage state prevents POLICY_MET for a REQUIRED scanner.
// SKIPPED_BY_POLICY (disabled) is incomplete for required scanners — never silent success (RD-012A).
// NOT_APPLICABLE does not block only when the tool legitimately determined non-applicability
// (e.g. no_supported_manifest), not because the operator disabled it.
func CoverageStateBlocksPolicyMet(state string) bool {
	switch state {
	case ScannerCoverageFailed, ScannerCoverageTimeout, ScannerCoverageUnavailable, ScannerCoverageSkippedByPolicy:
		return true
	default:
		return false
	}
}

// RequiredEvidenceSatisfied reports whether POLICY_MET is allowed given coverage.
func RequiredEvidenceSatisfied(sum ScannerCoverageSummary) bool {
	if sum.RequiredTotal == 0 {
		return sum.ZeroRequiredAllowed
	}
	return len(sum.RequiredIncomplete) == 0
}

// BuildScannerCoverageSummary classifies scanner results against the profile required set.
func BuildScannerCoverageSummary(profile string, e EffectiveSettings, results []struct {
	Scanner string
	Status  string
	Detail  string
}) ScannerCoverageSummary {
	requiredList := RequiredScannersForProfile(profile, e)
	requiredSet := map[string]struct{}{}
	for _, name := range requiredList {
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
	resultByName := map[string]struct {
		Status string
		Detail string
	}{}

	for _, result := range results {
		name := strings.ToLower(strings.TrimSpace(result.Scanner))
		resultByName[name] = struct {
			Status string
			Detail string
		}{Status: result.Status, Detail: result.Detail}
	}

	// Ensure every required scanner appears — synthesize disabled/unavailable when missing.
	for name := range requiredSet {
		if _, ok := resultByName[name]; ok {
			continue
		}
		status := "scanner_unavailable"
		detail := "no scanner result produced"
		if !ScannerEnabledInSettings(name, e) {
			status = "disabled"
			detail = "required by profile but operator-disabled"
		}
		resultByName[name] = struct {
			Status string
			Detail string
		}{Status: status, Detail: detail}
	}

	for name, result := range resultByName {
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

	for name := range requiredSet {
		if !seenRequired[name] {
			sum.RequiredIncomplete = append(sum.RequiredIncomplete, fmt.Sprintf("%s (UNAVAILABLE)", name))
		}
	}
	return sum
}

// FormatCoverageRatio returns e.g. "6/8 required analyzers completed".
func FormatCoverageRatio(sum ScannerCoverageSummary) string {
	if sum.RequiredTotal == 0 {
		if sum.ZeroRequiredAllowed {
			return "0/0 required analyzers (observation profile — no required analysis)"
		}
		return "0/0 required analyzers completed (incomplete — empty required set)"
	}
	incomplete := len(uniqueIncompleteNames(sum.RequiredIncomplete))
	completed := sum.RequiredTotal - incomplete
	if completed < 0 {
		completed = 0
	}
	return fmt.Sprintf("%d/%d required analyzers completed", completed, sum.RequiredTotal)
}

func uniqueIncompleteNames(items []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, item := range items {
		name := item
		if i := strings.IndexByte(item, ' '); i > 0 {
			name = item[:i]
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}
