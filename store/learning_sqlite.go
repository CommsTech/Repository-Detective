package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// RecordLearningEvent appends an idempotent learning event.
func (s *SQLiteStore) RecordLearningEvent(ctx context.Context, ev LearningEvent) (LearningEvent, error) {
	if ev.RepositoryID <= 0 {
		return LearningEvent{}, fmt.Errorf("repository_id required")
	}
	if strings.TrimSpace(ev.EventType) == "" {
		return LearningEvent{}, fmt.Errorf("event_type required")
	}
	ev.RuleID = NormalizeLearningRuleID(ev.Source, ev.RuleID)
	if ev.IdempotencyKey == "" {
		ev.IdempotencyKey = fmt.Sprintf("%d:%s:%s:%v", ev.RepositoryID, ev.EventType, ev.ScanID, ev.FindingID)
	}
	now := time.Now().UTC()
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = now
	}
	if len(ev.EvidenceJSON) == 0 {
		ev.EvidenceJSON = json.RawMessage(`{}`)
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO learning_events (
			repository_id, scan_id, finding_id, fingerprint, source, rule_id,
			event_type, evidence_json, created_at, created_by, confidence_delta, idempotency_key
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, ev.RepositoryID, ev.ScanID, nullInt64Ptr(ev.FindingID), ev.Fingerprint, ev.Source, ev.RuleID,
		ev.EventType, string(ev.EvidenceJSON), ev.CreatedAt.Format(time.RFC3339), ev.CreatedBy, ev.ConfidenceDelta, ev.IdempotencyKey)
	if err != nil {
		return LearningEvent{}, fmt.Errorf("record learning event: %w", err)
	}
	id, _ := res.LastInsertId()
	if id == 0 {
		return ev, nil
	}
	ev.ID = id
	_ = s.touchRuleReliabilityFromEvent(ctx, ev)
	return ev, nil
}

func nullInt64Ptr(v *int64) sql.NullInt64 {
	if v == nil || *v <= 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

func (s *SQLiteStore) touchRuleReliabilityFromEvent(ctx context.Context, ev LearningEvent) error {
	if ev.Source == "" && ev.RuleID == "" {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	var tp, fp, rv, dup, reapp int
	switch ev.EventType {
	case "user_marked_false_positive":
		fp = 1
	case "user_marked_true_positive", "resolved_verified":
		tp = 1
		if ev.EventType == "resolved_verified" {
			rv = 1
		}
	case "duplicate_linked":
		dup = 1
	case "finding_reappeared":
		reapp = 1
	case "scanner_failed":
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO rule_reliability_stats (
				repository_id, source, rule_id, scanner_failure_count, last_seen_at
			) VALUES (?, ?, ?, 1, ?)
			ON CONFLICT(repository_id, source, rule_id) DO UPDATE SET
				scanner_failure_count = scanner_failure_count + 1,
				last_seen_at = excluded.last_seen_at
		`, ev.RepositoryID, ev.Source, ev.RuleID, now)
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO rule_reliability_stats (
			repository_id, source, rule_id, findings_seen, true_positive_count, false_positive_count,
			resolved_verified_count, duplicate_count, reappeared_count, last_seen_at
		) VALUES (?, ?, ?, 1, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repository_id, source, rule_id) DO UPDATE SET
			findings_seen = findings_seen + 1,
			true_positive_count = true_positive_count + excluded.true_positive_count,
			false_positive_count = false_positive_count + excluded.false_positive_count,
			resolved_verified_count = resolved_verified_count + excluded.resolved_verified_count,
			duplicate_count = duplicate_count + excluded.duplicate_count,
			reappeared_count = reappeared_count + excluded.reappeared_count,
			last_seen_at = excluded.last_seen_at
	`, ev.RepositoryID, ev.Source, ev.RuleID, tp, fp, rv, dup, reapp, now)
	return err
}

// ListLearningEvents returns recent learning events for a repository.
func (s *SQLiteStore) ListLearningEvents(ctx context.Context, repositoryID int64, limit int) ([]LearningEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, repository_id, scan_id, finding_id, fingerprint, source, rule_id,
			event_type, evidence_json, created_at, created_by, confidence_delta, idempotency_key
		FROM learning_events WHERE repository_id = ? ORDER BY created_at DESC LIMIT ?
	`, repositoryID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLearningEvents(rows)
}

// CountLearningEventsByType returns fleet-wide learning event counts keyed by event_type.
func (s *SQLiteStore) CountLearningEventsByType(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT COALESCE(NULLIF(TRIM(event_type), ''), 'unknown') AS event_type, COUNT(1)
		FROM learning_events
		GROUP BY 1
		ORDER BY COUNT(1) DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("count learning events by type: %w", err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var typ string
		var n int
		if err := rows.Scan(&typ, &n); err != nil {
			return nil, fmt.Errorf("scan learning event type row: %w", err)
		}
		out[typ] = n
	}
	return out, rows.Err()
}

func scanLearningEvents(rows *sql.Rows) ([]LearningEvent, error) {
	var out []LearningEvent
	for rows.Next() {
		var ev LearningEvent
		var fid sql.NullInt64
		var created string
		var evidence string
		if err := rows.Scan(
			&ev.ID, &ev.RepositoryID, &ev.ScanID, &fid, &ev.Fingerprint, &ev.Source, &ev.RuleID,
			&ev.EventType, &evidence, &created, &ev.CreatedBy, &ev.ConfidenceDelta, &ev.IdempotencyKey,
		); err != nil {
			return nil, err
		}
		if fid.Valid {
			v := fid.Int64
			ev.FindingID = &v
		}
		ev.EvidenceJSON = json.RawMessage(evidence)
		ev.CreatedAt = parseTime(created)
		out = append(out, ev)
	}
	return out, rows.Err()
}

// RecordScannerHealth persists scanner run metadata for learning.
func (s *SQLiteStore) RecordScannerHealth(ctx context.Context, rec ScannerHealthRecord) error {
	if rec.RepositoryID <= 0 || rec.ScanID == "" || rec.Scanner == "" {
		return nil
	}
	now := time.Now().UTC()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO scanner_health_history (
			repository_id, scan_id, scanner, status, version, duration_ms, finding_count, error_class, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, rec.RepositoryID, rec.ScanID, rec.Scanner, rec.Status, rec.Version, rec.DurationMs, rec.FindingCount, rec.ErrorClass, rec.CreatedAt.Format(time.RFC3339))
	return err
}

// CreateRepoCalibrationRule persists an approved repo-scoped calibration rule.
func (s *SQLiteStore) CreateRepoCalibrationRule(ctx context.Context, rule RepoCalibrationRule) (RepoCalibrationRule, error) {
	now := time.Now().UTC()
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = now
	}
	rule.UpdatedAt = rule.CreatedAt
	if rule.Scope == "" {
		rule.Scope = SuppressionScopeRepo
	}
	expires := ""
	if rule.ExpiresAt != nil {
		expires = rule.ExpiresAt.UTC().Format(time.RFC3339)
	}
	active := 0
	if rule.Active {
		active = 1
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO repo_calibration_rules (
			repository_id, project_group_id, scope, source, rule_id, path_pattern, finding_category,
			action, reason, evidence_count, false_positive_rate, true_positive_rate, duplicate_rate,
			expires_at, active, created_at, updated_at, recommendation_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, nullInt64Ptr(rule.RepositoryID), nullInt64Ptr(rule.ProjectGroupID), rule.Scope, rule.Source, rule.RuleID,
		rule.PathPattern, rule.FindingCategory, rule.Action, rule.Reason, rule.EvidenceCount,
		rule.FalsePositiveRate, rule.TruePositiveRate, rule.DuplicateRate, nullString(expires), active,
		rule.CreatedAt.Format(time.RFC3339), rule.UpdatedAt.Format(time.RFC3339), nullInt64Ptr(rule.RecommendationID))
	if err != nil {
		return RepoCalibrationRule{}, err
	}
	id, _ := res.LastInsertId()
	rule.ID = id
	rule.Active = active == 1
	return rule, nil
}

// ListRepoCalibrationRules lists active calibration rules for a repository.
func (s *SQLiteStore) ListRepoCalibrationRules(ctx context.Context, repositoryID int64, activeOnly bool) ([]RepoCalibrationRule, error) {
	q := `SELECT id, repository_id, project_group_id, scope, source, rule_id, path_pattern, finding_category,
		action, reason, evidence_count, false_positive_rate, true_positive_rate, duplicate_rate,
		expires_at, active, created_at, updated_at, recommendation_id
		FROM repo_calibration_rules WHERE repository_id = ?`
	if activeOnly {
		q += ` AND active = 1`
	}
	q += ` ORDER BY updated_at DESC`
	rows, err := s.db.QueryContext(ctx, q, repositoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RepoCalibrationRule
	for rows.Next() {
		var r RepoCalibrationRule
		var repoID, pgID, recID sql.NullInt64
		var expires, created, updated sql.NullString
		var active int
		if err := rows.Scan(
			&r.ID, &repoID, &pgID, &r.Scope, &r.Source, &r.RuleID, &r.PathPattern, &r.FindingCategory,
			&r.Action, &r.Reason, &r.EvidenceCount, &r.FalsePositiveRate, &r.TruePositiveRate, &r.DuplicateRate,
			&expires, &active, &created, &updated, &recID,
		); err != nil {
			return nil, err
		}
		if repoID.Valid {
			v := repoID.Int64
			r.RepositoryID = &v
		}
		if pgID.Valid {
			v := pgID.Int64
			r.ProjectGroupID = &v
		}
		if recID.Valid {
			v := recID.Int64
			r.RecommendationID = &v
		}
		if expires.Valid {
			t := parseTime(expires.String)
			r.ExpiresAt = &t
		}
		r.Active = active == 1
		r.CreatedAt = parseTime(created.String)
		r.UpdatedAt = parseTime(updated.String)
		out = append(out, r)
	}
	return out, rows.Err()
}

// ExpireRepoCalibrationRule deactivates a calibration rule.
func (s *SQLiteStore) ExpireRepoCalibrationRule(ctx context.Context, ruleID int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		UPDATE repo_calibration_rules SET active = 0, expires_at = ?, updated_at = ? WHERE id = ?
	`, now, now, ruleID)
	return err
}

// GenerateRepoScopedRecommendations proposes calibration changes per repository.
func (s *SQLiteStore) GenerateRepoScopedRecommendations(ctx context.Context, repositoryID int64, minFindings int) (int, error) {
	if minFindings <= 0 {
		minFindings = 5
	}
	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := s.db.QueryContext(ctx, `
		SELECT source, rule_id,
			SUM(CASE WHEN event_type IN ('user_marked_false_positive') THEN 1 ELSE 0 END) AS fp,
			SUM(CASE WHEN event_type IN ('resolved_verified','user_marked_true_positive') THEN 1 ELSE 0 END) AS tp,
			COUNT(1) AS total
		FROM learning_events
		WHERE repository_id = ? AND (source != '' OR rule_id != '')
		GROUP BY source, rule_id
		HAVING total >= 3
	`, repositoryID)
	if err != nil {
		return 0, err
	}
	type candidate struct {
		source, ruleID string
		fp, total      int
		fpRate         float64
	}
	var candidates []candidate
	for rows.Next() {
		var source, ruleID string
		var fp, tp, total int
		if err := rows.Scan(&source, &ruleID, &fp, &tp, &total); err != nil {
			rows.Close()
			return 0, err
		}
		fpRate := float64(fp) / float64(total)
		if fpRate < 0.5 {
			continue
		}
		if total < repoScopedMinEvents(source, minFindings) {
			continue
		}
		candidates = append(candidates, candidate{source: source, ruleID: ruleID, fp: fp, total: total, fpRate: fpRate})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	count := 0
	for _, c := range candidates {
		var exists int
		if err := s.db.QueryRowContext(ctx, `
			SELECT COUNT(1) FROM calibration_recommendations
			WHERE scope = 'repo' AND repository_id = ? AND rule_id = ? AND source = ? AND status = 'proposed'
		`, repositoryID, c.ruleID, c.source).Scan(&exists); err != nil {
			return count, err
		}
		if exists > 0 {
			continue
		}
		reason := fmt.Sprintf("Repo %d: %d events, %.0f%% marked false positive (deterministic learning)", repositoryID, c.total, c.fpRate*100)
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO calibration_recommendations (
				scope, repository_id, recommendation_type, source, rule_id, category,
				current_action, recommended_action, reason, confidence, status, created_at, updated_at
			) VALUES ('repo', ?, 'downgrade_confidence', ?, ?, '', 'auto_issue', 'report_only', ?, ?, 'proposed', ?, ?)
		`, repositoryID, c.source, c.ruleID, reason, c.fpRate, now, now); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func repoScopedMinEvents(source string, configuredMin int) int {
	if configuredMin <= 0 {
		configuredMin = 5
	}
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "golangci-lint", "ruff", "shellcheck", "graph", "health", "static", "architecture", "semgrep":
		if configuredMin > 3 {
			return 3
		}
	}
	return configuredMin
}

// BackfillFalsePositiveLearningEvents records user_marked_false_positive events for findings
// already marked false_positive or suppressed that never fed the learning pipeline.
func (s *SQLiteStore) BackfillFalsePositiveLearningEvents(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 5000
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT f.id, f.repository_id,
			COALESCE(NULLIF(f.last_seen_scan_id, ''), NULLIF(f.first_seen_scan_id, ''), ''),
			f.fingerprint, f.source, f.rule_id, f.status
		FROM findings f
		WHERE f.status IN ('false_positive', 'suppressed')
		  AND f.repository_id > 0
		  AND NOT EXISTS (
			SELECT 1 FROM learning_events le
			WHERE le.finding_id = f.id AND le.event_type = 'user_marked_false_positive'
		  )
		ORDER BY f.id
		LIMIT ?
	`, limit)
	if err != nil {
		return 0, fmt.Errorf("query findings for backfill: %w", err)
	}
	defer rows.Close()

	type backfillRow struct {
		findingID, repoID int64
		scanID, fingerprint, source, ruleID, status string
	}
	var batch []backfillRow
	for rows.Next() {
		var row backfillRow
		if err := rows.Scan(&row.findingID, &row.repoID, &row.scanID, &row.fingerprint, &row.source, &row.ruleID, &row.status); err != nil {
			return 0, err
		}
		batch = append(batch, row)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	rows.Close()

	count := 0
	for _, row := range batch {
		fid := row.findingID
		evidence, _ := json.Marshal(map[string]any{
			"backfill":    true,
			"from_status": row.status,
			"finding_id":  row.findingID,
		})
		_, err := s.RecordLearningEvent(ctx, LearningEvent{
			RepositoryID:   row.repoID,
			ScanID:         row.scanID,
			FindingID:      &fid,
			Fingerprint:    row.fingerprint,
			Source:         row.source,
			RuleID:         row.ruleID,
			EventType:      "user_marked_false_positive",
			EvidenceJSON:   evidence,
			CreatedBy:      "backfill:learning_pipeline",
			IdempotencyKey: fmt.Sprintf("backfill:fp:finding:%d", row.findingID),
		})
		if err != nil {
			return count, fmt.Errorf("backfill finding %d: %w", row.findingID, err)
		}
		count++
	}
	return count, nil
}

// PurgePoisonedScannerFailureLearningEvents removes historical scanner_failed events
// recorded when a scanner bug misclassified successful runs as failures (e.g. gitleaks
// parse_failed before 2026-09-02). Poisoned events skew calibration toward noise.
func (s *SQLiteStore) PurgePoisonedScannerFailureLearningEvents(ctx context.Context, scannerNames []string, before time.Time) (int, error) {
	if len(scannerNames) == 0 {
		scannerNames = []string{"gitleaks", "gitleaks-history"}
	}
	if before.IsZero() {
		before = time.Date(2026, 9, 2, 4, 46, 0, 0, time.UTC)
	}
	cutoff := before.UTC().Format(time.RFC3339)
	placeholders := make([]string, len(scannerNames))
	args := make([]any, 0, len(scannerNames)*2+1)
	for i, name := range scannerNames {
		placeholders[i] = "?"
		args = append(args, name)
	}
	in := strings.Join(placeholders, ",")
	args = append(args, args[:len(scannerNames)]...)
	args = append(args, cutoff)
	query := fmt.Sprintf(`
		DELETE FROM learning_events
		WHERE event_type = 'scanner_failed'
		  AND (source IN (%s) OR rule_id IN (%s))
		  AND created_at < ?
	`, in, in)
	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("purge poisoned scanner_failed events: %w", err)
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		_, _ = s.db.ExecContext(ctx, fmt.Sprintf(`
			UPDATE rule_reliability_stats
			SET scanner_failure_count = 0
			WHERE source IN (%s) OR rule_id IN (%s)
		`, in, in), args[:len(scannerNames)*2]...)
	}
	return int(n), nil
}

// LearningHealthSummary aggregates learning metrics for dashboard.
func (s *SQLiteStore) LearningHealthSummary(ctx context.Context) (LearningHealthSummary, error) {
	var out LearningHealthSummary
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM learning_events`).Scan(&out.EventsTotal)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM calibration_recommendations WHERE status = 'proposed'`).Scan(&out.PendingRecommendations)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM repo_calibration_rules WHERE active = 1`).Scan(&out.ActiveRepoRules)
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM repo_calibration_rules WHERE active = 0 AND expires_at IS NOT NULL AND expires_at != ''
	`).Scan(&out.ExpiredRepoRules)
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM findings WHERE canonical_finding_id IS NOT NULL AND canonical_finding_id > 0
	`).Scan(&out.GroupedFindings)
	_ = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(false_positive_rate), 0) FROM calibration_rule_stats WHERE total_findings >= 3
	`).Scan(&out.AvgFalsePositiveRate)
	var failures, total int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM scanner_health_history WHERE status IN ('failed','error','timeout','parse_failed','timed_out')
	`).Scan(&failures)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM scanner_health_history`).Scan(&total)
	if total > 0 {
		out.ScannerFailureRate = float64(failures) / float64(total)
	}
	return out, nil
}

// ListRepositoryIDsAffectedByRule returns repos that have findings or learning events for a rule.
// Used when expanding a global calibration recommendation into repo-scoped accepts.
func (s *SQLiteStore) ListRepositoryIDsAffectedByRule(ctx context.Context, source, ruleID string, limit int) ([]int64, error) {
	if limit <= 0 {
		limit = 100
	}
	source = strings.TrimSpace(source)
	ruleID = strings.TrimSpace(ruleID)
	if source == "" && ruleID == "" {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT repository_id FROM (
			SELECT DISTINCT repository_id AS repository_id FROM findings
			WHERE repository_id > 0
			  AND (? = '' OR source = ?)
			  AND (? = '' OR rule_id = ?)
			UNION
			SELECT DISTINCT repository_id AS repository_id FROM learning_events
			WHERE repository_id > 0
			  AND (? = '' OR source = ?)
			  AND (? = '' OR rule_id = ?)
		)
		ORDER BY repository_id
		LIMIT ?
	`, source, source, ruleID, ruleID, source, source, ruleID, ruleID, limit)
	if err != nil {
		return nil, fmt.Errorf("list repos affected by rule: %w", err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// AssignStructuralGroup links findings sharing a structural hash to a canonical finding.
func (s *SQLiteStore) AssignStructuralGroup(ctx context.Context, repositoryID int64, structuralHash string, findingID int64) error {
	if structuralHash == "" || repositoryID <= 0 || findingID <= 0 {
		return nil
	}
	var canonical int64
	err := s.db.QueryRowContext(ctx, `
		SELECT id FROM findings
		WHERE repository_id = ? AND structural_hash = ? AND (canonical_finding_id IS NULL OR canonical_finding_id = id)
		ORDER BY id ASC LIMIT 1
	`, repositoryID, structuralHash).Scan(&canonical)
	if err == sql.ErrNoRows {
		canonical = findingID
	} else if err != nil {
		return err
	}
	note := fmt.Sprintf("Grouped by structural pattern (hash %s…)", truncateHash(structuralHash))
	_, err = s.db.ExecContext(ctx, `
		UPDATE findings SET structural_hash = ?, canonical_finding_id = ?,
			calibration_note = CASE WHEN calibration_note = '' THEN ? ELSE calibration_note END
		WHERE id = ?
	`, structuralHash, canonical, note, findingID)
	return err
}

func truncateHash(h string) string {
	if len(h) <= 12 {
		return h
	}
	return h[:12]
}
