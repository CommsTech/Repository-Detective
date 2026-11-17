package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (s *SQLiteStore) CreateRunnerJob(ctx context.Context, job RunnerJob) (RunnerJob, error) {
	if job.JobID == "" {
		return RunnerJob{}, fmt.Errorf("job_id is required")
	}
	if job.RepositoryID == 0 {
		return RunnerJob{}, fmt.Errorf("repository_id is required")
	}
	now := time.Now().UTC()
	if job.Status == "" {
		job.Status = RunnerJobStatusQueued
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	job.UpdatedAt = now
	if job.PolicySnapshotJSON == nil {
		job.PolicySnapshotJSON = json.RawMessage(`{}`)
	}
	if job.JobSpecJSON == nil {
		job.JobSpecJSON = json.RawMessage(`{}`)
	}
	if job.ResultSummaryJSON == nil {
		job.ResultSummaryJSON = json.RawMessage(`{}`)
	}

	res, err := s.db.ExecContext(ctx, `
		INSERT INTO runner_jobs (
			job_id, repository_id, scan_id, job_type, status, runner_mode,
			ref, commit_sha, pr_number, policy_snapshot_json, job_spec_json,
			result_summary_json, error, created_at, updated_at, started_at, finished_at, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		job.JobID, job.RepositoryID, nullString(job.ScanID), job.JobType, job.Status, job.RunnerMode,
		job.Ref, job.CommitSHA, job.PRNumber, string(job.PolicySnapshotJSON), string(job.JobSpecJSON),
		string(job.ResultSummaryJSON), job.Error, formatTime(job.CreatedAt), formatTime(job.UpdatedAt),
		nullTimePtr(job.StartedAt), nullTimePtr(job.FinishedAt), nullTimePtr(job.ExpiresAt),
	)
	if err != nil {
		return RunnerJob{}, fmt.Errorf("create runner job: %w", err)
	}
	id, _ := res.LastInsertId()
	job.ID = id
	return job, nil
}

func (s *SQLiteStore) GetRunnerJob(ctx context.Context, jobID string) (RunnerJob, error) {
	return scanRunnerJob(s.db.QueryRowContext(ctx, `
		SELECT id, job_id, repository_id, scan_id, job_type, status, runner_mode,
			ref, commit_sha, pr_number, policy_snapshot_json, job_spec_json,
			result_summary_json, error, created_at, updated_at, started_at, finished_at, expires_at
		FROM runner_jobs WHERE job_id = ?
	`, jobID))
}

func (s *SQLiteStore) GetRunnerJobByScanID(ctx context.Context, scanID string) (RunnerJob, error) {
	return scanRunnerJob(s.db.QueryRowContext(ctx, `
		SELECT id, job_id, repository_id, scan_id, job_type, status, runner_mode,
			ref, commit_sha, pr_number, policy_snapshot_json, job_spec_json,
			result_summary_json, error, created_at, updated_at, started_at, finished_at, expires_at
		FROM runner_jobs WHERE scan_id = ? ORDER BY created_at DESC LIMIT 1
	`, scanID))
}

func (s *SQLiteStore) ClaimNextRunnerJob(ctx context.Context, now time.Time) (RunnerJob, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return RunnerJob{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var jobID string
	err = tx.QueryRowContext(ctx, `
		SELECT job_id FROM runner_jobs
		WHERE status = ? AND (expires_at IS NULL OR expires_at > ?)
		ORDER BY created_at ASC LIMIT 1
	`, RunnerJobStatusQueued, formatTime(now)).Scan(&jobID)
	if err == sql.ErrNoRows {
		return RunnerJob{}, fmt.Errorf("no runner jobs available")
	}
	if err != nil {
		return RunnerJob{}, fmt.Errorf("select runner job: %w", err)
	}

	started := now
	_, err = tx.ExecContext(ctx, `
		UPDATE runner_jobs SET status = ?, started_at = ?, updated_at = ?
		WHERE job_id = ? AND status = ?
	`, RunnerJobStatusRunning, formatTime(started), formatTime(now), jobID, RunnerJobStatusQueued)
	if err != nil {
		return RunnerJob{}, fmt.Errorf("claim runner job: %w", err)
	}

	job, err := scanRunnerJob(tx.QueryRowContext(ctx, `
		SELECT id, job_id, repository_id, scan_id, job_type, status, runner_mode,
			ref, commit_sha, pr_number, policy_snapshot_json, job_spec_json,
			result_summary_json, error, created_at, updated_at, started_at, finished_at, expires_at
		FROM runner_jobs WHERE job_id = ?
	`, jobID))
	if err != nil {
		return RunnerJob{}, err
	}
	if err := tx.Commit(); err != nil {
		return RunnerJob{}, err
	}
	return job, nil
}

func (s *SQLiteStore) UpdateRunnerJob(ctx context.Context, job RunnerJob) error {
	job.UpdatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
		UPDATE runner_jobs SET status = ?, result_summary_json = ?, error = ?,
			updated_at = ?, started_at = ?, finished_at = ?
		WHERE job_id = ?
	`,
		job.Status, stringOrEmpty(job.ResultSummaryJSON), job.Error,
		formatTime(job.UpdatedAt), nullTimePtr(job.StartedAt), nullTimePtr(job.FinishedAt), job.JobID,
	)
	if err != nil {
		return fmt.Errorf("update runner job: %w", err)
	}
	return nil
}

func (s *SQLiteStore) CancelRunnerJob(ctx context.Context, jobID string) error {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `
		UPDATE runner_jobs SET status = ?, updated_at = ?, finished_at = ?
		WHERE job_id = ? AND status IN (?, ?, ?)
	`, RunnerJobStatusCancelled, formatTime(now), formatTime(now), jobID,
		RunnerJobStatusQueued, RunnerJobStatusDispatched, RunnerJobStatusRunning)
	if err != nil {
		return fmt.Errorf("cancel runner job: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("runner job not cancellable")
	}
	return nil
}

func (s *SQLiteStore) ListRunnerJobs(ctx context.Context, opts ListOptions) ([]RunnerJob, error) {
	opts = NormalizeListOptions(opts)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, job_id, repository_id, scan_id, job_type, status, runner_mode,
			ref, commit_sha, pr_number, policy_snapshot_json, job_spec_json,
			result_summary_json, error, created_at, updated_at, started_at, finished_at, expires_at
		FROM runner_jobs ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, opts.Limit, opts.Offset)
	if err != nil {
		return nil, fmt.Errorf("list runner jobs: %w", err)
	}
	defer rows.Close()
	return scanRunnerJobs(rows)
}

func (s *SQLiteStore) ListRunnerJobsByRepository(ctx context.Context, repositoryID int64, opts ListOptions) ([]RunnerJob, error) {
	opts = NormalizeListOptions(opts)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, job_id, repository_id, scan_id, job_type, status, runner_mode,
			ref, commit_sha, pr_number, policy_snapshot_json, job_spec_json,
			result_summary_json, error, created_at, updated_at, started_at, finished_at, expires_at
		FROM runner_jobs WHERE repository_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, repositoryID, opts.Limit, opts.Offset)
	if err != nil {
		return nil, fmt.Errorf("list runner jobs by repo: %w", err)
	}
	defer rows.Close()
	return scanRunnerJobs(rows)
}

func (s *SQLiteStore) CountRunningRunnerJobs(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM runner_jobs WHERE status IN (?, ?, ?)
	`, RunnerJobStatusQueued, RunnerJobStatusDispatched, RunnerJobStatusRunning).Scan(&count)
	return count, err
}

func (s *SQLiteStore) ExpireStaleRunnerJobs(ctx context.Context, now time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE runner_jobs SET status = ?, error = ?, updated_at = ?, finished_at = ?
		WHERE status IN (?, ?, ?) AND expires_at IS NOT NULL AND expires_at <= ?
	`, RunnerJobStatusExpired, "job expired", formatTime(now), formatTime(now),
		RunnerJobStatusQueued, RunnerJobStatusDispatched, RunnerJobStatusRunning, formatTime(now))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *SQLiteStore) SaveRunnerArtifact(ctx context.Context, artifact RunnerArtifact) error {
	if artifact.JobID == "" {
		return fmt.Errorf("job_id is required")
	}
	if artifact.CreatedAt.IsZero() {
		artifact.CreatedAt = time.Now().UTC()
	}
	if artifact.BodyJSON == nil {
		artifact.BodyJSON = json.RawMessage(`{}`)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO runner_artifacts (job_id, artifact_type, body_json, size_bytes, sha256, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, artifact.JobID, artifact.ArtifactType, string(artifact.BodyJSON), artifact.SizeBytes, artifact.SHA256, formatTime(artifact.CreatedAt))
	if err != nil {
		return fmt.Errorf("save runner artifact: %w", err)
	}
	return nil
}

func (s *SQLiteStore) TryRecordRunnerNonce(ctx context.Context, nonce string) (bool, error) {
	if strings.TrimSpace(nonce) == "" {
		return false, fmt.Errorf("nonce is required")
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO runner_nonces (nonce, created_at) VALUES (?, datetime('now'))
	`, nonce)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return false, nil
		}
		return false, fmt.Errorf("record runner nonce: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *SQLiteStore) CountRunnerJobsByStatus(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT status, COUNT(1) FROM runner_jobs GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		out[status] = count
	}
	return out, rows.Err()
}

// RunnerJobSummary holds recent runner queue telemetry for operator UI.
type RunnerJobSummary struct {
	LastJobAt *time.Time
	LastError string
}

func (s *SQLiteStore) RunnerJobSummary(ctx context.Context) (RunnerJobSummary, error) {
	var summary RunnerJobSummary
	var updatedAt, errMsg sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT updated_at, error FROM runner_jobs ORDER BY updated_at DESC LIMIT 1
	`).Scan(&updatedAt, &errMsg)
	if err == sql.ErrNoRows {
		return summary, nil
	}
	if err != nil {
		return summary, err
	}
	if updatedAt.Valid {
		t := parseTime(updatedAt.String)
		summary.LastJobAt = &t
	}
	if errMsg.Valid {
		summary.LastError = errMsg.String
	}
	return summary, nil
}

func scanRunnerJob(row *sql.Row) (RunnerJob, error) {
	var job RunnerJob
	var scanID, policy, spec, result, errMsg sql.NullString
	var createdAt, updatedAt string
	var startedAt, finishedAt, expiresAt sql.NullString
	err := row.Scan(
		&job.ID, &job.JobID, &job.RepositoryID, &scanID, &job.JobType, &job.Status, &job.RunnerMode,
		&job.Ref, &job.CommitSHA, &job.PRNumber, &policy, &spec, &result, &errMsg,
		&createdAt, &updatedAt, &startedAt, &finishedAt, &expiresAt,
	)
	if err == sql.ErrNoRows {
		return RunnerJob{}, fmt.Errorf("runner job not found")
	}
	if err != nil {
		return RunnerJob{}, err
	}
	if scanID.Valid {
		job.ScanID = scanID.String
	}
	if policy.Valid {
		job.PolicySnapshotJSON = json.RawMessage(policy.String)
	}
	if spec.Valid {
		job.JobSpecJSON = json.RawMessage(spec.String)
	}
	if result.Valid {
		job.ResultSummaryJSON = json.RawMessage(result.String)
	}
	if errMsg.Valid {
		job.Error = errMsg.String
	}
	job.CreatedAt = parseTime(createdAt)
	job.UpdatedAt = parseTime(updatedAt)
	if startedAt.Valid {
		t := parseTime(startedAt.String)
		job.StartedAt = &t
	}
	if finishedAt.Valid {
		t := parseTime(finishedAt.String)
		job.FinishedAt = &t
	}
	if expiresAt.Valid {
		t := parseTime(expiresAt.String)
		job.ExpiresAt = &t
	}
	return job, nil
}

func scanRunnerJobs(rows *sql.Rows) ([]RunnerJob, error) {
	var jobs []RunnerJob
	for rows.Next() {
		var job RunnerJob
		var scanID, policy, spec, result, errMsg sql.NullString
		var createdAt, updatedAt string
		var startedAt, finishedAt, expiresAt sql.NullString
		if err := rows.Scan(
			&job.ID, &job.JobID, &job.RepositoryID, &scanID, &job.JobType, &job.Status, &job.RunnerMode,
			&job.Ref, &job.CommitSHA, &job.PRNumber, &policy, &spec, &result, &errMsg,
			&createdAt, &updatedAt, &startedAt, &finishedAt, &expiresAt,
		); err != nil {
			return nil, err
		}
		if scanID.Valid {
			job.ScanID = scanID.String
		}
		if policy.Valid {
			job.PolicySnapshotJSON = json.RawMessage(policy.String)
		}
		if spec.Valid {
			job.JobSpecJSON = json.RawMessage(spec.String)
		}
		if result.Valid {
			job.ResultSummaryJSON = json.RawMessage(result.String)
		}
		if errMsg.Valid {
			job.Error = errMsg.String
		}
		job.CreatedAt = parseTime(createdAt)
		job.UpdatedAt = parseTime(updatedAt)
		if startedAt.Valid {
			t := parseTime(startedAt.String)
			job.StartedAt = &t
		}
		if finishedAt.Valid {
			t := parseTime(finishedAt.String)
			job.FinishedAt = &t
		}
		if expiresAt.Valid {
			t := parseTime(expiresAt.String)
			job.ExpiresAt = &t
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func nullString(v string) interface{} {
	if v == "" {
		return nil
	}
	return v
}

func stringOrEmpty(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}
	return string(raw)
}

func nullTimePtr(t *time.Time) interface{} {
	if t == nil || t.IsZero() {
		return nil
	}
	return formatTime(*t)
}
