package openclaw

import (
	"fmt"
	"sort"
	"strings"

	"git.commsnet.org/commstech/repository-detective/learning"
	"git.commsnet.org/commstech/repository-detective/store"
)

// HarnessVersion identifies the advisory packet/prompt contract.
// v3: value-mode candidate selection + tighter budgets (ponytail-influenced).
const HarnessVersion = "rd-cah-v3"

// CAHConfig controls CAH-gated candidate selection for AI recommendations.
type CAHConfig struct {
	Enabled               bool    `mapstructure:"ai_recommendations_cah_enabled"`
	MaxCandidates         int     `mapstructure:"ai_recommendations_cah_max_candidates"`
	MinUncertaintyScore   float64 `mapstructure:"ai_recommendations_cah_min_uncertainty_score"`
	TokenBudgetPerScan    int     `mapstructure:"ai_recommendations_token_budget_per_scan"`
	FailClosedOnRedaction bool    `mapstructure:"ai_recommendations_fail_closed_on_redaction_error"`
	RequireStrictJSON     bool    `mapstructure:"ai_recommendations_require_strict_json"`
	UseCAHHarness         bool    `mapstructure:"ai_recommendations_use_cah_harness"`
	MaxProtectedCoach     int     `mapstructure:"ai_recommendations_cah_max_protected_coach"`
}

// DefaultCAHConfig returns safe CAH defaults.
// Lean budgets: prefer a few actionable security findings over GRAPH-ORPHAN noise.
func DefaultCAHConfig() CAHConfig {
	return CAHConfig{
		Enabled:               true,
		MaxCandidates:         6,
		MinUncertaintyScore:   0.45,
		TokenBudgetPerScan:    1200,
		FailClosedOnRedaction: true,
		RequireStrictJSON:     true,
		UseCAHHarness:         true,
		MaxProtectedCoach:     2,
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
	ProtectedFromDowngrade bool    `json:"protected_from_downgrade"`
	Selected               bool    `json:"selected"`
	SkipReason             string  `json:"skip_reason,omitempty"`
	CoachLane              string  `json:"coach_lane,omitempty"` // review | protected_coach
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
	if evidence == "" {
		evidence = extractSnippet(inst.RawMetadataJSON)
	}
	evidenceScore := 0.2
	if len(evidence) > 20 {
		evidenceScore = 0.55
	}
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
	// Prefer actionable security/source diversity over repetitive lint noise.
	switch strings.ToLower(f.Source) {
	case "ruff", "tech_debt", "maintainability":
		uncertainty *= 0.55
	case "semgrep", "gosec", "gitleaks", "grype", "trivy", "govulncheck":
		uncertainty += 0.08
	}
	if isNoiseFinding(f) {
		uncertainty *= 0.35
	}
	if isActionableSecurityFinding(f) {
		uncertainty += 0.12
	}
	protected := learning.IsProtectedFromAutoDowngrade(f.Severity, f.Category)
	tokenEst := 90 + len(f.Title)/4 + len(evidence)/8 + len(f.RuleID)/8
	return CAHScore{
		Fingerprint: f.Fingerprint, UncertaintyScore: uncertainty,
		FindingConfidenceGap: gap, RuleFalsePositiveRate: fpRate,
		DuplicateHistory: dup, PreviousOperatorReview: hist.ClosedAsFalsePositive || hist.ClosedAsFixed,
		ScannerReliability: scannerReliability(f.Source), EvidenceCompleteness: evidenceScore,
		TokenCostEstimate: tokenEst, ProtectedFromDowngrade: protected,
	}
}

// isNoiseFinding reports GRAPH-ORPHAN / HEALTH / LINT style findings that rarely
// justify AI spend when budgets are tight (ponytail: skip work that won't ship value).
func isNoiseFinding(f store.Finding) bool {
	rule := strings.ToUpper(strings.TrimSpace(f.RuleID))
	src := strings.ToLower(strings.TrimSpace(f.Source))
	cat := strings.ToLower(strings.TrimSpace(f.Category))
	if strings.HasPrefix(rule, "GRAPH-ORPHAN") || strings.Contains(rule, "GRAPH-ORPHAN") {
		return true
	}
	if strings.HasPrefix(rule, "HEALTH-") || strings.HasPrefix(rule, "LINT-") {
		return true
	}
	if src == "ruff" || src == "tech_debt" || src == "maintainability" || src == "graph" {
		return true
	}
	if cat == "maintainability" || cat == "tech_debt" || cat == "code_smell" {
		return true
	}
	return false
}

func isActionableSecurityFinding(f store.Finding) bool {
	sev := strings.ToLower(strings.TrimSpace(f.Severity))
	if sev == "critical" || sev == "high" {
		if learning.IsProtectedFromAutoDowngrade(f.Severity, f.Category) ||
			strings.Contains(strings.ToLower(f.Category), "security") ||
			strings.Contains(strings.ToLower(f.Category), "secret") {
			return true
		}
	}
	src := strings.ToLower(f.Source)
	switch src {
	case "gitleaks", "gosec", "semgrep", "grype", "trivy", "govulncheck":
		return sev == "critical" || sev == "high" || sev == "medium"
	}
	// Mutable GitHub Actions / workflow secrets are high-value triage targets.
	path := strings.ToLower(f.FilePath)
	if strings.Contains(path, ".github/workflows/") || strings.Contains(path, ".gitea/workflows/") {
		return true
	}
	return false
}

func scannerReliability(source string) float64 {
	switch strings.ToLower(source) {
	case "gitleaks", "trivy", "grype", "govulncheck", "gosec":
		return 0.85
	case "semgrep":
		return 0.75
	case "static", "health":
		return 0.65
	case "graph", "tech_debt", "ruff":
		return 0.45
	default:
		return 0.55
	}
}

type scoredFinding struct {
	finding store.Finding
	score   CAHScore
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

	valueMode := normalizeValueMode(cfg.ValueMode)
	scored := make([]scoredFinding, 0, len(findings))
	for _, f := range findings {
		sc := ScoreFinding(f, instances[f.ID], history[f.Fingerprint])
		scored = append(scored, scoredFinding{finding: f, score: sc})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if valueMode == ValueModeActionableSecurity {
			ai, aj := valueRank(scored[i].finding), valueRank(scored[j].finding)
			if ai != aj {
				return ai > aj
			}
		}
		// Prefer higher uncertainty, then evidence, then severity rank.
		if scored[i].score.UncertaintyScore != scored[j].score.UncertaintyScore {
			return scored[i].score.UncertaintyScore > scored[j].score.UncertaintyScore
		}
		if scored[i].score.EvidenceCompleteness != scored[j].score.EvidenceCompleteness {
			return scored[i].score.EvidenceCompleteness > scored[j].score.EvidenceCompleteness
		}
		return severityRank(scored[i].finding.Severity) > severityRank(scored[j].finding.Severity)
	})

	budget := cah.TokenBudgetPerScan
	if budget <= 0 {
		budget = cfg.MaxTokensPerScan
	}
	maxTotal := cah.MaxCandidates
	if cfg.MaxFindingsPerScan > 0 && cfg.MaxFindingsPerScan < maxTotal {
		maxTotal = cfg.MaxFindingsPerScan
	}
	maxProtected := cah.MaxProtectedCoach
	if maxProtected < 0 {
		maxProtected = 0
	}

	var selected []store.Finding
	var scores []CAHScore
	sourceCounts := map[string]int{}
	protectedSelected := 0
	reviewSelected := 0

	for _, item := range scored {
		sc := item.score
		f := item.finding
		src := strings.ToLower(f.Source)

		if sc.ProtectedFromDowngrade {
			if protectedSelected >= maxProtected {
				sc.SkipReason = "protected coach lane full"
				scores = append(scores, sc)
				continue
			}
			if budget > 0 && sc.TokenCostEstimate > budget {
				sc.SkipReason = "token budget exhausted"
				scores = append(scores, sc)
				continue
			}
			if len(selected) >= maxTotal {
				sc.SkipReason = "max candidates reached"
				scores = append(scores, sc)
				continue
			}
			sc.Selected = true
			sc.CoachLane = "protected_coach"
			budget -= sc.TokenCostEstimate
			protectedSelected++
			sourceCounts[src]++
			scores = append(scores, sc)
			selected = append(selected, f)
			continue
		}

		if valueMode == ValueModeActionableSecurity && isNoiseFinding(f) && !sc.ProtectedFromDowngrade {
			// ponytail: do not spend AI tokens on orphan/lint/health noise.
			sc.SkipReason = "value_mode skips noise"
			scores = append(scores, sc)
			continue
		}
		if sc.UncertaintyScore < cah.MinUncertaintyScore {
			sc.SkipReason = "below uncertainty threshold"
			scores = append(scores, sc)
			continue
		}
		// Soft diversity: avoid flooding the packet with one noisy scanner.
		noiseCap := 3
		if valueMode == ValueModeActionableSecurity {
			noiseCap = 1
		}
		if sourceCounts[src] >= noiseCap && (src == "ruff" || src == "tech_debt" || src == "static" || src == "graph" || src == "health") {
			sc.SkipReason = "source diversity cap"
			scores = append(scores, sc)
			continue
		}
		if len(selected) >= maxTotal {
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
		sc.CoachLane = "review"
		budget -= sc.TokenCostEstimate
		reviewSelected++
		sourceCounts[src]++
		scores = append(scores, sc)
		selected = append(selected, f)
	}

	if len(selected) < maxTotal && len(findings) > 0 {
		selected = fillCAHCandidates(selected, findings, cah, cfg, maxTotal)
	}
	_ = reviewSelected
	return selected, scores
}

func fillCAHCandidates(selected []store.Finding, findings []store.Finding, cah CAHConfig, cfg Config, maxTotal int) []store.Finding {
	if maxTotal <= 0 {
		maxTotal = cah.MaxCandidates
	}
	valueMode := normalizeValueMode(cfg.ValueMode)
	seen := map[string]bool{}
	sourceCounts := map[string]int{}
	protectedCount := 0
	for _, f := range selected {
		seen[f.Fingerprint] = true
		src := strings.ToLower(f.Source)
		sourceCounts[src]++
		if learning.IsProtectedFromAutoDowngrade(f.Severity, f.Category) {
			protectedCount++
		}
	}
	// Prefer higher severity + non-lint sources when topping up the packet.
	remaining := append([]store.Finding(nil), findings...)
	sort.SliceStable(remaining, func(i, j int) bool {
		if valueMode == ValueModeActionableSecurity {
			ai, aj := valueRank(remaining[i]), valueRank(remaining[j])
			if ai != aj {
				return ai > aj
			}
		}
		ri, rj := severityRank(remaining[i].Severity), severityRank(remaining[j].Severity)
		if ri != rj {
			return ri > rj
		}
		return fillPriority(remaining[i].Source) > fillPriority(remaining[j].Source)
	})
	for _, f := range remaining {
		if len(selected) >= maxTotal {
			break
		}
		if seen[f.Fingerprint] {
			continue
		}
		if valueMode == ValueModeActionableSecurity && isNoiseFinding(f) &&
			!learning.IsProtectedFromAutoDowngrade(f.Severity, f.Category) {
			continue
		}
		src := strings.ToLower(f.Source)
		if sourceCounts[src] >= 2 && (src == "ruff" || src == "tech_debt" || src == "static" || src == "graph") {
			continue
		}
		if learning.IsProtectedFromAutoDowngrade(f.Severity, f.Category) {
			if protectedCount >= cah.MaxProtectedCoach {
				continue
			}
			protectedCount++
		}
		selected = append(selected, f)
		seen[f.Fingerprint] = true
		sourceCounts[src]++
	}
	return selected
}

func fillPriority(source string) int {
	switch strings.ToLower(source) {
	case "gitleaks", "gosec", "semgrep":
		return 5
	case "grype", "trivy", "govulncheck":
		return 4
	case "checkov", "hadolint":
		return 3
	case "static":
		return 2
	case "graph", "health", "ruff", "tech_debt":
		return 1
	default:
		return 2
	}
}

// valueRank prefers actionable security / workflow findings over orphan & lint noise.
func valueRank(f store.Finding) int {
	if isActionableSecurityFinding(f) {
		return 100 + severityRank(f.Severity)
	}
	if isNoiseFinding(f) {
		return severityRank(f.Severity)
	}
	return 40 + severityRank(f.Severity) + fillPriority(f.Source)
}

func severityRank(sev string) int {
	switch strings.ToLower(strings.TrimSpace(sev)) {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium", "warning", "warn":
		return 3
	case "low":
		return 2
	case "info", "informational", "note":
		return 1
	default:
		return 0
	}
}

// fallbackCAHCandidates picks a small advisory set when CAH filters everything.
func fallbackCAHCandidates(findings []store.Finding, cah CAHConfig, cfg Config) []store.Finding {
	limit := cah.MaxCandidates
	if limit <= 0 {
		limit = cfg.MaxFindingsPerScan
	}
	if limit <= 0 {
		limit = 25
	}
	var out []store.Finding
	protected := 0
	for _, f := range findings {
		if learning.IsProtectedFromAutoDowngrade(f.Severity, f.Category) {
			if protected >= cah.MaxProtectedCoach {
				continue
			}
			protected++
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
	if cah.MaxProtectedCoach == 0 {
		cah.MaxProtectedCoach = def.MaxProtectedCoach
	}
	if cfg.MaxTokensPerScan > 0 && cah.TokenBudgetPerScan > cfg.MaxTokensPerScan {
		cah.TokenBudgetPerScan = cfg.MaxTokensPerScan
	}
	return cah
}

// CoachFocusForFinding returns a short coaching cue for the agent.
func CoachFocusForFinding(f store.Finding, protected bool) string {
	if protected {
		return "Remediation coaching only — do not downgrade severity or mark false positive."
	}
	src := strings.ToLower(f.Source)
	switch {
	case src == "tech_debt" || strings.Contains(strings.ToLower(f.RuleID), "TECH"):
		return "Judge whether this is actionable debt vs noise; prefer calibrate_repo_scope or leave_visible when repetitive."
	case src == "ruff" || src == "staticcheck":
		return "Lint noise check — false positives and repo-scope calibration are common."
	case src == "gitleaks" || strings.Contains(strings.ToLower(f.Category), "secret"):
		return "Secret handling — rotate/remove if real; example/fixture paths often false positive."
	case src == "grype" || src == "trivy" || src == "govulncheck":
		return "Dependency risk — confirm reachability and patched versions; avoid panic upgrades."
	case src == "semgrep" || src == "gosec":
		return "Code security pattern — validate context before suggesting severity changes."
	default:
		return "Triage true positive vs false positive and recommend the safest next operator action."
	}
}

// DescribeFinding builds a compact description that is not a title duplicate.
func DescribeFinding(f store.Finding, evidence string) string {
	parts := []string{}
	if cat := strings.TrimSpace(f.Category); cat != "" {
		parts = append(parts, "category="+cat)
	}
	if src := strings.TrimSpace(f.Source); src != "" {
		parts = append(parts, "source="+src)
	}
	if rule := strings.TrimSpace(f.RuleID); rule != "" {
		parts = append(parts, "rule="+rule)
	}
	if pkg := strings.TrimSpace(f.PackageName); pkg != "" {
		parts = append(parts, "package="+pkg)
	}
	base := strings.Join(parts, "; ")
	ev := strings.TrimSpace(evidence)
	if ev == "" {
		if base == "" {
			return strings.TrimSpace(f.Title)
		}
		return base
	}
	if len(ev) > 240 {
		ev = ev[:240] + "…"
	}
	if base == "" {
		return ev
	}
	return fmt.Sprintf("%s | evidence=%s", base, ev)
}
