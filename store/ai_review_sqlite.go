package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// CreateAIAdvisoryReview inserts an advisory review record.
func (s *SQLiteStore) CreateAIAdvisoryReview(ctx context.Context, rec AIAdvisoryReview) (AIAdvisoryReview, error) {
	if rec.RepositoryID <= 0 || rec.ReviewID == "" {
		return AIAdvisoryReview{}, fmt.Errorf("repository_id and review_id required")
	}
	now := time.Now().UTC()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	if rec.StartedAt.IsZero() {
		rec.StartedAt = now
	}
	packet := string(rec.PacketJSON)
	if packet == "" {
		packet = "{}"
	}
	response := string(rec.ResponseJSON)
	if response == "" {
		response = "{}"
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO ai_advisory_reviews (
			review_id, scan_id, repository_id, scan_type, status,
			findings_sent, redaction_count, recommendations_count, overall_assessment,
			packet_json, response_json, error_message, model, started_at, finished_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.ReviewID, rec.ScanID, rec.RepositoryID, rec.ScanType, rec.Status,
		rec.FindingsSent, rec.RedactionCount, rec.RecommendationsCount, rec.OverallAssessment,
		packet, response, rec.ErrorMessage, rec.Model,
		formatTimePtr(rec.StartedAt), formatTimePtrPtr(rec.FinishedAt), formatTime(rec.CreatedAt),
	)
	if err != nil {
		return AIAdvisoryReview{}, fmt.Errorf("create ai advisory review: %w", err)
	}
	id, _ := res.LastInsertId()
	rec.ID = id
	return rec, nil
}

// UpdateAIAdvisoryReview updates completion fields.
func (s *SQLiteStore) UpdateAIAdvisoryReview(ctx context.Context, rec AIAdvisoryReview) error {
	if rec.ReviewID == "" {
		return fmt.Errorf("review_id required")
	}
	response := string(rec.ResponseJSON)
	if response == "" {
		response = "{}"
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE ai_advisory_reviews SET
			status=?, findings_sent=?, redaction_count=?, recommendations_count=?,
			overall_assessment=?, response_json=?, error_message=?, model=?, finished_at=?
		WHERE review_id=?`,
		rec.Status, rec.FindingsSent, rec.RedactionCount, rec.RecommendationsCount,
		rec.OverallAssessment, response, rec.ErrorMessage, rec.Model,
		formatTimePtrPtr(rec.FinishedAt), rec.ReviewID,
	)
	return err
}

// GetAIAdvisoryReviewByScanID returns the latest review for a scan.
func (s *SQLiteStore) GetAIAdvisoryReviewByScanID(ctx context.Context, scanID string) (AIAdvisoryReview, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, review_id, scan_id, repository_id, scan_type, status,
			findings_sent, redaction_count, recommendations_count, overall_assessment,
			packet_json, response_json, error_message, model, started_at, finished_at, created_at
		FROM ai_advisory_reviews WHERE scan_id=? ORDER BY created_at DESC LIMIT 1`, scanID)
	return scanAIAdvisoryReviewRow(row)
}

// GetAIAdvisoryReview returns a review by review_id.
func (s *SQLiteStore) GetAIAdvisoryReview(ctx context.Context, reviewID string) (AIAdvisoryReview, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, review_id, scan_id, repository_id, scan_type, status,
			findings_sent, redaction_count, recommendations_count, overall_assessment,
			packet_json, response_json, error_message, model, started_at, finished_at, created_at
		FROM ai_advisory_reviews WHERE review_id=?`, reviewID)
	return scanAIAdvisoryReviewRow(row)
}

// ListAIAdvisoryRecommendations lists recommendations for a review.
func (s *SQLiteStore) ListAIAdvisoryRecommendations(ctx context.Context, reviewID string) ([]AIAdvisoryRecommendation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, review_id, finding_fingerprint, classification, suggested_action,
			suggested_severity, suggested_confidence, reason, evidence_gaps_json,
			operator_status, created_at, updated_at
		FROM ai_advisory_recommendations WHERE review_id=? ORDER BY id`, reviewID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AIAdvisoryRecommendation
	for rows.Next() {
		var rec AIAdvisoryRecommendation
		var created, updated string
		if err := rows.Scan(&rec.ID, &rec.ReviewID, &rec.FindingFingerprint, &rec.Classification,
			&rec.SuggestedAction, &rec.SuggestedSeverity, &rec.SuggestedConfidence, &rec.Reason,
			&rec.EvidenceGapsJSON, &rec.OperatorStatus, &created, &updated); err != nil {
			return nil, err
		}
		rec.CreatedAt = parseTime(created)
		rec.UpdatedAt = parseTime(updated)
		out = append(out, rec)
	}
	return out, rows.Err()
}

// ListPendingAIAdvisoryRecommendations lists operator-pending recommendations.
func (s *SQLiteStore) ListPendingAIAdvisoryRecommendations(ctx context.Context, limit int) ([]AIAdvisoryRecommendation, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, review_id, finding_fingerprint, classification, suggested_action,
			suggested_severity, suggested_confidence, reason, evidence_gaps_json,
			operator_status, created_at, updated_at
		FROM ai_advisory_recommendations WHERE operator_status='pending'
		ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AIAdvisoryRecommendation
	for rows.Next() {
		var rec AIAdvisoryRecommendation
		var created, updated string
		if err := rows.Scan(&rec.ID, &rec.ReviewID, &rec.FindingFingerprint, &rec.Classification,
			&rec.SuggestedAction, &rec.SuggestedSeverity, &rec.SuggestedConfidence, &rec.Reason,
			&rec.EvidenceGapsJSON, &rec.OperatorStatus, &created, &updated); err != nil {
			return nil, err
		}
		rec.CreatedAt = parseTime(created)
		rec.UpdatedAt = parseTime(updated)
		out = append(out, rec)
	}
	return out, rows.Err()
}

// CreateAIAdvisoryRecommendation inserts one recommendation.
func (s *SQLiteStore) CreateAIAdvisoryRecommendation(ctx context.Context, rec AIAdvisoryRecommendation) (AIAdvisoryRecommendation, error) {
	now := time.Now().UTC()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	if rec.UpdatedAt.IsZero() {
		rec.UpdatedAt = now
	}
	if rec.OperatorStatus == "" {
		rec.OperatorStatus = "pending"
	}
	if rec.EvidenceGapsJSON == "" {
		rec.EvidenceGapsJSON = "[]"
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO ai_advisory_recommendations (
			review_id, finding_fingerprint, classification, suggested_action,
			suggested_severity, suggested_confidence, reason, evidence_gaps_json,
			operator_status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.ReviewID, rec.FindingFingerprint, rec.Classification, rec.SuggestedAction,
		rec.SuggestedSeverity, rec.SuggestedConfidence, rec.Reason, rec.EvidenceGapsJSON,
		rec.OperatorStatus, formatTime(rec.CreatedAt), formatTime(rec.UpdatedAt),
	)
	if err != nil {
		return AIAdvisoryRecommendation{}, err
	}
	id, _ := res.LastInsertId()
	rec.ID = id
	return rec, nil
}

// UpdateAIAdvisoryRecommendationStatus updates operator accept/reject.
func (s *SQLiteStore) UpdateAIAdvisoryRecommendationStatus(ctx context.Context, id int64, status string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE ai_advisory_recommendations SET operator_status=?, updated_at=? WHERE id=?`,
		status, formatTime(time.Now().UTC()), id,
	)
	return err
}

func scanAIAdvisoryReviewRow(row *sql.Row) (AIAdvisoryReview, error) {
	var rec AIAdvisoryReview
	var packet, response, started, finished, created sql.NullString
	err := row.Scan(&rec.ID, &rec.ReviewID, &rec.ScanID, &rec.RepositoryID, &rec.ScanType, &rec.Status,
		&rec.FindingsSent, &rec.RedactionCount, &rec.RecommendationsCount, &rec.OverallAssessment,
		&packet, &response, &rec.ErrorMessage, &rec.Model, &started, &finished, &created)
	if err != nil {
		return AIAdvisoryReview{}, err
	}
	if packet.Valid {
		rec.PacketJSON = []byte(packet.String)
	}
	if response.Valid {
		rec.ResponseJSON = []byte(response.String)
	}
	rec.StartedAt = parseTime(started.String)
	if finished.Valid && finished.String != "" {
		t := parseTime(finished.String)
		rec.FinishedAt = &t
	}
	rec.CreatedAt = parseTime(created.String)
	return rec, nil
}
