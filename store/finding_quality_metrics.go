package store

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// FindingQualityWindow selects a local metrics horizon (RD-024).
type FindingQualityWindow string

const (
	FindingQualityWindow7d  FindingQualityWindow = "7d"
	FindingQualityWindow30d FindingQualityWindow = "30d"
	FindingQualityWindowAll FindingQualityWindow = "all"
)

// ParseFindingQualityWindow maps query values to a supported window.
func ParseFindingQualityWindow(raw string) FindingQualityWindow {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "7d", "7", "week":
		return FindingQualityWindow7d
	case "30d", "30", "month":
		return FindingQualityWindow30d
	default:
		return FindingQualityWindowAll
	}
}

func (w FindingQualityWindow) since() *time.Time {
	now := time.Now().UTC()
	switch w {
	case FindingQualityWindow7d:
		t := now.Add(-7 * 24 * time.Hour)
		return &t
	case FindingQualityWindow30d:
		t := now.Add(-30 * 24 * time.Hour)
		return &t
	default:
		return nil
	}
}

// FindingQualityMetrics are operator-facing usefulness counters derived only from
// durable local SQLite data. No external telemetry.
type FindingQualityMetrics struct {
	Window FindingQualityWindow `json:"window"`

	FindingsOpened   int `json:"findings_opened"`
	FindingsResolved int `json:"findings_resolved"`
	FindingsReopened int `json:"findings_reopened"`

	FindingsBySeverity map[string]int `json:"findings_by_severity"`
	FindingsByScanner  map[string]int `json:"findings_by_scanner"`
	FindingsByCategory map[string]int `json:"findings_by_category"`

	NewFindings       int `json:"new_findings"`
	RecurringFindings int `json:"recurring_findings"`

	FalsePositiveDispositions int `json:"false_positive_dispositions"`
	// FalsePositiveDispositionRate = dispositions / (dispositions + resolved + still_open_reviewed_proxy).
	// Denominator is reviewed-like outcomes only — NOT all findings. Do not label this "false-positive rate".
	FalsePositiveDispositionRate float64 `json:"false_positive_disposition_rate"`
	FalsePositiveDispositionNote string  `json:"false_positive_disposition_note"`

	MedianTimeToResolutionHours *float64 `json:"median_time_to_resolution_hours,omitempty"`

	ScannerRunsCompleted  int     `json:"scanner_runs_completed"`
	ScannerRunsFailed     int     `json:"scanner_runs_failed"`
	ScannerCompletionRate float64 `json:"scanner_completion_rate"`

	PolicyOutcomes map[string]int `json:"policy_outcomes,omitempty"`

	RepeatedFindingsSuppressed int `json:"repeated_findings_suppressed"`

	CalibrationProposals int `json:"calibration_proposals"`
	CalibrationAccepted  int `json:"calibration_accepted"`
	CalibrationRejected  int `json:"calibration_rejected"`
	CalibrationReverted  int `json:"calibration_reverted"`

	Definitions map[string]string `json:"definitions"`
}

// FindingQualityMetrics computes local finding-quality metrics for a time window.
func (s *SQLiteStore) FindingQualityMetrics(ctx context.Context, window FindingQualityWindow) (FindingQualityMetrics, error) {
	out := FindingQualityMetrics{
		Window:             window,
		FindingsBySeverity: map[string]int{},
		FindingsByScanner:  map[string]int{},
		FindingsByCategory: map[string]int{},
		PolicyOutcomes:     map[string]int{},
		Definitions: map[string]string{
			"findings_opened":                 "Count of findings whose first_seen_at falls in the window (all-time: all rows).",
			"findings_resolved":               "Count of findings with status resolved_verified (and last_seen_at in window when windowed).",
			"findings_reopened":               "Count of lifecycle_events with event_type still_present or reopen-like messages in the window.",
			"new_findings":                    "Findings with exactly one distinct scan instance (or first_seen_scan_id == last_seen_scan_id).",
			"recurring_findings":              "Findings seen across more than one scan id.",
			"false_positive_disposition_rate": "false_positive_dispositions / (false_positive_dispositions + findings_resolved). Unreviewed open findings are excluded from the denominator.",
			"scanner_completion_rate":         "completed-like scanner_results / (completed-like + failed-like) joined through scans in window.",
			"median_time_to_resolution_hours": "Median of (resolved lifecycle timestamp − first_seen_at) for resolved_verified findings with usable timestamps.",
		},
		FalsePositiveDispositionNote: "Not a true-positive/false-positive rate. Unreviewed findings are neither.",
	}

	since := window.since()
	findingTimeFilter := ""
	var args []any
	if since != nil {
		findingTimeFilter = " AND first_seen_at >= ? "
		args = append(args, since.Format(time.RFC3339))
	}

	qOpened := `SELECT COUNT(1) FROM findings WHERE 1=1` + findingTimeFilter
	if err := s.db.QueryRowContext(ctx, qOpened, args...).Scan(&out.FindingsOpened); err != nil {
		return out, fmt.Errorf("findings opened: %w", err)
	}

	resolvedFilter := ` status = 'resolved_verified' `
	resolvedArgs := []any{}
	if since != nil {
		resolvedFilter += ` AND last_seen_at >= ? `
		resolvedArgs = append(resolvedArgs, since.Format(time.RFC3339))
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM findings WHERE `+resolvedFilter, resolvedArgs...).Scan(&out.FindingsResolved); err != nil {
		return out, fmt.Errorf("findings resolved: %w", err)
	}

	reopenQ := `SELECT COUNT(1) FROM lifecycle_events WHERE event_type IN ('still_present','reopened','finding_reopened')`
	reopenArgs := []any{}
	if since != nil {
		reopenQ += ` AND created_at >= ?`
		reopenArgs = append(reopenArgs, since.Format(time.RFC3339))
	}
	_ = s.db.QueryRowContext(ctx, reopenQ, reopenArgs...).Scan(&out.FindingsReopened)

	if err := scanKeyedCounts(ctx, s, `SELECT LOWER(severity), COUNT(1) FROM findings WHERE 1=1`+findingTimeFilter+` GROUP BY LOWER(severity)`, args, out.FindingsBySeverity); err != nil {
		return out, err
	}
	if err := scanKeyedCounts(ctx, s, `SELECT LOWER(source), COUNT(1) FROM findings WHERE 1=1`+findingTimeFilter+` GROUP BY LOWER(source)`, args, out.FindingsByScanner); err != nil {
		return out, err
	}
	if err := scanKeyedCounts(ctx, s, `SELECT LOWER(category), COUNT(1) FROM findings WHERE 1=1`+findingTimeFilter+` GROUP BY LOWER(category)`, args, out.FindingsByCategory); err != nil {
		return out, err
	}

	newQ := `SELECT COUNT(1) FROM findings WHERE first_seen_scan_id = last_seen_scan_id` + findingTimeFilter
	_ = s.db.QueryRowContext(ctx, newQ, args...).Scan(&out.NewFindings)
	recQ := `SELECT COUNT(1) FROM findings WHERE first_seen_scan_id != last_seen_scan_id AND last_seen_scan_id != ''` + findingTimeFilter
	_ = s.db.QueryRowContext(ctx, recQ, args...).Scan(&out.RecurringFindings)

	fpQ := `SELECT COUNT(1) FROM findings WHERE status = 'false_positive'`
	fpArgs := []any{}
	if since != nil {
		fpQ += ` AND last_seen_at >= ?`
		fpArgs = append(fpArgs, since.Format(time.RFC3339))
	}
	_ = s.db.QueryRowContext(ctx, fpQ, fpArgs...).Scan(&out.FalsePositiveDispositions)
	denom := out.FalsePositiveDispositions + out.FindingsResolved
	if denom > 0 {
		out.FalsePositiveDispositionRate = float64(out.FalsePositiveDispositions) / float64(denom)
	}

	// Median TTR from first_seen_at → resolved_verified last_seen_at (best durable proxy).
	ttrRows, err := s.db.QueryContext(ctx, `
		SELECT first_seen_at, last_seen_at FROM findings
		WHERE status = 'resolved_verified' AND first_seen_at != '' AND last_seen_at != ''`)
	if err == nil {
		var hours []float64
		for ttrRows.Next() {
			var firstS, lastS string
			if err := ttrRows.Scan(&firstS, &lastS); err != nil {
				continue
			}
			first := parseTime(firstS)
			last := parseTime(lastS)
			if first.IsZero() || last.IsZero() || !last.After(first) {
				continue
			}
			if since != nil && last.Before(*since) {
				continue
			}
			hours = append(hours, last.Sub(first).Hours())
		}
		ttrRows.Close()
		if len(hours) > 0 {
			sort.Float64s(hours)
			med := hours[len(hours)/2]
			if len(hours)%2 == 0 {
				med = (hours[len(hours)/2-1] + hours[len(hours)/2]) / 2
			}
			med = math.Round(med*10) / 10
			out.MedianTimeToResolutionHours = &med
		}
	}

	// Scanner completion via scanner_results (+ optional scan time window via scans.started_at).
	scanJoin := `
		SELECT
			SUM(CASE WHEN sr.status IN ('ok','success','completed','passed','clean','found') THEN 1 ELSE 0 END),
			SUM(CASE WHEN sr.status IN ('failed','error','timeout','timed_out','binary_missing','parse_failed') THEN 1 ELSE 0 END)
		FROM scanner_results sr`
	scanArgs := []any{}
	if since != nil {
		scanJoin = `
		SELECT
			SUM(CASE WHEN sr.status IN ('ok','success','completed','passed','clean','found') THEN 1 ELSE 0 END),
			SUM(CASE WHEN sr.status IN ('failed','error','timeout','timed_out','binary_missing','parse_failed') THEN 1 ELSE 0 END)
		FROM scanner_results sr
		INNER JOIN scans sc ON sc.id = sr.scan_id
		WHERE sc.started_at >= ?`
		scanArgs = append(scanArgs, since.Format(time.RFC3339))
	}
	var completed, failed int
	_ = s.db.QueryRowContext(ctx, scanJoin, scanArgs...).Scan(&completed, &failed)
	out.ScannerRunsCompleted = completed
	out.ScannerRunsFailed = failed
	if completed+failed > 0 {
		out.ScannerCompletionRate = float64(completed) / float64(completed+failed)
	}

	// Policy outcomes if stored on scans.summary_json
	polQ := `
		SELECT COALESCE(json_extract(summary_json, '$.policy_outcome'), json_extract(summary_json, '$.policyOutcome'), '') AS outcome, COUNT(1)
		FROM scans WHERE status = 'completed'`
	polArgs := []any{}
	if since != nil {
		polQ += ` AND started_at >= ?`
		polArgs = append(polArgs, since.Format(time.RFC3339))
	}
	polQ += ` GROUP BY outcome`
	if rows, err := s.db.QueryContext(ctx, polQ, polArgs...); err == nil {
		for rows.Next() {
			var k string
			var v int
			if err := rows.Scan(&k, &v); err == nil && strings.TrimSpace(k) != "" {
				out.PolicyOutcomes[k] = v
			}
		}
		rows.Close()
	}

	supQ := `SELECT COUNT(1) FROM findings WHERE status = 'suppressed'`
	supArgs := []any{}
	if since != nil {
		supQ += ` AND last_seen_at >= ?`
		supArgs = append(supArgs, since.Format(time.RFC3339))
	}
	_ = s.db.QueryRowContext(ctx, supQ, supArgs...).Scan(&out.RepeatedFindingsSuppressed)

	calQ := func(status string) int {
		q := `SELECT COUNT(1) FROM calibration_recommendations WHERE status = ?`
		a := []any{status}
		if since != nil {
			q += ` AND updated_at >= ?`
			a = append(a, since.Format(time.RFC3339))
		}
		var n int
		_ = s.db.QueryRowContext(ctx, q, a...).Scan(&n)
		return n
	}
	out.CalibrationProposals = calQ("proposed")
	out.CalibrationAccepted = calQ("accepted")
	out.CalibrationRejected = calQ("rejected")
	out.CalibrationReverted = calQ("reverted")

	return out, nil
}

func scanKeyedCounts(ctx context.Context, s *SQLiteStore, query string, args []any, dest map[string]int) error {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		var v int
		if err := rows.Scan(&k, &v); err != nil {
			return err
		}
		if k == "" {
			k = "(empty)"
		}
		dest[k] = v
	}
	return rows.Err()
}
