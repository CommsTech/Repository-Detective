package openclaw

import (
	"math"
	"strings"

	"git.commsnet.org/commstech/repository-detective/learning"
	"git.commsnet.org/commstech/repository-detective/store"
)

// CAHConfig controls CAH-gated candidate selection for AI recommendations.
type CAHConfig struct {
	Enabled              bool    `mapstructure:"ai_recommendations_cah_enabled"`
	MaxCandidates        int     `mapstructure:"ai_recommendations_cah_max_candidates"`
	MinUncertaintyScore  float64 `mapstructure:"ai_recommendations_cah_min_uncertainty_score"`
	TokenBudgetPerScan   int     `mapstructure:"ai_recommendations_token_budget_per_scan"`
	FailClosedOnRedaction bool   `mapstructure:"ai_recommendations_fail_closed_on_redaction_error"`
	RequireStrictJSON    bool    `mapstructure:"ai_recommendations_require_strict_json"`
	UseCAHHarness        bool    `mapstructure:"ai_recommendations_use_cah_harness"`
}

// DefaultCAHConfig returns safe CAH defaults.
func DefaultCAHConfig() CAHConfig {
	return CAHConfig{
		Enabled:              true,
		MaxCandidates:        10,
		MinUncertaintyScore:  0.45,
		TokenBudgetPerScan:   2000,
		FailClosedOnRedaction: true,
		RequireStrictJSON:    true,
		UseCAHHarness:        true,
	}
}

// CAHScore holds compact harness scoring for one finding.
type CAHScore struct {
	Fingerprint            string  `json:"fingerprint"`
	UncertaintyScore       float64 `json:"uncertainty_score"`
	FindingConfidenceGap   float64 `json:"finding_confidence_gap"`
	RuleFalsePositiveRate  float64 `json:"rule_false_positive_rate"`
	DuplicateHistory       bool    `json:"duplicate_history"`
	PreviousOperatorReview bool    `json:"previous_operator_decisions"`
	ScannerReliability     float64 `json:"scanner_reliability"`
	EvidenceCompleteness   float64 `json:"evidence_completeness"`
	TokenCostEstimate      int     `json:"token_cost_estimate"`
	Selected               bool    `json:"selected"`
	SkipReason             string  `json:"skip_reason,omitempty"`
}

// ScoreFinding estimates whether AI review may help operator confidence.
func ScoreFinding(f store.Finding, inst store.FindingInstance, hist FindingHistory) CAHScore {
	conf := f.Confidence
	if conf <= 0 {
		conf = 0.5
	}
	gap := 1.0 - conf
	if f.Severity == "critical" || f.Severity == "high" {
		gap *= 0.25
	}
	evidence := strings.TrimSpace(inst.EvidenceRedacted)
	evidenceScore := 0.3
	if len(evidence) > 40 {
		evidenceScore = 0.7
	}
	if len(evidence) > 200 {
		evidenceScore = 0.9
	}
	fpRate := 0.0
	if hist.ClosedAsFalsePositive {
		fpRate = 0.8
	}
	dup := hist.SeenBefore && f.Status == store.FindingStatusOpen
	uncertainty := gap*0.45 + fpRate*0.25 + (1-evidenceScore)*0.2
	if dup {
		uncertainty += 0.1
	}
	if learning.IsProtectedFromAutoDowngrade(f.Severity, f.Category) {
		uncertainty = math.Min(uncertainty, 0.35)
	}
	tokenEst := 80 + len(f.Title)/4 + len(evidence)/8
	return CAHScore{
		Fingerprint: f.Fingerprint, UncertaintyScore: uncertainty,
		FindingConfidenceGap: gap, RuleFalsePositiveRate: fpRate,
		DuplicateHistory: dup, PreviousOperatorReview: hist.ClosedAsFalsePositive || hist.ClosedAsFixed,
		ScannerReliability: scannerReliability(f.Source), EvidenceCompleteness: evidenceScore,
		TokenCostEstimate: tokenEst,
	}
}

func scannerReliability(source string) float64 {
	switch strings.ToLower(source) {
	case "gitleaks", "trivy", "grype", "govulncheck", "gosec":
		return 0.85
	case "static", "health":
		return 0.65
	case "graph":
		return 0.45
	default:
		return 0.55
	}
}

// SelectCAHCandidates picks findings worth sending to AI under token budget.
func SelectCAHCandidates(findings []store.Finding, instances map[int64]store.FindingInstance, history map[string]FindingHistory, cfg Config, cah CAHConfig) ([]store.Finding, []CAHScore) {
	cah = normalizeCAH(cah, cfg)
	if !cah.Enabled || !cah.UseCAHHarness {
		limit := cfg.MaxFindingsPerScan
		if limit <= 0 {
			limit = 25
		}
		if len(findings) > limit {
			findings = findings[:limit]
		}
		return findings, nil
	}
	var scores []CAHScore
	budget := cah.TokenBudgetPerScan
	if budget <= 0 {
		budget = cfg.MaxTokensPerScan
	}
	var selected []store.Finding
	for _, f := range findings {
		if learning.IsProtectedFromAutoDowngrade(f.Severity, f.Category) {
			sc := ScoreFinding(f, instances[f.ID], history[f.Fingerprint])
			sc.SkipReason = "protected severity/category"
			scores = append(scores, sc)
			continue
		}
		sc := ScoreFinding(f, instances[f.ID], history[f.Fingerprint])
		if sc.UncertaintyScore < cah.MinUncertaintyScore {
			sc.SkipReason = "below uncertainty threshold"
			scores = append(scores, sc)
			continue
		}
		if len(selected) >= cah.MaxCandidates {
			sc.SkipReason = "max candidates reached"
			scores = append(scores, sc)
			continue
		}
		if budget > 0 && sc.TokenCostEstimate > budget {
			sc.SkipReason = "token budget exhausted"
			scores = append(scores, sc)
			continue
		}
		sc.Selected = true
		budget -= sc.TokenCostEstimate
		scores = append(scores, sc)
		selected = append(selected, f)
	}
	if len(selected) == 0 && len(findings) > 0 && onlyUncertaintySkips(scores) {
		selected = fallbackCAHCandidates(findings, cah, cfg)
	}
	return selected, scores
}

func onlyUncertaintySkips(scores []CAHScore) bool {
	if len(scores) == 0 {
		return true
	}
	for _, sc := range scores {
		switch sc.SkipReason {
		case "", "below uncertainty threshold", "protected severity/category":
			continue
		default:
			return false
		}
	}
	return true
}

// fallbackCAHCandidates picks a small advisory set when CAH filters everything (e.g. all high-confidence).
func fallbackCAHCandidates(findings []store.Finding, cah CAHConfig, cfg Config) []store.Finding {
	limit := cah.MaxCandidates
	if limit <= 0 {
		limit = cfg.MaxFindingsPerScan
	}
	if limit <= 0 {
		limit = 25
	}
	var out []store.Finding
	for _, f := range findings {
		if learning.IsProtectedFromAutoDowngrade(f.Severity, f.Category) {
			continue
		}
		out = append(out, f)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func normalizeCAH(cah CAHConfig, cfg Config) CAHConfig {
	def := DefaultCAHConfig()
	if cah.MaxCandidates <= 0 {
		cah.MaxCandidates = def.MaxCandidates
	}
	if cah.MinUncertaintyScore <= 0 {
		cah.MinUncertaintyScore = def.MinUncertaintyScore
	}
	if cah.TokenBudgetPerScan <= 0 {
		cah.TokenBudgetPerScan = def.TokenBudgetPerScan
	}
	if cfg.MaxTokensPerScan > 0 && cah.TokenBudgetPerScan > cfg.MaxTokensPerScan {
		cah.TokenBudgetPerScan = cfg.MaxTokensPerScan
	}
	return cah
}
