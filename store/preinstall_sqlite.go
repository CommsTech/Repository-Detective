package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func (s *SQLiteStore) CreateAuditRequest(ctx context.Context, req AuditRequest) (AuditRequest, error) {
	if req.AuditID == "" {
		return AuditRequest{}, fmt.Errorf("audit id required")
	}
	if req.Status == "" {
		req.Status = AuditStatusQueued
	}
	if req.Recommendation == "" {
		req.Recommendation = AuditRecommendationUnknown
	}
	if req.StartedAt.IsZero() {
		req.StartedAt = time.Now().UTC()
	}
	if len(req.SummaryJSON) == 0 {
		req.SummaryJSON = jsonRawObject()
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO audit_requests (
			audit_id, repo_url, normalized_repo_url, repo_host, repo_owner, repo_name,
			commit_sha, default_branch, audit_depth, status, risk_score, recommendation,
			started_at, finished_at, summary_json, error
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		req.AuditID, req.RepoURL, req.NormalizedRepoURL, req.RepoHost, req.RepoOwner, req.RepoName,
		req.CommitSHA, req.DefaultBranch, req.AuditDepth, req.Status, req.RiskScore, req.Recommendation,
		formatTime(req.StartedAt), nullTimeString(req.FinishedAt), string(req.SummaryJSON), req.Error,
	)
	if err != nil {
		return AuditRequest{}, fmt.Errorf("create audit request: %w", err)
	}
	return req, nil
}

func (s *SQLiteStore) UpdateAuditRequest(ctx context.Context, req AuditRequest) error {
	if req.AuditID == "" {
		return fmt.Errorf("audit id required")
	}
	summary := req.SummaryJSON
	if len(summary) == 0 {
		summary = jsonRawObject()
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE audit_requests SET
			commit_sha = ?, default_branch = ?, status = ?, risk_score = ?, recommendation = ?,
			finished_at = ?, summary_json = ?, error = ?
		WHERE audit_id = ?
	`,
		req.CommitSHA, req.DefaultBranch, req.Status, req.RiskScore, req.Recommendation,
		nullTimeString(req.FinishedAt), string(summary), req.Error, req.AuditID,
	)
	if err != nil {
		return fmt.Errorf("update audit request: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetAuditRequest(ctx context.Context, auditID string) (AuditRequest, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT audit_id, repo_url, normalized_repo_url, repo_host, repo_owner, repo_name,
			commit_sha, default_branch, audit_depth, status, risk_score, recommendation,
			started_at, finished_at, summary_json, error
		FROM audit_requests WHERE audit_id = ?
	`, auditID)
	return scanAuditRequest(row)
}

func (s *SQLiteStore) ListAuditRequests(ctx context.Context, opts ListOptions) ([]AuditRequest, error) {
	opts = NormalizeListOptions(opts)
	rows, err := s.db.QueryContext(ctx, `
		SELECT audit_id, repo_url, normalized_repo_url, repo_host, repo_owner, repo_name,
			commit_sha, default_branch, audit_depth, status, risk_score, recommendation,
			started_at, finished_at, summary_json, error
		FROM audit_requests
		ORDER BY started_at DESC
		LIMIT ? OFFSET ?
	`, opts.Limit, opts.Offset)
	if err != nil {
		return nil, fmt.Errorf("list audit requests: %w", err)
	}
	defer rows.Close()

	var out []AuditRequest
	for rows.Next() {
		req, err := scanAuditRequest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, req)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) AddAuditFindings(ctx context.Context, findings []AuditFinding) error {
	if len(findings) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO audit_findings (
			audit_id, fingerprint, category, severity, confidence, source, rule_id,
			file_path, line, title, evidence_redacted, metadata_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare audit findings: %w", err)
	}
	defer stmt.Close()

	for _, f := range findings {
		meta := f.MetadataJSON
		if len(meta) == 0 {
			meta = jsonRawObject()
		}
		created := f.CreatedAt
		if created.IsZero() {
			created = time.Now().UTC()
		}
		if _, err := stmt.ExecContext(ctx,
			f.AuditID, f.Fingerprint, f.Category, f.Severity, f.Confidence, f.Source, f.RuleID,
			f.FilePath, f.Line, f.Title, f.EvidenceRedacted, string(meta), formatTime(created),
		); err != nil {
			return fmt.Errorf("insert audit finding: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit audit findings: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ListAuditFindings(ctx context.Context, auditID string) ([]AuditFinding, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, audit_id, fingerprint, category, severity, confidence, source, rule_id,
			file_path, line, title, evidence_redacted, metadata_json, created_at
		FROM audit_findings WHERE audit_id = ?
		ORDER BY
			CASE severity
				WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3
				ELSE 4 END,
			title
	`, auditID)
	if err != nil {
		return nil, fmt.Errorf("list audit findings: %w", err)
	}
	defer rows.Close()

	var out []AuditFinding
	for rows.Next() {
		f, err := scanAuditFinding(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) AddDisclosureReport(ctx context.Context, report DisclosureReport) (DisclosureReport, error) {
	if report.AuditID == "" {
		return DisclosureReport{}, fmt.Errorf("audit id required")
	}
	if report.GeneratedAt.IsZero() {
		report.GeneratedAt = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO disclosure_reports (
			audit_id, finding_id, report_type, sensitivity, title, body_markdown, confidence,
			approved_by_user, submitted_externally, submission_target, submission_notes, generated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		report.AuditID, nullableFindingID(report.FindingID), report.ReportType, report.Sensitivity,
		report.Title, report.BodyMarkdown, report.Confidence,
		boolToInt(report.ApprovedByUser), boolToInt(report.SubmittedExternally),
		report.SubmissionTarget, report.SubmissionNotes, formatTime(report.GeneratedAt),
	)
	if err != nil {
		return DisclosureReport{}, fmt.Errorf("add disclosure report: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return DisclosureReport{}, fmt.Errorf("disclosure report id: %w", err)
	}
	report.ID = id
	return report, nil
}

func (s *SQLiteStore) ListDisclosureReports(ctx context.Context, auditID string) ([]DisclosureReport, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, audit_id, finding_id, report_type, sensitivity, title, body_markdown, confidence,
			approved_by_user, submitted_externally, submission_target, submission_notes, generated_at
		FROM disclosure_reports WHERE audit_id = ?
		ORDER BY generated_at DESC
	`, auditID)
	if err != nil {
		return nil, fmt.Errorf("list disclosure reports: %w", err)
	}
	defer rows.Close()

	var out []DisclosureReport
	for rows.Next() {
		r, err := scanDisclosureReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetDisclosureReport(ctx context.Context, id int64) (DisclosureReport, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, audit_id, finding_id, report_type, sensitivity, title, body_markdown, confidence,
			approved_by_user, submitted_externally, submission_target, submission_notes, generated_at
		FROM disclosure_reports WHERE id = ?
	`, id)
	return scanDisclosureReport(row)
}

func (s *SQLiteStore) MarkDisclosureReportReviewed(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE disclosure_reports SET approved_by_user = 1 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("mark disclosure report reviewed: %w", err)
	}
	return nil
}

type auditRowScanner interface {
	Scan(dest ...any) error
}

func scanAuditRequest(row auditRowScanner) (AuditRequest, error) {
	var req AuditRequest
	var startedAt, finishedAt sql.NullString
	var summary string
	if err := row.Scan(
		&req.AuditID, &req.RepoURL, &req.NormalizedRepoURL, &req.RepoHost, &req.RepoOwner, &req.RepoName,
		&req.CommitSHA, &req.DefaultBranch, &req.AuditDepth, &req.Status, &req.RiskScore, &req.Recommendation,
		&startedAt, &finishedAt, &summary, &req.Error,
	); err != nil {
		return AuditRequest{}, fmt.Errorf("scan audit request: %w", err)
	}
	req.StartedAt = parseTime(startedAt.String)
	if finishedAt.Valid {
		t := parseTime(finishedAt.String)
		req.FinishedAt = &t
	}
	req.SummaryJSON = []byte(summary)
	return req, nil
}

func scanAuditFinding(row auditRowScanner) (AuditFinding, error) {
	var f AuditFinding
	var meta, createdAt string
	if err := row.Scan(
		&f.ID, &f.AuditID, &f.Fingerprint, &f.Category, &f.Severity, &f.Confidence, &f.Source, &f.RuleID,
		&f.FilePath, &f.Line, &f.Title, &f.EvidenceRedacted, &meta, &createdAt,
	); err != nil {
		return AuditFinding{}, fmt.Errorf("scan audit finding: %w", err)
	}
	f.MetadataJSON = []byte(meta)
	f.CreatedAt = parseTime(createdAt)
	return f, nil
}

func scanDisclosureReport(row auditRowScanner) (DisclosureReport, error) {
	var r DisclosureReport
	var findingID sql.NullInt64
	var approved, submitted int
	var generatedAt string
	if err := row.Scan(
		&r.ID, &r.AuditID, &findingID, &r.ReportType, &r.Sensitivity, &r.Title, &r.BodyMarkdown, &r.Confidence,
		&approved, &submitted, &r.SubmissionTarget, &r.SubmissionNotes, &generatedAt,
	); err != nil {
		return DisclosureReport{}, fmt.Errorf("scan disclosure report: %w", err)
	}
	if findingID.Valid {
		id := findingID.Int64
		r.FindingID = &id
	}
	r.ApprovedByUser = intToBool(approved)
	r.SubmittedExternally = intToBool(submitted)
	r.GeneratedAt = parseTime(generatedAt)
	return r, nil
}

func nullableFindingID(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

func jsonRawObject() []byte {
	return []byte("{}")
}
