package store

import (
	"context"
	"fmt"
)

func (s *SQLiteStore) ScanQualityReport(ctx context.Context) (ScanQualityReport, error) {
	var report ScanQualityReport
	report.FindingsBySeverity = map[string]int{}
	report.FindingsByCategory = map[string]int{}
	report.FindingsBySource = map[string]int{}

	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT repository_id) FROM scans WHERE status = 'completed'
	`).Scan(&report.ReposScanned); err != nil {
		return report, fmt.Errorf("repos scanned: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM findings`).Scan(&report.TotalFindings); err != nil {
		return report, fmt.Errorf("total findings: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM findings WHERE status = 'open'
	`).Scan(&report.OpenFindings); err != nil {
		return report, fmt.Errorf("open findings: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM findings WHERE status IN ('suppressed', 'false_positive')
	`).Scan(&report.SuppressedFindings); err != nil {
		return report, fmt.Errorf("suppressed findings: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM findings WHERE status = 'false_positive'
	`).Scan(&report.FalsePositiveFindings); err != nil {
		return report, fmt.Errorf("false positive findings: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM external_issues WHERE state = 'open'
	`).Scan(&report.ExternalIssuesOpen); err != nil {
		return report, fmt.Errorf("external issues: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM remediation_plans`).Scan(&report.RemediationPlansGenerated); err != nil {
		return report, fmt.Errorf("remediation plans: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM patch_attempts WHERE pull_request_number IS NOT NULL AND pull_request_number > 0
	`).Scan(&report.PatchAttemptsOpened); err != nil {
		return report, fmt.Errorf("patch attempts opened: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM closure_evidence WHERE status = 'resolved_verified'
	`).Scan(&report.PatchAttemptsVerified); err != nil {
		return report, fmt.Errorf("verified closures: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM scanner_results WHERE status IN ('failed', 'error', 'timeout')
	`).Scan(&report.ScannerFailures); err != nil {
		return report, fmt.Errorf("scanner failures: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM (
			SELECT repository_id FROM scans WHERE status = 'completed'
			GROUP BY repository_id
			HAVING SUM(CAST(json_extract(summary_json, '$.issues_found') AS INTEGER)) = 0
		)
	`).Scan(&report.ReposWithNoFindings); err != nil {
		return report, fmt.Errorf("repos with no findings: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT repository_id) FROM findings
		WHERE status = 'open' AND LOWER(severity) IN ('critical', 'high')
	`).Scan(&report.ReposWithCriticalHigh); err != nil {
		return report, fmt.Errorf("repos critical/high: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `SELECT LOWER(severity), COUNT(1) FROM findings GROUP BY LOWER(severity)`)
	if err != nil {
		return report, err
	}
	for rows.Next() {
		var k string
		var v int
		if err := rows.Scan(&k, &v); err != nil {
			rows.Close()
			return report, err
		}
		report.FindingsBySeverity[k] = v
	}
	rows.Close()

	catRows, err := s.db.QueryContext(ctx, `SELECT LOWER(category), COUNT(1) FROM findings GROUP BY LOWER(category)`)
	if err != nil {
		return report, err
	}
	for catRows.Next() {
		var k string
		var v int
		if err := catRows.Scan(&k, &v); err != nil {
			catRows.Close()
			return report, err
		}
		report.FindingsByCategory[k] = v
	}
	catRows.Close()

	srcRows, err := s.db.QueryContext(ctx, `SELECT LOWER(source), COUNT(1) FROM findings GROUP BY LOWER(source)`)
	if err != nil {
		return report, err
	}
	for srcRows.Next() {
		var k string
		var v int
		if err := srcRows.Scan(&k, &v); err != nil {
			srcRows.Close()
			return report, err
		}
		report.FindingsBySource[k] = v
	}
	srcRows.Close()

	chRow := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM findings WHERE status = 'open' AND LOWER(severity) IN ('critical', 'high')
	`)
	_ = chRow.Scan(&report.StrictActionableFindings)
	if report.OpenFindings > 0 {
		report.StrictActionableRatio = float64(report.StrictActionableFindings) / float64(report.OpenFindings)
	}
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM findings WHERE status = 'open' AND source = 'graph'
	`).Scan(&report.GraphFindingsOpen)
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM findings WHERE status = 'open' AND source = 'graph'
		AND LOWER(severity) NOT IN ('critical', 'high')
	`).Scan(&report.ReportOnlyEstimate)

	ruleRows, err := s.db.QueryContext(ctx, `
		SELECT rule_id, source, COUNT(1) AS cnt FROM findings
		WHERE status = 'open' AND rule_id != '' GROUP BY rule_id, source ORDER BY cnt DESC LIMIT 10
	`)
	if err == nil {
		for ruleRows.Next() {
			var rc RuleCount
			if err := ruleRows.Scan(&rc.RuleID, &rc.Source, &rc.Count); err == nil {
				report.TopNoisyRules = append(report.TopNoisyRules, rc)
			}
		}
		ruleRows.Close()
	}
	supRows, err := s.db.QueryContext(ctx, `
		SELECT rule_id, COALESCE(source,''), COUNT(1) FROM finding_suppressions
		WHERE active = 1 AND rule_id != '' GROUP BY rule_id, source ORDER BY COUNT(1) DESC LIMIT 10
	`)
	if err == nil {
		for supRows.Next() {
			var rc RuleCount
			if err := supRows.Scan(&rc.RuleID, &rc.Source, &rc.Count); err == nil {
				report.TopSuppressedRules = append(report.TopSuppressedRules, rc)
			}
		}
		supRows.Close()
	}
	failRows, err := s.db.QueryContext(ctx, `
		SELECT scanner_name, status, COUNT(1) FROM scanner_results
		WHERE status IN ('failed', 'error', 'timeout', 'binary_missing', 'parse_failed', 'timed_out')
		GROUP BY scanner_name, status ORDER BY COUNT(1) DESC LIMIT 20
	`)
	if err == nil {
		for failRows.Next() {
			var sc ScannerStatusCount
			if err := failRows.Scan(&sc.ScannerName, &sc.Status, &sc.Count); err == nil {
				report.ScannerFailureBreakdown = append(report.ScannerFailureBreakdown, sc)
			}
		}
		failRows.Close()
	}

	report.ActionableFindings = report.OpenFindings
	if report.TotalFindings > 0 {
		report.ActionableRatio = float64(report.StrictActionableFindings) / float64(report.TotalFindings)
	}
	return report, nil
}
