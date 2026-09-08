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
	ExcludedByCalibration int // reserved for callers that know calibration counts
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
	case out.Critical > 0:
		out.Headline = "Action required"
	case out.High > 0:
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
	b.WriteString(fmt.Sprintf("- %d findings eligible for automated remediation\n", p.AutoRemediationReady))
	b.WriteString(fmt.Sprintf("- %d findings require human triage\n", p.NeedsHumanTriage))
	if p.ExcludedByCalibration > 0 {
		b.WriteString(fmt.Sprintf("- %d excluded by accepted calibration/report-only policy\n", p.ExcludedByCalibration))
	}
	b.WriteString(fmt.Sprintf("- %d total findings in this filing set\n", p.Total))
	return b.String()
}
