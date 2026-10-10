package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// CountLearningEventsSince counts events of a given type created at or after since (RFC3339 / comparable text).
func (s *SQLiteStore) CountLearningEventsSince(ctx context.Context, eventType string, since time.Time) (int, error) {
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return 0, fmt.Errorf("event_type required")
	}
	cutoff := since.UTC().Format(time.RFC3339)
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM learning_events
		WHERE event_type = ? AND created_at >= ?
	`, eventType, cutoff).Scan(&n)
	return n, err
}

// GetAIAdvisoryRecommendationByID loads one advisory recommendation by numeric id.
func (s *SQLiteStore) GetAIAdvisoryRecommendationByID(ctx context.Context, id int64) (AIAdvisoryRecommendation, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, review_id, finding_fingerprint, classification, suggested_action,
			suggested_severity, suggested_confidence, reason, evidence_gaps_json,
			operator_status, created_at, updated_at
		FROM ai_advisory_recommendations WHERE id=?`, id)
	var rec AIAdvisoryRecommendation
	var created, updated string
	err := row.Scan(&rec.ID, &rec.ReviewID, &rec.FindingFingerprint, &rec.Classification,
		&rec.SuggestedAction, &rec.SuggestedSeverity, &rec.SuggestedConfidence, &rec.Reason,
		&rec.EvidenceGapsJSON, &rec.OperatorStatus, &created, &updated)
	if err != nil {
		return AIAdvisoryRecommendation{}, err
	}
	rec.CreatedAt = parseTime(created)
	rec.UpdatedAt = parseTime(updated)
	return rec, nil
}

// IngestAIAdvisoryCalibrateSuggestions turns pending AI calibrate/leave_visible noise
// suggestions into soft FP learning events so recompute can propose repo calibrations.
// Lesson from 2026-10-02: advisory triage sat pending forever and never trained the learner.
func (s *SQLiteStore) IngestAIAdvisoryCalibrateSuggestions(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, v.repository_id, v.scan_id, r.finding_fingerprint, r.suggested_action,
			r.classification, r.reason,
			COALESCE(f.source, ''), COALESCE(f.rule_id, ''), COALESCE(f.category, '')
		FROM ai_advisory_recommendations r
		JOIN ai_advisory_reviews v ON v.review_id = r.review_id
		LEFT JOIN findings f
			ON f.fingerprint = r.finding_fingerprint AND f.repository_id = v.repository_id
		WHERE r.operator_status = 'pending'
		  AND lower(r.suggested_action) IN ('calibrate_repo_scope', 'leave_visible')
		  AND lower(r.classification) IN ('possible_false_positive', 'likely_false_positive')
		ORDER BY r.created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return 0, err
	}
	type pending struct {
		recID, repoID                               int64
		scanID, fingerprint, action, classification string
		reason, source, ruleID, category            string
	}
	var batch []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.recID, &p.repoID, &p.scanID, &p.fingerprint, &p.action, &p.classification, &p.reason, &p.source, &p.ruleID, &p.category); err != nil {
			rows.Close()
			return 0, err
		}
		batch = append(batch, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	ingested := 0
	for _, p := range batch {
		if p.repoID <= 0 || strings.TrimSpace(p.ruleID) == "" {
			continue
		}
		if categoryProtectedFromAutoCalibrate(p.category) && !informationalToolingRuleForLearning(p.source, p.ruleID) {
			continue
		}
		ev := LearningEvent{
			RepositoryID:   p.repoID,
			ScanID:         p.scanID,
			Fingerprint:    p.fingerprint,
			Source:         p.source,
			RuleID:         p.ruleID,
			EventType:      "ai_advisory_calibrate_suggested",
			CreatedBy:      "ai-advisory-ingest",
			IdempotencyKey: fmt.Sprintf("ai-calibrate:%d", p.recID),
			EvidenceJSON: mustJSON(map[string]any{
				"ai_recommendation_id": p.recID,
				"suggested_action":     p.action,
				"classification":       p.classification,
				"reason":               trimReason(p.reason, 400),
			}),
		}
		if _, err := s.RecordLearningEvent(ctx, ev); err != nil {
			return ingested, err
		}
		// Advance operator_status so the same advisory is not re-ingested forever
		// and so the UI can show AI-assisted learning instead of a stuck pending queue.
		if err := s.UpdateAIAdvisoryRecommendationStatus(ctx, p.recID, "ingested_for_learning"); err != nil {
			return ingested, err
		}
		ingested++
	}
	return ingested, nil
}

// GenerateCalibrationFromAIAdvisory proposes repo report_only calibrations when the same
// rule receives repeated pending AI calibrate_repo_scope suggestions (minCount).
func (s *SQLiteStore) GenerateCalibrationFromAIAdvisory(ctx context.Context, minCount int) (int, error) {
	if minCount <= 0 {
		minCount = 2
	}
	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := s.db.QueryContext(ctx, `
		SELECT v.repository_id, COALESCE(f.source, ''), COALESCE(f.rule_id, ''), COALESCE(f.category, ''),
			COUNT(1) AS n, MAX(r.reason) AS reason
		FROM ai_advisory_recommendations r
		JOIN ai_advisory_reviews v ON v.review_id = r.review_id
		JOIN findings f
			ON f.fingerprint = r.finding_fingerprint AND f.repository_id = v.repository_id
		WHERE r.operator_status IN ('pending', 'ingested_for_learning')
		  AND lower(r.suggested_action) = 'calibrate_repo_scope'
		  AND lower(r.classification) IN ('possible_false_positive', 'likely_false_positive')
		  AND COALESCE(f.rule_id, '') != ''
		GROUP BY v.repository_id, f.source, f.rule_id, f.category
		HAVING n >= ?
	`, minCount)
	if err != nil {
		return 0, err
	}
	type cluster struct {
		repoID                           int64
		source, ruleID, category, reason string
		n                                int
	}
	var clusters []cluster
	for rows.Next() {
		var c cluster
		if err := rows.Scan(&c.repoID, &c.source, &c.ruleID, &c.category, &c.n, &c.reason); err != nil {
			rows.Close()
			return 0, err
		}
		clusters = append(clusters, c)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	count := 0
	for _, c := range clusters {
		if categoryProtectedFromAutoCalibrate(c.category) && !informationalToolingRuleForLearning(c.source, c.ruleID) {
			continue
		}
		var exists int
		if err := s.db.QueryRowContext(ctx, `
			SELECT COUNT(1) FROM calibration_recommendations
			WHERE scope = 'repo' AND repository_id = ? AND rule_id = ? AND source = ? AND status = 'proposed'
		`, c.repoID, c.ruleID, c.source).Scan(&exists); err != nil {
			return count, err
		}
		if exists > 0 {
			continue
		}
		confidence := float64(c.n) / float64(c.n+1)
		if confidence > 0.9 {
			confidence = 0.9
		}
		recReason := fmt.Sprintf(
			"Repo %d: %d pending AI calibrate_repo_scope suggestions for %s/%s (learned from advisory triage)",
			c.repoID, c.n, c.source, c.ruleID,
		)
		if trimmed := trimReason(c.reason, 180); trimmed != "" {
			recReason += " — " + trimmed
		}
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO calibration_recommendations (
				scope, repository_id, recommendation_type, source, rule_id, category,
				current_action, recommended_action, reason, confidence, status, created_at, updated_at
			) VALUES ('repo', ?, 'downgrade_confidence', ?, ?, ?, 'auto_issue', 'report_only', ?, ?, 'proposed', ?, ?)
		`, c.repoID, c.source, c.ruleID, c.category, recReason, confidence, now, now); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// FindingSourceRuleForFingerprint resolves source/rule/category for an advisory fingerprint.
func (s *SQLiteStore) FindingSourceRuleForFingerprint(ctx context.Context, repositoryID int64, fingerprint string) (source, ruleID, category string, err error) {
	err = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(source,''), COALESCE(rule_id,''), COALESCE(category,'')
		FROM findings WHERE repository_id = ? AND fingerprint = ? LIMIT 1
	`, repositoryID, fingerprint).Scan(&source, &ruleID, &category)
	return source, ruleID, category, err
}

func trimReason(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{}`)
	}
	return b
}

// categoryProtectedFromAutoCalibrate mirrors learning.IsProtectedFromAutoDowngrade
// without importing learning (store ↔ learning cycle).
func categoryProtectedFromAutoCalibrate(category string) bool {
	cat := strings.ToLower(strings.TrimSpace(category))
	if cat == "" {
		return false
	}
	for _, k := range []string{"secrets", "secret", "hardcoded_secret", "dependency_vulnerability", "security"} {
		if strings.Contains(cat, k) {
			return true
		}
	}
	return false
}

// informationalToolingRuleForLearning mirrors learning.IsInformationalToolingRule
// so AI calibrate suggestions for gosec G104/G304/G204 and graph/lint noise can
// train the learner even when scanners label the category as "security".
func informationalToolingRuleForLearning(source, ruleID string) bool {
	src := strings.ToLower(strings.TrimSpace(source))
	rule := strings.ToUpper(strings.TrimSpace(ruleID))
	rule = strings.Trim(rule, "`\"' ")
	if strings.Contains(src, "gosec") {
		switch rule {
		case "G104", "G304", "G204":
			return true
		}
	}
	if strings.HasPrefix(rule, "LINT-") || strings.HasPrefix(rule, "GRAPH-") ||
		strings.HasPrefix(rule, "OPT-") || strings.HasPrefix(rule, "QUAL-") {
		return true
	}
	if strings.HasPrefix(rule, "HEALTH-") && !strings.Contains(rule, "SECRET") {
		return true
	}
	return false
}
