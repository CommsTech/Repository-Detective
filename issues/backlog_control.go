package issues

import (
	"strings"

	"git.commsnet.org/commstech/repository-detective/ai"
)

const BacklogControlNote = "New issue filing paused by backlog-control mode."

// BacklogControlConfig pauses low-priority new issue creation during dogfood burn-down.
type BacklogControlConfig struct {
	Enabled              bool
	MaxOpenIssues        int
	AllowNewIssueSeverity []string
	AllowMinConfidence   float64
	UpdateExistingOnly   bool
	AllowedSources       map[string]bool
	AllowedRuleIDs       map[string]bool
}

// DefaultBacklogControlConfig returns conservative defaults (disabled).
func DefaultBacklogControlConfig() BacklogControlConfig {
	return BacklogControlConfig{
		Enabled:              false,
		MaxOpenIssues:        0,
		AllowNewIssueSeverity: []string{"high", "critical"},
		AllowMinConfidence:   0.85,
		UpdateExistingOnly:   true,
	}
}

func normalizeSeverityList(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, v := range values {
		key := strings.ToLower(strings.TrimSpace(v))
		if key != "" {
			out[key] = true
		}
	}
	return out
}

// ShouldBlockNewIssue reports whether backlog control blocks creating a new forge issue.
// Existing issue updates and backfill are handled elsewhere and are never blocked here.
func (bc BacklogControlConfig) ShouldBlockNewIssue(issue *ai.CodeIssue, openIssueCount int) (blocked bool, reason string) {
	if !bc.Enabled || issue == nil {
		return false, ""
	}

	if bc.MaxOpenIssues > 0 && openIssueCount >= bc.MaxOpenIssues {
		sev := strings.ToLower(strings.TrimSpace(issue.Severity))
		allowed := normalizeSeverityList(bc.AllowNewIssueSeverity)
		if !allowed[sev] {
			return true, BacklogControlNote + " Open issue cap reached."
		}
	}

	if len(bc.AllowedSources) > 0 {
		src := strings.ToLower(strings.TrimSpace(issue.Source))
		if bc.AllowedSources[src] {
			return false, ""
		}
	}
	if len(bc.AllowedRuleIDs) > 0 {
		rule := strings.ToUpper(strings.TrimSpace(issue.RuleID))
		if bc.AllowedRuleIDs[rule] {
			return false, ""
		}
	}

	allowedSev := normalizeSeverityList(bc.AllowNewIssueSeverity)
	sev := strings.ToLower(strings.TrimSpace(issue.Severity))
	if !allowedSev[sev] {
		return true, BacklogControlNote + " Severity below backlog-control threshold."
	}

	minConf := bc.AllowMinConfidence
	if minConf <= 0 {
		minConf = 0.85
	}
	if issue.Confidence > 0 && issue.Confidence < minConf {
		return true, BacklogControlNote + " Confidence below backlog-control threshold."
	}

	return false, ""
}

// ShouldBlockSummaryIssue blocks rollup/summary tickets during backlog control.
func (bc BacklogControlConfig) ShouldBlockSummaryIssue() bool {
	return bc.Enabled
}
