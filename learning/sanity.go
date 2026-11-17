package learning

import (
	"context"
	"strings"
)

// SanityConfig controls optional LLM false-positive gate (disabled by default).
type SanityConfig struct {
	Enabled          bool
	MaxTokensPerScan int
	ApplyActions     bool
	LowMediumOnly    bool
}

// DefaultSanityConfig returns safe defaults (off).
func DefaultSanityConfig() SanityConfig {
	return SanityConfig{Enabled: false, MaxTokensPerScan: 0, ApplyActions: false, LowMediumOnly: true}
}

// DeepSanityDefaultMaxTokens is the per-scan token budget when Deep enables the
// gate without an explicit llm_sanity_gate_max_tokens_per_scan override.
const DeepSanityDefaultMaxTokens = 800

// ConfigForScan builds SanityConfig for a resolved scan. When enabled is false
// the gate stays fully off; when enabled with no token budget, Deep's default
// budget is applied so ShouldEvaluate can run.
func ConfigForScan(enabled bool, maxTokens int, applyActions, lowMediumOnly bool) SanityConfig {
	if !enabled {
		return DefaultSanityConfig()
	}
	if maxTokens <= 0 {
		maxTokens = DeepSanityDefaultMaxTokens
	}
	return SanityConfig{
		Enabled:          true,
		MaxTokensPerScan: maxTokens,
		ApplyActions:     applyActions,
		LowMediumOnly:    lowMediumOnly,
	}
}

// SanityDecision is strict JSON output from optional LLM gate.
type SanityDecision struct {
	RealIssue  string  `json:"real_issue"` // true | false | unknown
	Confidence float64 `json:"confidence"`
	Rationale  string  `json:"rationale"`
	SafeAction string  `json:"safe_action"`
}

// SanityGate evaluates low/medium findings when enabled.
type SanityGate struct {
	cfg SanityConfig
}

// NewSanityGate creates a gate from config.
func NewSanityGate(cfg SanityConfig) *SanityGate {
	return &SanityGate{cfg: cfg}
}

// ShouldEvaluate reports whether a finding may be sent to optional LLM review.
func (g *SanityGate) ShouldEvaluate(severity, category string) bool {
	if g == nil || !g.cfg.Enabled || g.cfg.MaxTokensPerScan <= 0 {
		return false
	}
	if IsProtectedFromAutoDowngrade(severity, category) {
		return false
	}
	if g.cfg.LowMediumOnly {
		s := strings.ToLower(severity)
		return s == "low" || s == "medium" || s == "info"
	}
	return true
}

// Evaluate is a deterministic stub when LLM is disabled or unavailable.
func (g *SanityGate) Evaluate(_ context.Context, _ string, severity, category string) (SanityDecision, bool) {
	if !g.ShouldEvaluate(severity, category) {
		return SanityDecision{}, false
	}
	return SanityDecision{
		RealIssue:  "unknown",
		Confidence: 0,
		Rationale:  "LLM sanity gate enabled; provider review not yet applied",
		SafeAction: "no_change",
	}, false
}
