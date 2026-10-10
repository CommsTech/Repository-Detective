package preinstall

import (
	"strings"

	"git.commsnet.org/commstech/repository-detective/scanners"
	"git.commsnet.org/commstech/repository-detective/store"
)

// RiskOutcome is the computed risk score and recommendation.
type RiskOutcome struct {
	Score          int
	Recommendation string
	Explanation    map[string]any
}

// ComputeRiskScore calculates a transparent 0–100 risk score from findings and scanner results.
func ComputeRiskScore(findings []store.AuditFinding, scannerResults []store.AuditScannerResult) RiskOutcome {
	score := 0
	breakdown := map[string]int{}
	secretSeen := false

	for _, f := range findings {
		points := severityPoints(f.Severity, f.Source, f.Category)
		if f.Confidence > 0 && f.Confidence < 0.6 {
			if isHealthQualityFinding(f.Source, f.Category) || strings.EqualFold(f.Source, "graph") {
				points = 1
				breakdown["needs_review"] = breakdown["needs_review"] + points
				score += points
				continue
			}
		}
		if f.Source == "gitleaks" || f.Category == "secret" || strings.Contains(strings.ToLower(f.Title), "secret") {
			secretSeen = true
			if points < 20 {
				points = 20
			}
		}
		score += points
		breakdown[f.Severity] = breakdown[f.Severity] + points
	}

	for _, sr := range scannerResults {
		if isBadScannerStatus(sr.Status) {
			score += 5
			breakdown["scanner_failure"] = breakdown["scanner_failure"] + 5
		}
	}

	if score > 100 {
		score = 100
	}

	rec := store.AuditRecommendationSafe
	if secretSeen && score < 20 {
		rec = store.AuditRecommendationCaution
	}
	switch {
	case score >= 50:
		rec = store.AuditRecommendationDoNotInstall
	case score >= 20:
		rec = store.AuditRecommendationCaution
	default:
		if !secretSeen {
			rec = store.AuditRecommendationSafe
		}
	}

	return RiskOutcome{
		Score:          score,
		Recommendation: rec,
		Explanation: map[string]any{
			"formula":          "critical +35, high +20, medium +10, low +3, scanner failure +5, secrets minimum caution",
			"breakdown":        breakdown,
			"finding_count":    len(findings),
			"secret_finding":   secretSeen,
			"scanner_failures": countBadScanners(scannerResults),
		},
	}
}

func severityPoints(severity, source, category string) int {
	points := baseSeverityPoints(severity)
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "govulncheck" || source == "gosec" || source == "hadolint" || source == "checkov" {
		if strings.ToLower(severity) == "high" && points < 25 {
			return 25
		}
	}
	if source == "staticcheck" || source == "hadolint" || isHealthQualityFinding(source, category) {
		if strings.ToLower(severity) == "low" {
			return 1
		}
		if strings.ToLower(severity) == "info" || strings.ToLower(severity) == "style" {
			return 1
		}
		if points > 10 {
			return 10
		}
	}
	return points
}

func baseSeverityPoints(severity string) int {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return 35
	case "high":
		return 20
	case "medium":
		return 10
	case "low":
		return 3
	default:
		return 3
	}
}

func isBadScannerStatus(status string) bool {
	switch scanners.Status(status) {
	case scanners.StatusFailed, scanners.StatusTimedOut, scanners.StatusParseFailed, scanners.StatusBinaryMissing:
		return true
	default:
		return false
	}
}

func countBadScanners(results []store.AuditScannerResult) int {
	n := 0
	for _, r := range results {
		if isBadScannerStatus(r.Status) {
			n++
		}
	}
	return n
}

func isHealthQualityFinding(source, category string) bool {
	switch strings.ToLower(source) {
	case "tech_debt", "reliability", "maintainability", "test_gap", "performance", "health", "ai_generated_risk", "graph", "staticcheck":
		return true
	}
	switch strings.ToLower(category) {
	case "tech_debt", "reliability", "maintainability", "test_gap", "performance", "code_quality", "ai_generated_risk", "architecture":
		return true
	}
	return false
}
