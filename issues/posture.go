package issues

import (
	"fmt"
	"strings"

	"git.commsnet.org/commstech/repository-detective/ai"
)

// RepositoryPosture is a commercially useful scan summary without a fake percentage score.
type RepositoryPosture struct {
	Headline              string
	Critical              int
	High                  int
	Medium                int
	Low                   int
	Info                  int
	Total                 int
	AutoRemediationReady  int
	NeedsHumanTriage      int
	ExcludedByCalibration int
	NewSincePrevious      int
	ResolvedSincePrevious int
	TriggerReason         string
}

// PostureTriggerConfig controls when a rare forge posture issue is filed.
type PostureTriggerConfig struct {
	HighThreshold           int // escalate when High >= this (default 3)
	ActionableJumpThreshold int // escalate on material jump in C+H+M (default 10)
}

// DefaultPostureTriggerConfig returns sensible defaults.
func DefaultPostureTriggerConfig() PostureTriggerConfig {
	return PostureTriggerConfig{HighThreshold: 3, ActionableJumpThreshold: 10}
}

func (p RepositoryPosture) actionable() int {
	return p.Critical + p.High + p.Medium
}

// BuildRepositoryPosture summarizes findings for dashboards / rare forge posture issues.
func BuildRepositoryPosture(result *ai.CodeAnalysisResult) RepositoryPosture {
	out := RepositoryPosture{Headline: "Healthy"}
	if result == nil {
		out.Headline = "Unknown"
		return out
	}
	for i := range result.Issues {
		issue := &result.Issues[i]
		out.Total++
		switch strings.ToLower(strings.TrimSpace(issue.Severity)) {
		case "critical":
			out.Critical++
		case "high":
			out.High++
		case "medium", "warning", "warn":
			out.Medium++
		case "low":
			out.Low++
		default:
			out.Info++
		}
		if issue.SafeForAutoPR && strings.EqualFold(issue.Fixable, "true") {
			out.AutoRemediationReady++
		}
		if ConfidenceNeedsHumanReview(issue.Confidence) || strings.EqualFold(issue.ReportingAction, "manual_review") {
			out.NeedsHumanTriage++
		}
	}
	switch {
	case out.Critical > 0, out.High > 0:
		out.Headline = "Action required"
	case out.Medium > 0:
		out.Headline = "Review recommended"
	case out.Total > 0:
		out.Headline = "Monitor"
	default:
		out.Headline = "Healthy"
	}
	return out
}

// ShouldCreatePostureIssue decides whether to file a rare forge posture issue.
// Finding volume alone is never enough — risk and regression drive the gate.
func ShouldCreatePostureIssue(cur RepositoryPosture, prev *RepositoryPosture, cfg PostureTriggerConfig) (bool, string) {
	if cfg.HighThreshold <= 0 {
		cfg.HighThreshold = DefaultPostureTriggerConfig().HighThreshold
	}
	if cfg.ActionableJumpThreshold <= 0 {
		cfg.ActionableJumpThreshold = DefaultPostureTriggerConfig().ActionableJumpThreshold
	}
	if cur.Critical > 0 {
		return true, "critical_present"
	}
	if cur.High >= cfg.HighThreshold {
		return true, "high_threshold"
	}
	if prev != nil {
		if cur.Critical > prev.Critical {
			return true, "critical_regression"
		}
		if cur.High > prev.High {
			return true, "high_regression"
		}
		jump := cur.actionable() - prev.actionable()
		if jump >= cfg.ActionableJumpThreshold {
			return true, "actionable_jump"
		}
	}
	return false, ""
}

// shouldCreateSummaryIssue wraps ShouldCreatePostureIssue for the manager (no prior scan → risk-only).
// A single Critical, or High count above threshold, is enough. Volume alone never gates.
func shouldCreateSummaryIssue(codeIssues []ai.CodeIssue) bool {
	cur := BuildRepositoryPosture(&ai.CodeAnalysisResult{Issues: codeIssues})
	ok, _ := ShouldCreatePostureIssue(cur, nil, DefaultPostureTriggerConfig())
	return ok
}

// EnrichPostureDelta fills new/resolved counts when a previous posture is available.
func EnrichPostureDelta(cur *RepositoryPosture, prev *RepositoryPosture) {
	if cur == nil || prev == nil {
		return
	}
	cur.NewSincePrevious = maxInt(0, cur.actionable()-prev.actionable())
	cur.ResolvedSincePrevious = maxInt(0, prev.actionable()-cur.actionable())
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// RenderRepositoryPostureMarkdown renders the operator-facing posture block.
func RenderRepositoryPostureMarkdown(p RepositoryPosture) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Repository posture: %s\n\n", p.Headline))
	parts := []string{}
	if p.Critical > 0 {
		parts = append(parts, fmt.Sprintf("%d Critical", p.Critical))
	}
	if p.High > 0 {
		parts = append(parts, fmt.Sprintf("%d High", p.High))
	}
	if p.Medium > 0 {
		parts = append(parts, fmt.Sprintf("%d Medium", p.Medium))
	}
	if p.Low > 0 {
		parts = append(parts, fmt.Sprintf("%d Low", p.Low))
	}
	if len(parts) == 0 {
		b.WriteString("No severity-tagged findings in this scan set.\n\n")
	} else {
		b.WriteString(strings.Join(parts, " · ") + "\n\n")
	}
	if p.NewSincePrevious > 0 {
		b.WriteString(fmt.Sprintf("- %d new since previous scan\n", p.NewSincePrevious))
	}
	if p.ResolvedSincePrevious > 0 {
		b.WriteString(fmt.Sprintf("- %d resolved since previous scan\n", p.ResolvedSincePrevious))
	}
	if p.ExcludedByCalibration > 0 {
		b.WriteString(fmt.Sprintf("- %d findings removed from the operator queue by accepted calibration\n", p.ExcludedByCalibration))
	}
	b.WriteString(fmt.Sprintf("- %d eligible for automated remediation\n", p.AutoRemediationReady))
	b.WriteString(fmt.Sprintf("- %d require human review\n", p.NeedsHumanTriage))
	b.WriteString(fmt.Sprintf("- %d total findings in this filing set\n", p.Total))
	if p.TriggerReason != "" {
		b.WriteString(fmt.Sprintf("\n_Escalation reason: `%s`_\n", p.TriggerReason))
	}
	return b.String()
}
