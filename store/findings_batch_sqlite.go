package store

import (
	"context"
	"fmt"
	"strings"
)

// ListFindingsByIDs loads finding rows in one query (used by reconciliation preview).
func (s *SQLiteStore) ListFindingsByIDs(ctx context.Context, ids []int64) (map[int64]Finding, error) {
	out := make(map[int64]Finding, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := "SELECT id, repository_id, fingerprint, category, severity, confidence, source, rule_id," +
		" package_name, file_path, line, title, status, first_seen_scan_id, last_seen_scan_id, first_seen_at, last_seen_at" +
		" FROM findings WHERE id IN (" + strings.Join(placeholders, ",") + ")"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list findings by ids: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var finding Finding
		var firstSeen, lastSeen string
		if err := rows.Scan(
			&finding.ID,
			&finding.RepositoryID,
			&finding.Fingerprint,
			&finding.Category,
			&finding.Severity,
			&finding.Confidence,
			&finding.Source,
			&finding.RuleID,
			&finding.PackageName,
			&finding.FilePath,
			&finding.Line,
			&finding.Title,
			&finding.Status,
			&finding.FirstSeenScanID,
			&finding.LastSeenScanID,
			&firstSeen,
			&lastSeen,
		); err != nil {
			return nil, err
		}
		finding.FirstSeenAt = parseTime(firstSeen)
		finding.LastSeenAt = parseTime(lastSeen)
		out[finding.ID] = finding
	}
	return out, rows.Err()
}
