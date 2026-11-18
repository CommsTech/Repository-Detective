package store_test

import (
	"context"
	"testing"
)

func TestCalibrationSummaryIncludesMaturityFields(t *testing.T) {
	s := openTestStore(t)
	summary, err := s.CalibrationSummary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"proposed_recommendations", "accepted_recommendations", "pending_recommendations",
		"noisy_rules", "actionable_rules", "scanner_reliability", "recommendations_pending",
	} {
		if _, ok := summary[key]; !ok {
			t.Fatalf("missing calibration summary key %q", key)
		}
	}
}
