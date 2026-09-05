package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SaveReconciliationRun persists a reconciliation run and its items.
func (s *SQLiteStore) SaveReconciliationRun(ctx context.Context, run ReconciliationRun, items []ReconciliationItemRecord) error {
	if run.CreatedAt.IsZero() {
		run.CreatedAt = time.Now().UTC()
	}
	preview := 0
	if run.Preview {
		preview = 1
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("save reconciliation run: %w", err)
	}
	// defer Rollback is safe after Commit; ignored error is intentional on success path.
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO issue_reconciliation_runs (run_id, repository_id, preview, item_count, applied, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(run_id) DO UPDATE SET item_count = excluded.item_count, applied = excluded.applied
	`, run.RunID, run.RepositoryID, preview, run.ItemCount, run.Applied, run.CreatedAt.Format(time.RFC3339)); err != nil {
		return fmt.Errorf("save reconciliation run: %w", err)
	}
	if len(items) > 0 {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO issue_reconciliation_items (run_id, issue_number, finding_id, status, proposed_action, reason)
			VALUES (?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("save reconciliation items: %w", err)
		}
		defer stmt.Close()
		for _, rec := range items {
			if _, err := stmt.ExecContext(ctx, rec.RunID, rec.IssueNumber, rec.FindingID, rec.Status, rec.ProposedAction, rec.Reason); err != nil {
				return fmt.Errorf("save reconciliation item: %w", err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit reconciliation run: %w", err)
	}
	return nil
}

// GetReconciliationRun loads a reconciliation run with items.
func (s *SQLiteStore) GetReconciliationRun(ctx context.Context, runID string) (ReconciliationRun, []ReconciliationItemRecord, error) {
	var run ReconciliationRun
	var preview int
	var createdAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT run_id, repository_id, preview, item_count, applied, created_at
		FROM issue_reconciliation_runs WHERE run_id = ?
	`, runID).Scan(&run.RunID, &run.RepositoryID, &preview, &run.ItemCount, &run.Applied, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return ReconciliationRun{}, nil, fmt.Errorf("reconciliation run not found")
		}
		return ReconciliationRun{}, nil, err
	}
	run.Preview = preview == 1
	run.CreatedAt = parseTime(createdAt)

	rows, err := s.db.QueryContext(ctx, `
		SELECT run_id, issue_number, finding_id, status, proposed_action, reason
		FROM issue_reconciliation_items WHERE run_id = ? ORDER BY issue_number
	`, runID)
	if err != nil {
		return run, nil, err
	}
	defer rows.Close()
	var items []ReconciliationItemRecord
	for rows.Next() {
		var item ReconciliationItemRecord
		if err := rows.Scan(&item.RunID, &item.IssueNumber, &item.FindingID, &item.Status, &item.ProposedAction, &item.Reason); err != nil {
			return run, nil, err
		}
		items = append(items, item)
	}
	return run, items, rows.Err()
}

// RecomputeCalibrationRuleStats aggregates local evidence into rule stats.
func (s *SQLiteStore) RecomputeCalibrationRuleStats(ctx context.Context) (int, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			COALESCE(f.source, ''),
			COALESCE(f.rule_id, ''),
			COALESCE(f.category, ''),
			COUNT(1) AS total,
			SUM(CASE WHEN ei.finding_id IS NOT NULL THEN 1 ELSE 0 END) AS issues_created,
			SUM(CASE WHEN f.status = 'suppressed' THEN 1 ELSE 0 END) AS suppressions,
			SUM(CASE WHEN f.status = 'false_positive' THEN 1 ELSE 0 END) AS false_positives,
			SUM(CASE WHEN f.status = 'resolved_verified' THEN 1 ELSE 0 END) AS verified_fixes,
			SUM(CASE WHEN f.status = 'open' THEN 1 ELSE 0 END) AS still_present,
			MAX(f.last_seen_at) AS last_seen
		FROM findings f
		LEFT JOIN (
			SELECT DISTINCT finding_id FROM external_issues
		) ei ON ei.finding_id = f.id
		WHERE COALESCE(f.rule_id, '') != '' OR COALESCE(f.source, '') != ''
		GROUP BY f.source, f.rule_id, f.category
	`)
	if err != nil {
		return 0, fmt.Errorf("aggregate rule stats: %w", err)
	}
	defer rows.Close()

	type statRow struct {
		source, ruleID, category, lastSeen      string
		total, issues, sup, fp, verified, still int
		fpRate, actionableRate                  float64
		recAction                               string
	}
	var batch []statRow
	for rows.Next() {
		var row statRow
		if err := rows.Scan(&row.source, &row.ruleID, &row.category, &row.total, &row.issues, &row.sup, &row.fp, &row.verified, &row.still, &row.lastSeen); err != nil {
			return 0, err
		}
		if row.total > 0 {
			row.fpRate = float64(row.sup+row.fp) / float64(row.total)
			row.actionableRate = float64(row.verified) / float64(row.total)
		}
		row.recAction = "manual_review"
		if row.fpRate >= 0.5 {
			row.recAction = "report_only"
		} else if row.actionableRate >= 0.3 {
			row.recAction = "auto_issue"
		}
		batch = append(batch, row)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	// defer Rollback is safe after Commit; ignored error is intentional on success path.
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO calibration_rule_stats (
			source, rule_id, category, total_findings, issues_created, suppressions,
			false_positives, verified_fixes, still_present, last_seen_at,
			actionable_rate, false_positive_rate, recommended_default_action, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source, rule_id, category) DO UPDATE SET
			total_findings = excluded.total_findings,
			issues_created = excluded.issues_created,
			suppressions = excluded.suppressions,
			false_positives = excluded.false_positives,
			verified_fixes = excluded.verified_fixes,
			still_present = excluded.still_present,
			last_seen_at = excluded.last_seen_at,
			actionable_rate = excluded.actionable_rate,
			false_positive_rate = excluded.false_positive_rate,
			recommended_default_action = excluded.recommended_default_action,
			updated_at = excluded.updated_at
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	for _, row := range batch {
		if _, err := stmt.ExecContext(ctx, row.source, row.ruleID, row.category, row.total, row.issues, row.sup, row.fp, row.verified, row.still, row.lastSeen,
			row.actionableRate, row.fpRate, row.recAction, now); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(batch), nil
}

// ListCalibrationRuleStats returns aggregated rule statistics.
func (s *SQLiteStore) ListCalibrationRuleStats(ctx context.Context, limit int) ([]CalibrationRuleStat, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT source, rule_id, category, total_findings, issues_created, suppressions,
			false_positives, verified_fixes, still_present, last_seen_at,
			actionable_rate, false_positive_rate, recommended_default_action
		FROM calibration_rule_stats
		ORDER BY false_positive_rate DESC, total_findings DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCalibrationRuleStats(rows)
}

// GenerateCalibrationRecommendations creates proposed calibration changes from stats.
func (s *SQLiteStore) GenerateCalibrationRecommendations(ctx context.Context, minFindings int) (int, error) {
	if minFindings <= 0 {
		minFindings = 20
	}
	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := s.db.QueryContext(ctx, `
		SELECT source, rule_id, category, total_findings, false_positive_rate, recommended_default_action
		FROM calibration_rule_stats
		WHERE total_findings >= ? AND false_positive_rate >= 0.4
	`, minFindings)
	if err != nil {
		return 0, err
	}
	type candidate struct {
		source, ruleID, category, recAction string
		total                               int
		fpRate                              float64
	}
	var candidates []candidate
	for rows.Next() {
		var source, ruleID, category, recAction string
		var total int
		var fpRate float64
		if err := rows.Scan(&source, &ruleID, &category, &total, &fpRate, &recAction); err != nil {
			rows.Close()
			return 0, err
		}
		candidates = append(candidates, candidate{
			source: source, ruleID: ruleID, category: category, recAction: recAction,
			total: total, fpRate: fpRate,
		})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	count := 0
	for _, c := range candidates {
		reason := fmt.Sprintf("%d findings, %.0f%% false-positive/suppression rate", c.total, c.fpRate*100)
		var exists int
		if err := s.db.QueryRowContext(ctx, `
			SELECT COUNT(1) FROM calibration_recommendations
			WHERE scope = 'global' AND rule_id = ? AND source = ? AND status = 'proposed'
		`, c.ruleID, c.source).Scan(&exists); err != nil {
			return count, fmt.Errorf("check existing recommendation: %w", err)
		}
		if exists > 0 {
			continue
		}
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO calibration_recommendations (
				scope, repository_id, recommendation_type, source, rule_id, category,
				current_action, recommended_action, reason, confidence, status, created_at, updated_at
			) VALUES ('global', NULL, 'report_only', ?, ?, ?, 'auto_issue', ?, ?, ?, 'proposed', ?, ?)
		`, c.source, c.ruleID, c.category, c.recAction, reason, c.fpRate, now, now); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// ListCalibrationRecommendations lists calibration recommendations.
func (s *SQLiteStore) ListCalibrationRecommendations(ctx context.Context, status string, limit int) ([]CalibrationRecommendation, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
		SELECT id, scope, repository_id, recommendation_type, source, rule_id, category,
			current_action, recommended_action, reason, confidence, status, created_at, updated_at
		FROM calibration_recommendations WHERE 1=1`
	args := []any{}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY confidence DESC, updated_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CalibrationRecommendation
	for rows.Next() {
		var rec CalibrationRecommendation
		var repoID sql.NullInt64
		var createdAt, updatedAt string
		if err := rows.Scan(
			&rec.ID, &rec.Scope, &repoID, &rec.RecommendationType, &rec.Source, &rec.RuleID, &rec.Category,
			&rec.CurrentAction, &rec.RecommendedAction, &rec.Reason, &rec.Confidence, &rec.Status,
			&createdAt, &updatedAt,
		); err != nil {
			return nil, err
		}
		if repoID.Valid {
			id := repoID.Int64
			rec.RepositoryID = &id
		}
		rec.CreatedAt = parseTime(createdAt)
		rec.UpdatedAt = parseTime(updatedAt)
		out = append(out, rec)
	}
	return out, rows.Err()
}

// UpdateCalibrationRecommendationStatus sets recommendation status.
func (s *SQLiteStore) UpdateCalibrationRecommendationStatus(ctx context.Context, id int64, status string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		UPDATE calibration_recommendations SET status = ?, updated_at = ? WHERE id = ?
	`, status, now, id)
	return err
}

// CalibrationSummary aggregates calibration metrics for dashboard.
func (s *SQLiteStore) CalibrationSummary(ctx context.Context) (map[string]any, error) {
	out := map[string]any{}
	var proposed, accepted, rejected int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM calibration_recommendations WHERE status = 'proposed'`).Scan(&proposed); err != nil {
		return nil, fmt.Errorf("count proposed recommendations: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM calibration_recommendations WHERE status = 'accepted'`).Scan(&accepted); err != nil {
		return nil, fmt.Errorf("count accepted recommendations: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM calibration_recommendations WHERE status = 'rejected'`).Scan(&rejected); err != nil {
		return nil, fmt.Errorf("count rejected recommendations: %w", err)
	}
	out["proposed_recommendations"] = proposed
	out["accepted_recommendations"] = accepted
	out["rejected_recommendations"] = rejected
	out["pending_recommendations"] = proposed

	noisy, _ := s.ListCalibrationRuleStats(ctx, 10)
	out["noisy_rules"] = noisy
	actionable, _ := s.ListCalibrationRuleStatsByActionable(ctx, 10)
	out["actionable_rules"] = actionable
	reliability, _ := s.ScannerReliabilitySummary(ctx, 10)
	out["scanner_reliability"] = reliability
	pending, _ := s.ListCalibrationRecommendations(ctx, "proposed", 10)
	out["recommendations_pending"] = pending
	return out, nil
}

// ListCalibrationRuleStatsByActionable returns rules with high verified-fix rates.
func (s *SQLiteStore) ListCalibrationRuleStatsByActionable(ctx context.Context, limit int) ([]CalibrationRuleStat, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT source, rule_id, category, total_findings, issues_created, suppressions,
			false_positives, verified_fixes, still_present, last_seen_at,
			actionable_rate, false_positive_rate, recommended_default_action
		FROM calibration_rule_stats
		WHERE total_findings >= 5
		ORDER BY actionable_rate DESC, verified_fixes DESC, total_findings DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCalibrationRuleStats(rows)
}

// ScannerReliabilitySummary aggregates scanner failure counts for operator review.
func (s *SQLiteStore) ScannerReliabilitySummary(ctx context.Context, limit int) ([]ScannerStatusCount, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT scanner_name, status, COUNT(1) FROM scanner_results
		WHERE status IN ('failed', 'error', 'timeout', 'binary_missing', 'parse_failed', 'timed_out')
		GROUP BY scanner_name, status ORDER BY COUNT(1) DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ScannerStatusCount
	for rows.Next() {
		var sc ScannerStatusCount
		if err := rows.Scan(&sc.ScannerName, &sc.Status, &sc.Count); err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

func scanCalibrationRuleStats(rows *sql.Rows) ([]CalibrationRuleStat, error) {
	var out []CalibrationRuleStat
	for rows.Next() {
		var st CalibrationRuleStat
		var lastSeen string
		if err := rows.Scan(
			&st.Source, &st.RuleID, &st.Category, &st.TotalFindings, &st.IssuesCreated,
			&st.Suppressions, &st.FalsePositives, &st.VerifiedFixes, &st.StillPresent,
			&lastSeen, &st.ActionableRate, &st.FalsePositiveRate, &st.RecommendedDefaultAction,
		); err != nil {
			return nil, err
		}
		st.LastSeenAt = parseTime(lastSeen)
		out = append(out, st)
	}
	return out, rows.Err()
}
