package analyzers

import (
	"testing"

	"git.commsnet.org/commstech/bugbot/ai"
)

func TestComputeOverallScoreEmpty(t *testing.T) {
	if got := ComputeOverallScore(nil); got != 1.0 {
		t.Fatalf("empty issues want 1.0, got %v", got)
	}
}

func TestComputeOverallScorePenalizesSeverity(t *testing.T) {
	clean := ComputeOverallScore([]ai.CodeIssue{{Severity: "low"}})
	critical := ComputeOverallScore([]ai.CodeIssue{{Severity: "critical"}})
	if critical >= clean {
		t.Fatalf("critical %v should be below low %v", critical, clean)
	}
}
