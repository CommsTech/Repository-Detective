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
		HAVING total >= ?
	`, repositoryID, minFindings)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var source, ruleID string
		var fp, tp, total int
		if err := rows.Scan(&source, &ruleID, &fp, &tp, &total); err != nil {
			return count, err
		}
		fpRate := float64(fp) / float64(total)
		if fpRate < 0.5 {
			continue
		}
		var exists int
		_ = s.db.QueryRowContext(ctx, `
			SELECT COUNT(1) FROM calibration_recommendations
			WHERE scope = 'repo' AND repository_id = ? AND rule_id = ? AND source = ? AND status = 'proposed'
		`, repositoryID, ruleID, source).Scan(&exists)
		if exists > 0 {
			continue
		}
		reason := fmt.Sprintf("Repo %d: %d events, %.0f%% marked false positive (deterministic learning)", repositoryID, total, fpRate*100)
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO calibration_recommendations (
				scope, repository_id, recommendation_type, source, rule_id, category,
				current_action, recommended_action, reason, confidence, status, created_at, updated_at
			) VALUES ('repo', ?, 'downgrade_confidence', ?, ?, '', 'auto_issue', 'report_only', ?, ?, 'proposed', ?, ?)
		`, repositoryID, source, ruleID, reason, fpRate, now, now); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
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
