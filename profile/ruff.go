package profile

import (
	"strings"

	"git.commsnet.org/commstech/repository-detective/scanners"
)

// CalibrateRuffResults downgrades style-only Ruff findings for homelab/infra repos.
func CalibrateRuffResults(results []scanners.RunResult, prof RepoProfile) []scanners.RunResult {
	if !IsHomelabInfra(prof) {
		return results
	}
	out := make([]scanners.RunResult, len(results))
	for i, r := range results {
		out[i] = r
		if strings.ToLower(r.Scanner) != "ruff" {
			continue
		}
		if len(r.Findings) == 0 {
			continue
		}
		findings := make([]scanners.Finding, len(r.Findings))
		copy(findings, r.Findings)
		for j, f := range findings {
			sev, conf := RuffSeverity(f.Reference, f.Severity, prof)
			findings[j].Severity = sev
			findings[j].Confidence = conf
		}
		out[i].Findings = findings
	}
	return out
}

// RuffSeverity adjusts Ruff rule severity for homelab/infra repos.
func RuffSeverity(code, defaultSeverity string, prof RepoProfile) (string, float64) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if !IsHomelabInfra(prof) {
		return defaultSeverity, 0.9
	}
	if isRuffSecurityCode(code) {
		return defaultSeverity, 0.85
	}
	if isRuffCorrectnessCode(code) {
		if defaultSeverity == "high" || defaultSeverity == "critical" {
			return "medium", 0.7
		}
		return defaultSeverity, 0.75
	}
	if isRuffStyleCode(code) {
		return "info", 0.45
	}
	return "info", 0.5
}

func isRuffSecurityCode(code string) bool {
	return strings.HasPrefix(code, "S") || strings.HasPrefix(code, "B")
}

func isRuffCorrectnessCode(code string) bool {
	switch {
	case strings.HasPrefix(code, "F821"), strings.HasPrefix(code, "F822"), strings.HasPrefix(code, "F823"):
		return true
	case strings.HasPrefix(code, "F401"), strings.HasPrefix(code, "F841"):
		return false
	case strings.HasPrefix(code, "F"):
		return true
	case strings.HasPrefix(code, "E9"), strings.HasPrefix(code, "E7"):
		return true
	default:
		return false
	}
}

func isRuffStyleCode(code string) bool {
	switch {
	case strings.HasPrefix(code, "I"), strings.HasPrefix(code, "W"), strings.HasPrefix(code, "N"):
		return true
	case strings.HasPrefix(code, "COM"), strings.HasPrefix(code, "Q"), strings.HasPrefix(code, "UP"):
		return true
	case code == "E501", strings.HasPrefix(code, "E5"):
		return true
	default:
		return false
	}
}
