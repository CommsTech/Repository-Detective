package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func (s *SQLiteStore) SavePatchAttempt(ctx context.Context, attempt PatchAttemptRecord) (PatchAttemptRecord, error) {
	now := time.Now().UTC()
	if attempt.CreatedAt.IsZero() {
		attempt.CreatedAt = now
	}
	attempt.UpdatedAt = now
	if attempt.Status == "" {
		attempt.Status = PatchAttemptStatusProposed
	}

	res, err := s.db.ExecContext(ctx, `
		INSERT INTO patch_attempts (
			attempt_id, plan_id, repository_id, finding_id,
			branch_name, base_ref, base_commit_sha, status,
			diff_summary, files_changed_json, tests_run_json, validation_summary,
			pull_request_number, pull_request_url, error, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		attempt.AttemptID, attempt.PlanID, attempt.RepositoryID, nullInt64Value(attempt.FindingID),
		attempt.BranchName, attempt.BaseRef, attempt.BaseCommitSHA, attempt.Status,
		attempt.DiffSummary, stringOrEmptyJSON(attempt.FilesChangedJSON),
		stringOrEmptyJSON(attempt.TestsRunJSON), attempt.ValidationSummary,
		nullIntValue(attempt.PullRequestNumber), attempt.PullRequestURL, attempt.Error,
		formatTime(attempt.CreatedAt), formatTime(attempt.UpdatedAt),
	)
	if err != nil {
		return PatchAttemptRecord{}, fmt.Errorf("insert patch attempt: %w", err)
	}
	id, _ := res.LastInsertId()
	attempt.ID = id
	return attempt, nil
}

func (s *SQLiteStore) UpdatePatchAttempt(ctx context.Context, attempt PatchAttemptRecord) error {
	attempt.UpdatedAt = time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `
		UPDATE patch_attempts SET
			status = ?, diff_summary = ?, files_changed_json = ?, tests_run_json = ?,
			validation_summary = ?, pull_request_number = ?, pull_request_url = ?,
			base_commit_sha = ?, branch_name = ?, error = ?, updated_at = ?
		WHERE attempt_id = ?
	`,
		attempt.Status, attempt.DiffSummary, stringOrEmptyJSON(attempt.FilesChangedJSON),
		stringOrEmptyJSON(attempt.TestsRunJSON), attempt.ValidationSummary,
		nullIntValue(attempt.PullRequestNumber), attempt.PullRequestURL,
		attempt.BaseCommitSHA, attempt.BranchName, attempt.Error,
		formatTime(attempt.UpdatedAt), attempt.AttemptID,
	)
	if err != nil {
		return fmt.Errorf("update patch attempt: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("patch attempt not found")
	}
	return nil
}

func (s *SQLiteStore) GetPatchAttemptByAttemptID(ctx context.Context, attemptID string) (PatchAttemptRecord, error) {
	row := s.db.QueryRowContext(ctx, patchAttemptSelect+` WHERE attempt_id = ?`, attemptID)
	rec, err := scanPatchAttempt(row)
	if err != nil {
		return PatchAttemptRecord{}, fmt.Errorf("get patch attempt: %w", err)
	}
	return rec, nil
}

func (s *SQLiteStore) ListPatchAttemptsByPlanID(ctx context.Context, planID string) ([]PatchAttemptRecord, error) {
	rows, err := s.db.QueryContext(ctx, patchAttemptSelect+`
		WHERE plan_id = ? ORDER BY created_at DESC
	`, planID)
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

const patchAttemptSelect = `
	SELECT id, attempt_id, plan_id, repository_id, finding_id,
		branch_name, base_ref, base_commit_sha, status,
		diff_summary, files_changed_json, tests_run_json, validation_summary,
		pull_request_number, pull_request_url, error,
		merged_at, merge_commit_sha, created_at, updated_at
	FROM patch_attempts
`

func scanPatchAttempt(row *sql.Row) (PatchAttemptRecord, error) {
	var rec PatchAttemptRecord
	var findingID sql.NullInt64
	var prNum sql.NullInt64
	var mergedAt sql.NullString
	var files, tests string
	var created, updated string
	err := row.Scan(
		&rec.ID, &rec.AttemptID, &rec.PlanID, &rec.RepositoryID, &findingID,
		&rec.BranchName, &rec.BaseRef, &rec.BaseCommitSHA, &rec.Status,
		&rec.DiffSummary, &files, &tests, &rec.ValidationSummary,
		&prNum, &rec.PullRequestURL, &rec.Error,
		&mergedAt, &rec.MergeCommitSHA, &created, &updated,
	)
	if err != nil {
		return PatchAttemptRecord{}, err
	}
	fillPatchAttemptRecord(&rec, findingID, prNum, mergedAt, files, tests, created, updated)
	return rec, nil
}

func scanPatchAttemptRow(rows *sql.Rows) (PatchAttemptRecord, error) {
	var rec PatchAttemptRecord
	var findingID sql.NullInt64
	var prNum sql.NullInt64
	var mergedAt sql.NullString
	var files, tests string
	var created, updated string
	err := rows.Scan(
		&rec.ID, &rec.AttemptID, &rec.PlanID, &rec.RepositoryID, &findingID,
		&rec.BranchName, &rec.BaseRef, &rec.BaseCommitSHA, &rec.Status,
		&rec.DiffSummary, &files, &tests, &rec.ValidationSummary,
		&prNum, &rec.PullRequestURL, &rec.Error,
		&mergedAt, &rec.MergeCommitSHA, &created, &updated,
	)
	if err != nil {
		return PatchAttemptRecord{}, err
	}
	fillPatchAttemptRecord(&rec, findingID, prNum, mergedAt, files, tests, created, updated)
	return rec, nil
}

func fillPatchAttemptRecord(rec *PatchAttemptRecord, findingID, prNum sql.NullInt64, mergedAt sql.NullString, files, tests, created, updated string) {
	if findingID.Valid {
		v := findingID.Int64
		rec.FindingID = &v
	}
	if prNum.Valid {
		v := int(prNum.Int64)
		rec.PullRequestNumber = &v
	}
	if mergedAt.Valid && mergedAt.String != "" {
		t := parseTime(mergedAt.String)
		rec.MergedAt = &t
	}
	rec.FilesChangedJSON = []byte(files)
	rec.TestsRunJSON = []byte(tests)
	rec.CreatedAt = parseTime(created)
	rec.UpdatedAt = parseTime(updated)
}

func nullIntValue(v *int) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*v), Valid: true}
}
