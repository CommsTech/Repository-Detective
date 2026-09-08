package store

import (
	"context"
	"fmt"
	"time"
)

// NoiseReductionSummary powers the dashboard "Noise Reduction" card (RD-PRODUCT-002).
type NoiseReductionSummary struct {
	WindowDays int `json:"window_days"`

	RawFindings            int `json:"raw_findings"`
	StructurallyGrouped    int `json:"structurally_grouped"`
	CalibratedNoise        int `json:"calibrated_noise"`
	ActionablePresented    int `json:"actionable_presented"`
	KeptOutOfQueue         int `json:"kept_out_of_queue"`
	CriticalAutoSuppressed int `json:"critical_auto_suppressed"`

	FindingsPerScanBefore float64 `json:"findings_per_scan_before"`
	FindingsPerScanAfter  float64 `json:"findings_per_scan_after"`
	WorkloadReductionPct  float64 `json:"workload_reduction_pct"`

	Headline string `json:"headline"`
	Subline  string `json:"subline"`
}

// NoiseReductionSummary computes operator-facing noise reduction proof metrics.
func (s *SQLiteStore) NoiseReductionSummary(ctx context.Context, windowDays int) (NoiseReductionSummary, error) {
	if windowDays <= 0 {
		windowDays = 30
	}
	out := NoiseReductionSummary{WindowDays: windowDays}
	since := time.Now().UTC().Add(-time.Duration(windowDays) * 24 * time.Hour).Format(time.RFC3339)

	// Raw = finding_instances created in window (pre-dedup volume).
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM finding_instances WHERE created_at >= ?
	`, since).Scan(&out.RawFindings); err != nil {
		return out, fmt.Errorf("raw findings: %w", err)
	}

	// Structurally grouped fingerprints (open unique findings).
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM findings WHERE status = 'open' AND suppressed = 0
	`).Scan(&out.StructurallyGrouped); err != nil {
		// fallback without suppressed column shape
		_ = s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM findings WHERE status = 'open'`).Scan(&out.StructurallyGrouped)
	}

	// Calibrated / suppressed noise.
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM findings WHERE suppressed = 1
	`).Scan(&out.CalibratedNoise); err != nil {
		out.CalibratedNoise = 0
	}

	// Actionable = open + not suppressed.
	out.ActionablePresented = out.StructurallyGrouped
	if out.RawFindings > out.ActionablePresented {
		out.KeptOutOfQueue = out.RawFindings - out.ActionablePresented
	}

	// Critical never auto-suppressed (product promise).
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM findings
		WHERE suppressed = 1 AND LOWER(severity) = 'critical'
	`).Scan(&out.CriticalAutoSuppressed)

	var scanCount int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM scans WHERE status IN ('completed','complete','success') AND completed_at >= ?
	`, since).Scan(&scanCount)
	if scanCount <= 0 {
		_ = s.db.QueryRowContext(ctx, `
			SELECT COUNT(1) FROM scans WHERE started_at >= ?
		`, since).Scan(&scanCount)
	}

	if scanCount > 0 {
		out.FindingsPerScanBefore = float64(out.RawFindings) / float64(scanCount)
		out.FindingsPerScanAfter = float64(out.ActionablePresented) / float64(scanCount)
		if out.FindingsPerScanBefore > 0 {
			out.WorkloadReductionPct = (1.0 - (out.FindingsPerScanAfter / out.FindingsPerScanBefore)) * 100.0
			if out.WorkloadReductionPct < 0 {
				out.WorkloadReductionPct = 0
			}
		}
	}

	out.Headline = fmt.Sprintf("%s repetitive/noisy records kept out of the operator queue", formatIntComma(out.KeptOutOfQueue))
	out.Subline = fmt.Sprintf("%d critical findings automatically suppressed", out.CriticalAutoSuppressed)
	return out, nil
}

func formatIntComma(n int) string {
	if n < 0 {
		n = 0
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	for i, c := range reverseString(s) {
		if i > 0 && i%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	return reverseString(string(out))
}

func reverseString(s string) string {
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}
