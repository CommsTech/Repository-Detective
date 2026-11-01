package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func (s *SQLiteStore) batchRepoSettings(ctx context.Context, repoIDs []int64) (map[int64]RepoSettings, error) {
	out := make(map[int64]RepoSettings, len(repoIDs))
	if len(repoIDs) == 0 {
		return out, nil
	}
	placeholders, args := inClauseInt64(repoIDs)
	rows, err := s.db.QueryContext(ctx, `
		SELECT repository_id, enabled, schedule_enabled, scan_profile, issue_policy
		FROM repo_settings WHERE repository_id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("batch repo settings: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var enabled, scheduleEnabled sql.NullInt64
		var scanProfile, issuePolicy sql.NullString
		if err := rows.Scan(&id, &enabled, &scheduleEnabled, &scanProfile, &issuePolicy); err != nil {
			return nil, err
		}
		rs := RepoSettings{RepositoryID: id}
		rs.Enabled = nullBoolPtr(enabled)
		rs.ScheduleEnabled = nullBoolPtr(scheduleEnabled)
		if scanProfile.Valid {
			v := scanProfile.String
			rs.ScanProfile = &v
		}
		if issuePolicy.Valid {
			v := issuePolicy.String
			rs.IssuePolicy = &v
		}
		out[id] = rs
	}
	return out, rows.Err()
}

func (s *SQLiteStore) batchRepositoryControlMetrics(ctx context.Context, repoIDs []int64) (map[int64]repoControlMetrics, error) {
	out := make(map[int64]repoControlMetrics, len(repoIDs))
	if len(repoIDs) == 0 {
		return out, nil
	}
	if err := s.batchLastScanMetrics(ctx, repoIDs, out); err != nil {
		return nil, err
	}
	if err := s.batchForgeOpenCounts(ctx, repoIDs, out); err != nil {
		return nil, err
	}
	if err := s.batchReportOnlyFindingCounts(ctx, repoIDs, out); err != nil {
		return nil, err
	}
	if err := s.batchActivePresentCounts(ctx, repoIDs, out); err != nil {
		return nil, err
	}
	if err := s.batchResolvedVerifiedCounts(ctx, repoIDs, out); err != nil {
		return nil, err
	}
	if err := s.batchDuplicateCounts(ctx, repoIDs, out); err != nil {
		return nil, err
	}
	if err := s.batchUnmappedIssueCounts(ctx, repoIDs, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SQLiteStore) batchLastScanMetrics(ctx context.Context, repoIDs []int64, out map[int64]repoControlMetrics) error {
	placeholders, args := inClauseInt64(repoIDs)
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.repository_id, s.id, s.summary_json,
			COALESCE((SELECT COUNT(1) FROM finding_instances fi WHERE fi.scan_id = s.id), 0)
		FROM scans s
		INNER JOIN (
			SELECT repository_id, MAX(started_at) AS max_started FROM scans
			WHERE repository_id IN (`+placeholders+`) GROUP BY repository_id
		) lm ON s.repository_id = lm.repository_id AND s.started_at = lm.max_started
	`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var repoID int64
		var scanID string
		var summaryJSON []byte
		var findingCount int
		if err := rows.Scan(&repoID, &scanID, &summaryJSON, &findingCount); err != nil {
			return err
		}
		m := out[repoID]
		m.LastScanID = scanID
		m.ScanFindingsTotal = findingCount
		pipeline := PipelineStateFromSummary(summaryJSON)
		m.IssueSyncStatus = pipeline.IssueSyncStatus
		if m.ScanFindingsTotal == 0 && pipeline.IssuesFound > 0 {
			m.ScanFindingsTotal = pipeline.IssuesFound
		}
		m.DryRunReportOnly = dryRunFromSummary(summaryJSON)
		out[repoID] = m
	}
	return rows.Err()
}

func (s *SQLiteStore) batchForgeOpenCounts(ctx context.Context, repoIDs []int64, out map[int64]repoControlMetrics) error {
	placeholders, args := inClauseInt64(repoIDs)
	rows, err := s.db.QueryContext(ctx, `
		SELECT f.repository_id, COUNT(1)
		FROM external_issues e
		INNER JOIN findings f ON f.id = e.finding_id
		WHERE f.repository_id IN (`+placeholders+`) AND e.state = 'open'
		GROUP BY f.repository_id
	`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var repoID int64
		var n int
		if err := rows.Scan(&repoID, &n); err != nil {
			return err
		}
		m := out[repoID]
		m.ForgeOpenIssues = n
		out[repoID] = m
	}
	return rows.Err()
}

func (s *SQLiteStore) batchReportOnlyFindingCounts(ctx context.Context, repoIDs []int64, out map[int64]repoControlMetrics) error {
	placeholders, args := inClauseInt64(repoIDs)
	rows, err := s.db.QueryContext(ctx, `
		SELECT repository_id, COUNT(1) FROM findings f
		WHERE repository_id IN (`+placeholders+`) AND status = ?
		AND NOT EXISTS (SELECT 1 FROM external_issues e WHERE e.finding_id = f.id AND e.state = 'open')
		GROUP BY repository_id
	`, append(args, FindingStatusOpen)...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var repoID int64
		var n int
		if err := rows.Scan(&repoID, &n); err != nil {
			return err
		}
		m := out[repoID]
		m.ReportOnlyFindings = n
		out[repoID] = m
	}
	return rows.Err()
}

func (s *SQLiteStore) batchActivePresentCounts(ctx context.Context, repoIDs []int64, out map[int64]repoControlMetrics) error {
	scanIDs := make([]string, 0, len(repoIDs))
	scanToRepo := make(map[string]int64, len(repoIDs))
	for _, id := range repoIDs {
		sid := out[id].LastScanID
		if sid == "" {
			continue
		}
		scanIDs = append(scanIDs, sid)
		scanToRepo[sid] = id
	}
	if len(scanIDs) == 0 {
		return nil
	}
	placeholders, args := inClauseStrings(scanIDs)
	// Prefer EXISTS over joining findings — the join plan was ~10x slower on large instance tables.
	rows, err := s.db.QueryContext(ctx, `
		SELECT fi.scan_id, COUNT(1)
		FROM finding_instances fi
		WHERE fi.scan_id IN (`+placeholders+`)
		  AND EXISTS (
			SELECT 1 FROM findings f
			WHERE f.id = fi.finding_id AND f.status = ?
		  )
		GROUP BY fi.scan_id
	`, append(args, FindingStatusOpen)...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var scanID string
		var n int
		if err := rows.Scan(&scanID, &n); err != nil {
			return err
		}
		repoID, ok := scanToRepo[scanID]
		if !ok {
			continue
		}
		m := out[repoID]
		m.ActivePresentOpen = n
		out[repoID] = m
	}
	return rows.Err()
}

func (s *SQLiteStore) batchResolvedVerifiedCounts(ctx context.Context, repoIDs []int64, out map[int64]repoControlMetrics) error {
	placeholders, args := inClauseInt64(repoIDs)
	rows, err := s.db.QueryContext(ctx, `
		SELECT f.repository_id, COUNT(1)
		FROM findings f
		INNER JOIN external_issues e ON e.finding_id = f.id AND e.state = 'open'
		WHERE f.repository_id IN (`+placeholders+`) AND f.status = ?
		GROUP BY f.repository_id
	`, append(args, FindingStatusResolvedVerified)...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var repoID int64
		var n int
		if err := rows.Scan(&repoID, &n); err != nil {
			return err
		}
		m := out[repoID]
		m.ResolvedVerified = n
		out[repoID] = m
	}
	return rows.Err()
}

func (s *SQLiteStore) batchDuplicateCounts(ctx context.Context, repoIDs []int64, out map[int64]repoControlMetrics) error {
	placeholders, args := inClauseInt64(repoIDs)
	rows, err := s.db.QueryContext(ctx, `
		SELECT repository_id, COUNT(1) FROM findings
		WHERE repository_id IN (`+placeholders+`) AND canonical_finding_id IS NOT NULL
		GROUP BY repository_id
	`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var repoID int64
		var n int
		if err := rows.Scan(&repoID, &n); err != nil {
			return err
		}
		m := out[repoID]
		m.Duplicates = n
		out[repoID] = m
	}
	return rows.Err()
}

func (s *SQLiteStore) batchUnmappedIssueCounts(ctx context.Context, repoIDs []int64, out map[int64]repoControlMetrics) error {
	scanIDs := make([]string, 0, len(repoIDs))
	for _, id := range repoIDs {
		if sid := out[id].LastScanID; sid != "" {
			scanIDs = append(scanIDs, sid)
		}
	}
	placeholders, args := inClauseInt64(repoIDs)
	query := `
		SELECT f.repository_id, COUNT(1)
		FROM external_issues e
		INNER JOIN findings f ON f.id = e.finding_id
		WHERE f.repository_id IN (` + placeholders + `) AND e.state = 'open'
	`
	if len(scanIDs) == 0 {
		query += ` GROUP BY f.repository_id`
	} else {
		scanPH, scanArgs := inClauseStrings(scanIDs)
		query += `
		AND NOT EXISTS (
			SELECT 1 FROM finding_instances fi
			WHERE fi.finding_id = f.id AND fi.scan_id IN (` + scanPH + `)
		)
		GROUP BY f.repository_id`
		args = append(args, scanArgs...)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var repoID int64
		var n int
		if err := rows.Scan(&repoID, &n); err != nil {
			return err
		}
		m := out[repoID]
		m.UnmappedOpenIssues = n
		out[repoID] = m
	}
	return rows.Err()
}

func inClauseInt64(ids []int64) (string, []any) {
	parts := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		parts[i] = "?"
		args[i] = id
	}
	return strings.Join(parts, ","), args
}

func inClauseStrings(values []string) (string, []any) {
	parts := make([]string, len(values))
	args := make([]any, len(values))
	for i, v := range values {
		parts[i] = "?"
		args[i] = v
	}
	return strings.Join(parts, ","), args
}
