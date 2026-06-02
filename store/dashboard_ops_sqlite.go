package store

import (
	"context"
	"database/sql"
	"fmt"
)

func (s *SQLiteStore) enrichOperatorDashboard(ctx context.Context, summary *DashboardSummary) error {
	if err := s.loadFindingBacklog(ctx, summary); err != nil {
		return err
	}
	if err := s.loadScannerPlatformRollups(ctx, summary); err != nil {
		return err
	}
	if err := s.loadScanHealth(ctx, summary); err != nil {
		return err
	}
	return nil
}

func (s *SQLiteStore) loadFindingBacklog(ctx context.Context, summary *DashboardSummary) error {
	b := &summary.Backlog
	b.OpenUnique = summary.OpenFindingsCount
	b.RawDetectorHits7d = summary.IssuesDetectedInScans
	b.CriticalOpen = summary.OpenFindingsBySeverity["critical"]
	b.HighOpen = summary.OpenFindingsBySeverity["high"]
	b.MediumOpen = summary.OpenFindingsBySeverity["medium"]
	b.LowOpen = summary.OpenFindingsBySeverity["low"]

	row := s.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN status = ? AND first_seen_at >= datetime('now', '-7 days') THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = ? AND first_seen_at < datetime('now', '-7 days') AND last_seen_at >= datetime('now', '-7 days') THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status != ? AND status != '' AND status != ? THEN 1 ELSE 0 END), 0)
		FROM findings
	`, FindingStatusOpen, FindingStatusOpen, FindingStatusResolvedVerified, FindingStatusOpen, FindingStatusResolvedVerified)
	if err := row.Scan(&b.NewLast7Days, &b.RegressionsLast7Days, &b.ResolvedVerified, &b.ClosedOther); err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("finding backlog counts: %w", err)
	}

	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM finding_instances WHERE created_at >= datetime('now', '-7 days')
	`).Scan(&b.RawInstances7d); err != nil {
		return fmt.Errorf("raw instances 7d: %w", err)
	}
	return nil
}

func (s *SQLiteStore) loadScannerPlatformRollups(ctx context.Context, summary *DashboardSummary) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT sr.scanner_name,
			SUM(CASE WHEN sr.status = 'binary_missing' THEN 1 ELSE 0 END),
			COUNT(DISTINCT CASE WHEN sr.status = 'binary_missing' THEN sr.scan_id END),
			COUNT(DISTINCT CASE WHEN sr.status = 'binary_missing' THEN s.repository_id END),
			SUM(CASE WHEN sr.status IN ('failed', 'timed_out', 'parse_failed', 'error') THEN 1 ELSE 0 END),
			COUNT(DISTINCT CASE WHEN sr.status IN ('failed', 'timed_out', 'parse_failed', 'error') THEN sr.scan_id END),
			COUNT(DISTINCT CASE WHEN sr.status IN ('failed', 'timed_out', 'parse_failed', 'error') THEN s.repository_id END)
		FROM scanner_results sr
		JOIN scans s ON s.id = sr.scan_id
		GROUP BY sr.scanner_name
	`)
	if err != nil {
		return fmt.Errorf("scanner platform rollups: %w", err)
	}
	defer rows.Close()

	rollups := map[string]scannerDBRollup{}
	for rows.Next() {
		var name string
		var r scannerDBRollup
		var rawMissing, rawFail int
		if err := rows.Scan(&name, &rawMissing, &r.MissingScans, &r.MissingRepos, &rawFail, &r.FailureScans, &r.FailureRepos); err != nil {
			return err
		}
		rollups[name] = r
		summary.Platform.RawMissingEvents += rawMissing
		summary.Platform.RawFailureEvents += rawFail
	}
	summary.platformRollups = rollups

	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT scanner_name) FROM scanner_results WHERE status = 'binary_missing'
	`).Scan(&summary.ScannerToolsMissingCount); err != nil {
		return fmt.Errorf("unique missing tools: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT scanner_name) FROM scanner_results
		WHERE status IN ('failed', 'timed_out', 'parse_failed', 'error')
	`).Scan(&summary.ScannerFailuresCount); err != nil {
		return fmt.Errorf("unique failed scanners: %w", err)
	}
	return rows.Err()
}

func (s *SQLiteStore) loadScanHealth(ctx context.Context, summary *DashboardSummary) error {
	h := &summary.ScanHealth
	h.FailedScans = summary.FailedScansCount

	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM scans WHERE status = 'completed'`).Scan(&h.CompletedScans); err != nil {
		return fmt.Errorf("completed scans: %w", err)
	}

	buckets := map[string]int{}
	failRows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.repository_id, s.error, s.started_at, r.full_name
		FROM scans s
		JOIN repositories r ON r.id = s.repository_id
		WHERE s.status = 'failed'
		ORDER BY s.started_at DESC
		LIMIT 15
	`)
	if err != nil {
		return fmt.Errorf("failed scans: %w", err)
	}
	defer failRows.Close()
	for failRows.Next() {
		var brief FailedScanBrief
		var started string
		if err := failRows.Scan(&brief.ScanID, &brief.RepositoryID, &brief.Error, &started, &brief.RepoFullName); err != nil {
			return err
		}
		brief.StartedAt = parseTime(started)
		brief.Bucket = ClassifyScanFailure(brief.Error)
		buckets[brief.Bucket]++
		h.RecentFailedScans = append(h.RecentFailedScans, brief)
	}
	for bucket, count := range buckets {
		h.FailureBuckets = append(h.FailureBuckets, ScanFailureBucket{
			Bucket: bucket,
			Label:  scanFailureBucketLabel(bucket),
			Count:  count,
		})
	}

	neverRows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.full_name FROM repositories r
		WHERE NOT EXISTS (
			SELECT 1 FROM scans s WHERE s.repository_id = r.id AND s.status = 'completed'
		)
		LIMIT 10
	`)
	if err != nil {
		return fmt.Errorf("never scanned repos: %w", err)
	}
	defer neverRows.Close()
	for neverRows.Next() {
		var brief RepoAttentionBrief
		if err := neverRows.Scan(&brief.RepositoryID, &brief.FullName); err != nil {
			return err
		}
		brief.Reason = "No successful scan recorded"
		h.ReposNeedingAttention = append(h.ReposNeedingAttention, brief)
	}

	staleRows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.full_name, MAX(s.finished_at)
		FROM repositories r
		JOIN scans s ON s.repository_id = r.id AND s.status = 'completed'
		GROUP BY r.id
		HAVING MAX(s.finished_at) < datetime('now', '-30 days')
		LIMIT 10
	`)
	if err != nil {
		return fmt.Errorf("stale scan repos: %w", err)
	}
	defer staleRows.Close()
	for staleRows.Next() {
		var brief RepoAttentionBrief
		var finished string
		if err := staleRows.Scan(&brief.RepositoryID, &brief.FullName, &finished); err != nil {
			return err
		}
		t := parseTime(finished)
		brief.LastScanAt = &t
		brief.Reason = "Last successful scan over 30 days ago"
		h.ReposNeedingAttention = append(h.ReposNeedingAttention, brief)
	}
	return nil
}
