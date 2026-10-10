package store

import (
	"context"
	"database/sql"
	"fmt"
)

// GetExternalIssueByFingerprint returns the newest open external issue for a repo fingerprint.
func (s *SQLiteStore) GetExternalIssueByFingerprint(ctx context.Context, repositoryID int64, forgeType, fingerprint string) (ExternalIssue, error) {
	if repositoryID <= 0 || fingerprint == "" {
		return ExternalIssue{}, sql.ErrNoRows
	}
	forgeType = normalizeForgeType(forgeType)
	var issue ExternalIssue
	var createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT ei.id, ei.finding_id, ei.forge_type, ei.issue_number, ei.issue_url, ei.state, ei.created_at, ei.updated_at
		FROM external_issues ei
		JOIN findings f ON f.id = ei.finding_id
		WHERE f.repository_id = ? AND f.fingerprint = ? AND ei.forge_type = ?
		ORDER BY CASE WHEN ei.state = 'open' THEN 0 ELSE 1 END, ei.updated_at DESC
		LIMIT 1
	`, repositoryID, fingerprint, forgeType).Scan(
		&issue.ID, &issue.FindingID, &issue.ForgeType, &issue.IssueNumber, &issue.IssueURL, &issue.State, &createdAt, &updatedAt,
	)
	if err != nil {
		return ExternalIssue{}, err
	}
	issue.CreatedAt = parseTime(createdAt)
	issue.UpdatedAt = parseTime(updatedAt)
	return issue, nil
}

func normalizeForgeType(forgeType string) string {
	switch forgeType {
	case ForgeTypeGitHub:
		return ForgeTypeGitHub
	default:
		return ForgeTypeGitea
	}
}

// GetExternalIssueByIssueNumber returns a mapping for a forge issue number in a repository.
func (s *SQLiteStore) GetExternalIssueByIssueNumber(ctx context.Context, repositoryID int64, forgeType string, issueNumber int) (ExternalIssue, error) {
	if repositoryID <= 0 || issueNumber <= 0 {
		return ExternalIssue{}, sql.ErrNoRows
	}
	forgeType = normalizeForgeType(forgeType)
	var issue ExternalIssue
	var createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT ei.id, ei.finding_id, ei.forge_type, ei.issue_number, ei.issue_url, ei.state, ei.created_at, ei.updated_at
		FROM external_issues ei
		JOIN findings f ON f.id = ei.finding_id
		WHERE f.repository_id = ? AND ei.forge_type = ? AND ei.issue_number = ?
		LIMIT 1
	`, repositoryID, forgeType, issueNumber).Scan(
		&issue.ID, &issue.FindingID, &issue.ForgeType, &issue.IssueNumber, &issue.IssueURL, &issue.State, &createdAt, &updatedAt,
	)
	if err != nil {
		return ExternalIssue{}, err
	}
	issue.CreatedAt = parseTime(createdAt)
	issue.UpdatedAt = parseTime(updatedAt)
	return issue, nil
}

func (s *SQLiteStore) ListExternalIssueNumbersByRepository(ctx context.Context, repositoryID int64, forgeType string) (map[int]int64, error) {
	forgeType = normalizeForgeType(forgeType)
	rows, err := s.db.QueryContext(ctx, `
		SELECT ei.issue_number, ei.finding_id
		FROM external_issues ei
		JOIN findings f ON f.id = ei.finding_id
		WHERE f.repository_id = ? AND ei.forge_type = ?
	`, repositoryID, forgeType)
	if err != nil {
		return nil, fmt.Errorf("list external issue numbers: %w", err)
	}
	defer rows.Close()
	out := map[int]int64{}
	for rows.Next() {
		var num int
		var findingID int64
		if err := rows.Scan(&num, &findingID); err != nil {
			return nil, err
		}
		out[num] = findingID
	}
	return out, rows.Err()
}
