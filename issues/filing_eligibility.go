package issues

import (
	"fmt"
	"math"
	"strings"

	"git.commsnet.org/commstech/repository-detective/ai"
)

// EligibleForAutomaticFiling decides whether a finding may reach a forge.
// Confidence is a scanner/AI estimate, not proof of truth. This gate reduces
// accidental low-evidence issue spam; it is not a substitute for policy,
// lifecycle deduplication, redaction or human review of external disclosures.
func EligibleForAutomaticFiling(issue *ai.CodeIssue, minimumConfidence float64) (bool, string) {
	if issue == nil {
		return false, "finding is missing"
	}
	if minimumConfidence <= 0 {
		minimumConfidence = 0.5
	}
	if minimumConfidence > 1 {
		return false, "invalid confidence threshold"
	}
	if math.IsNaN(issue.Confidence) || math.IsInf(issue.Confidence, 0) || issue.Confidence < minimumConfidence || issue.Confidence > 1 || issue.Confidence <= 0 {
		return false, fmt.Sprintf("confidence %.2f does not meet gate %.2f", issue.Confidence, minimumConfidence)
	}
	if strings.TrimSpace(issue.Title) == "" || strings.TrimSpace(issue.Category) == "" {
		return false, "finding title or category missing"
	}
	if issue.FromAI {
		// Autonomous AI claims need more evidence than deterministic results.
		if issue.Confidence < 0.90 {
			return false, "AI finding below 0.90 corroboration threshold"
		}
		if strings.TrimSpace(issue.File) == "" || issue.LineNumber <= 0 {
			return false, "AI finding missing source location"
		}
		if strings.TrimSpace(issue.Description) == "" || (strings.TrimSpace(issue.Evidence) == "" && strings.TrimSpace(issue.CodeSnippet) == "") {
			return false, "AI finding missing explanation or supporting evidence"
		}
	}
	return true, "eligible for configured issue policy and forge deduplication"
}
