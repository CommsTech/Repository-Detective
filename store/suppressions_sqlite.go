package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func (s *SQLiteStore) CreateFindingSuppression(ctx context.Context, sup FindingSuppression) (FindingSuppression, error) {
	now := time.Now().UTC()
	if sup.CreatedAt.IsZero() {
		sup.CreatedAt = now
	}
	sup.UpdatedAt = now
	sup.Scope = NormalizeSuppressionScope(sup.Scope)
	if sup.Scope == SuppressionScopeGlobal {
		sup.RepositoryID = nil
	}
	if sup.Scope == SuppressionScopeRepo && (sup.RepositoryID == nil || *sup.RepositoryID <= 0) {
		return FindingSuppression{}, fmt.Errorf("repository_id required for repo-scoped suppression")
	}

	var expiresAt sql.NullString
	if sup.ExpiresAt != nil {
		expiresAt = sql.NullString{String: sup.ExpiresAt.UTC().Format(time.RFC3339), Valid: true}
	}
	var repoID sql.NullInt64
	if sup.RepositoryID != nil {
		repoID = sql.NullInt64{Int64: *sup.RepositoryID, Valid: true}
	}
	active := 1
	if !sup.Active {
		active = 0
	}

	res, err := s.db.ExecContext(ctx, `
		INSERT INTO finding_suppressions (
			repository_id, fingerprint, source, rule_id, category, severity,
			scope, reason, created_by, expires_at, active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		repoID, nullIfEmpty(sup.Fingerprint), nullIfEmpty(sup.Source), nullIfEmpty(sup.RuleID),
		nullIfEmpty(sup.Category), nullIfEmpty(sup.Severity),
		sup.Scope, sup.Reason, sup.CreatedBy, expiresAt, active,
		sup.CreatedAt.Format(time.RFC3339), sup.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return FindingSuppression{}, fmt.Errorf("create finding suppression: %w", err)
	}
	id, _ := res.LastInsertId()
	sup.ID = id
	sup.Active = active == 1
	return sup, nil
}

func (s *SQLiteStore) DisableFindingSuppression(ctx context.Context, id int64) (FindingSuppression, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.db.ExecContext(ctx, `
		UPDATE finding_suppressions SET active = 0, updated_at = ? WHERE id = ?`, now, id); err != nil {
		return FindingSuppression{}, fmt.Errorf("disable suppression: %w", err)
	}
	return s.GetFindingSuppression(ctx, id)
}

func (s *SQLiteStore) GetFindingSuppression(ctx context.Context, id int64) (FindingSuppression, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, repository_id, COALESCE(fingerprint,''), COALESCE(source,''), COALESCE(rule_id,''),
			COALESCE(category,''), COALESCE(severity,''), scope, reason, created_by, expires_at,
			active, created_at, updated_at
		FROM finding_suppressions WHERE id = ?`, id)
	return scanFindingSuppression(row)
}

func (s *SQLiteStore) ListFindingSuppressions(ctx context.Context, filter SuppressionFilter) ([]FindingSuppression, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 500 {
		filter.Limit = 500
	}
	query := `
		SELECT id, repository_id, COALESCE(fingerprint,''), COALESCE(source,''), COALESCE(rule_id,''),
			COALESCE(category,''), COALESCE(severity,''), scope, reason, created_by, expires_at,
			active, created_at, updated_at
		FROM finding_suppressions WHERE 1=1`
	args := []any{}
	if filter.RepositoryID > 0 {
		query += ` AND (repository_id = ? OR scope = 'global')`
		args = append(args, filter.RepositoryID)
	}
	if filter.Scope != "" {
		query += ` AND scope = ?`
		args = append(args, strings.ToLower(filter.Scope))
	}
	if filter.ActiveOnly {
		query += ` AND active = 1 AND (expires_at IS NULL OR expires_at > datetime('now'))`
	}
	query += ` ORDER BY updated_at DESC LIMIT ? OFFSET ?`
	args = append(args, filter.Limit, filter.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list suppressions: %w", err)
	}
	defer rows.Close()
	var out []FindingSuppression
	for rows.Next() {
		sup, err := scanFindingSuppression(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sup)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ListActiveSuppressionsForRepository(ctx context.Context, repositoryID int64) ([]FindingSuppression, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, repository_id, COALESCE(fingerprint,''), COALESCE(source,''), COALESCE(rule_id,''),
			COALESCE(category,''), COALESCE(severity,''), scope, reason, created_by, expires_at,
			active, created_at, updated_at
		FROM finding_suppressions
		WHERE active = 1
		  AND (expires_at IS NULL OR expires_at > datetime('now'))
		  AND (
		    scope = 'global'
		    OR (scope = 'repo' AND repository_id = ?)
		  )
		ORDER BY id`, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("list active suppressions: %w", err)
	}
	defer rows.Close()
	now := time.Now().UTC()
	var out []FindingSuppression
	for rows.Next() {
		sup, err := scanFindingSuppression(rows)
		if err != nil {
			return nil, err
		}
		if sup.ExpiresAt != nil && !sup.ExpiresAt.After(now) {
			continue
		}
		out = append(out, sup)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) CountSuppressedFindings(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM findings
		WHERE status IN ('suppressed', 'false_positive')`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count suppressed findings: %w", err)
	}
	return n, nil
}

func scanFindingSuppression(row interface {
	Scan(dest ...any) error
}) (FindingSuppression, error) {
	var sup FindingSuppression
	var repoID sql.NullInt64
	var expiresAt sql.NullString
	var active int
	var createdAt, updatedAt string
	if err := row.Scan(
		&sup.ID, &repoID, &sup.Fingerprint, &sup.Source, &sup.RuleID,
		&sup.Category, &sup.Severity, &sup.Scope, &sup.Reason, &sup.CreatedBy, &expiresAt,
		&active, &createdAt, &updatedAt,
	); err != nil {
		return FindingSuppression{}, err
	}
	if repoID.Valid {
		id := repoID.Int64
		sup.RepositoryID = &id
	}
	if expiresAt.Valid && expiresAt.String != "" {
		t, err := time.Parse(time.RFC3339, expiresAt.String)
		if err == nil {
			sup.ExpiresAt = &t
		}
	}
	sup.Active = active == 1
	sup.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	sup.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return sup, nil
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func (s *SQLiteStore) annotateFindingSuppression(ctx context.Context, items []FindingListItem) []FindingListItem {
	if len(items) == 0 {
		return items
	}
	now := time.Now().UTC()
	byRepo := map[int64][]FindingSuppression{}
	for i := range items {
		st := strings.ToLower(items[i].Status)
		if st == FindingStatusSuppressed || st == FindingStatusFalsePositive {
			items[i].Suppressed = true
			continue
		}
		sups, ok := byRepo[items[i].RepositoryID]
		if !ok {
			var err error
			sups, err = s.ListActiveSuppressionsForRepository(ctx, items[i].RepositoryID)
			if err != nil {
				continue
			}
			byRepo[items[i].RepositoryID] = sups
		}
		in := FindingMatchInput{
			RepositoryID: items[i].RepositoryID,
			Fingerprint:  items[i].Fingerprint,
			Source:       items[i].Source,
			RuleID:       items[i].RuleID,
			Category:     items[i].Category,
			Severity:     items[i].Severity,
		}
		if suppressed, sup := IsSuppressedByList(sups, in, now); suppressed {
			items[i].Suppressed = true
			items[i].SuppressionReason = sup.Reason
		}
	}
	return items
}
