package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func (s *SQLiteStore) SaveClosureEvidence(ctx context.Context, rec ClosureEvidenceRecord) (ClosureEvidenceRecord, error) {
	now := time.Now().UTC()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	rec.UpdatedAt = now
	if rec.Status == "" {
		rec.Status = ClosureStatusPendingRescan
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO closure_evidence (
			finding_id, patch_attempt_id, repository_id, fingerprint,
			merge_commit_sha, verification_scan_id, original_source, scanner_status,
			fingerprint_present, status, reason, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		rec.FindingID, nullString(rec.PatchAttemptID), rec.RepositoryID, rec.Fingerprint,
		rec.MergeCommitSHA, rec.VerificationScanID, rec.OriginalSource, rec.ScannerStatus,
		boolToInt(rec.FingerprintPresent), rec.Status, rec.Reason,
		formatTime(rec.CreatedAt), formatTime(rec.UpdatedAt),
	)
	if err != nil {
		return ClosureEvidenceRecord{}, fmt.Errorf("insert closure evidence: %w", err)
	}
	id, _ := res.LastInsertId()
	rec.ID = id
	return rec, nil
}

func (s *SQLiteStore) UpdateClosureEvidence(ctx context.Context, rec ClosureEvidenceRecord) error {
	rec.UpdatedAt = time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `
		UPDATE closure_evidence SET
			verification_scan_id = ?, scanner_status = ?, fingerprint_present = ?,
			status = ?, reason = ?, updated_at = ?
		WHERE id = ?
	`,
		rec.VerificationScanID, rec.ScannerStatus, boolToInt(rec.FingerprintPresent),
		rec.Status, rec.Reason, formatTime(rec.UpdatedAt), rec.ID,
	)
	if err != nil {
		return fmt.Errorf("update closure evidence: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("closure evidence not found")
	}
	return nil
}

func (s *SQLiteStore) GetLatestClosureEvidenceByFindingID(ctx context.Context, findingID int64) (ClosureEvidenceRecord, error) {
	row := s.db.QueryRowContext(ctx, closureEvidenceSelect+`
		WHERE finding_id = ? ORDER BY updated_at DESC LIMIT 1
	`, findingID)
	return scanClosureEvidenceRow(row)
}

func (s *SQLiteStore) ListClosureEvidenceByRepositoryAndStatus(ctx context.Context, repositoryID int64, status string) ([]ClosureEvidenceRecord, error) {
	rows, err := s.db.QueryContext(ctx, closureEvidenceSelect+`
		WHERE repository_id = ? AND status = ? ORDER BY updated_at DESC
	`, repositoryID, status)
	if err != nil {
		return nil, fmt.Errorf("list closure evidence: %w", err)
	}
	defer rows.Close()
	var out []ClosureEvidenceRecord
	for rows.Next() {
		rec, err := scanClosureEvidenceRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ClosureSummary(ctx context.Context) (ClosureSummary, error) {
	var summary ClosureSummary
	row := s.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0)
		FROM closure_evidence
	`, ClosureStatusPendingRescan, ClosureStatusVerified, ClosureStatusBlocked)
	if err := row.Scan(&summary.PendingRescan, &summary.Verified, &summary.Blocked); err != nil {
		if err == sql.ErrNoRows {
			return ClosureSummary{}, nil
		}
		return ClosureSummary{}, err
	}
	return summary, nil
}

func (s *SQLiteStore) UpdateFindingStatus(ctx context.Context, findingID int64, status string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE findings SET status = ? WHERE id = ?`, status, findingID)
	if err != nil {
		return fmt.Errorf("update finding status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("finding not found")
	}
	return nil
}

func (s *SQLiteStore) ListPatchAttemptsByRepositoryAndStatus(ctx context.Context, repositoryID int64, status string) ([]PatchAttemptRecord, error) {
	query := patchAttemptSelect + ` WHERE status = ?`
	args := []any{status}
	if repositoryID > 0 {
		query += ` AND repository_id = ?`
		args = append(args, repositoryID)
	}
	query += ` ORDER BY updated_at DESC`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list patch attempts: %w", err)
	}
	defer rows.Close()
	var out []PatchAttemptRecord
	for rows.Next() {
		rec, err := scanPatchAttemptRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetPatchAttemptForClosure(ctx context.Context, attemptID string) (PatchAttemptRecord, Finding, error) {
	rec, err := s.GetPatchAttemptByAttemptID(ctx, attemptID)
	if err != nil {
		return PatchAttemptRecord{}, Finding{}, err
	}
	var finding Finding
	if rec.FindingID != nil {
		finding, err = s.getFindingByID(ctx, *rec.FindingID)
		if err != nil {
			return rec, Finding{}, err
		}
	}
	return rec, finding, nil
}

func (s *SQLiteStore) UpdatePatchAttemptMerged(ctx context.Context, attemptID, mergeSHA string, mergedAt time.Time) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE patch_attempts SET status = ?, merge_commit_sha = ?, merged_at = ?, updated_at = ?
		WHERE attempt_id = ?
	`, PatchAttemptStatusPRMerged, mergeSHA, formatTime(mergedAt.UTC()), formatTime(time.Now().UTC()), attemptID)
	if err != nil {
		return fmt.Errorf("update patch attempt merged: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("patch attempt not found")
	}
	return nil
}

func (s *SQLiteStore) getFindingByID(ctx context.Context, id int64) (Finding, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, repository_id, fingerprint, category, severity, confidence, source, rule_id,
			package_name, file_path, line, title, status,
			first_seen_scan_id, last_seen_scan_id, first_seen_at, last_seen_at
		FROM findings WHERE id = ?
	`, id)
	var f Finding
	var firstSeen, lastSeen string
	err := row.Scan(
		&f.ID, &f.RepositoryID, &f.Fingerprint, &f.Category, &f.Severity, &f.Confidence,
		&f.Source, &f.RuleID, &f.PackageName, &f.FilePath, &f.Line, &f.Title, &f.Status,
		&f.FirstSeenScanID, &f.LastSeenScanID, &firstSeen, &lastSeen,
	)
	if err != nil {
		return Finding{}, err
	}
	f.FirstSeenAt = parseTime(firstSeen)
	f.LastSeenAt = parseTime(lastSeen)
	return f, nil
}

const closureEvidenceSelect = `
	SELECT id, finding_id, patch_attempt_id, repository_id, fingerprint,
		merge_commit_sha, verification_scan_id, original_source, scanner_status,
		fingerprint_present, status, reason, created_at, updated_at
	FROM closure_evidence
`

func scanClosureEvidenceRow(row *sql.Row) (ClosureEvidenceRecord, error) {
	var rec ClosureEvidenceRecord
	var patchAttempt sql.NullString
	var fpPresent int
	var created, updated string
	err := row.Scan(
		&rec.ID, &rec.FindingID, &patchAttempt, &rec.RepositoryID, &rec.Fingerprint,
		&rec.MergeCommitSHA, &rec.VerificationScanID, &rec.OriginalSource, &rec.ScannerStatus,
		&fpPresent, &rec.Status, &rec.Reason, &created, &updated,
	)
	if err != nil {
		return ClosureEvidenceRecord{}, err
	}
	if patchAttempt.Valid {
		rec.PatchAttemptID = patchAttempt.String
	}
	rec.FingerprintPresent = fpPresent != 0
	rec.CreatedAt = parseTime(created)
	rec.UpdatedAt = parseTime(updated)
	return rec, nil
}

func scanClosureEvidenceRows(rows *sql.Rows) (ClosureEvidenceRecord, error) {
	var rec ClosureEvidenceRecord
	var patchAttempt sql.NullString
	var fpPresent int
	var created, updated string
	err := rows.Scan(
		&rec.ID, &rec.FindingID, &patchAttempt, &rec.RepositoryID, &rec.Fingerprint,
		&rec.MergeCommitSHA, &rec.VerificationScanID, &rec.OriginalSource, &rec.ScannerStatus,
		&fpPresent, &rec.Status, &rec.Reason, &created, &updated,
	)
	if err != nil {
		return ClosureEvidenceRecord{}, err
	}
	if patchAttempt.Valid {
		rec.PatchAttemptID = patchAttempt.String
	}
	rec.FingerprintPresent = fpPresent != 0
	rec.CreatedAt = parseTime(created)
	rec.UpdatedAt = parseTime(updated)
	return rec, nil
}
