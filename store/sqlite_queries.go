package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (s *SQLiteStore) GetRepository(ctx context.Context, id int64) (Repository, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+repositoryColumns+` FROM repositories WHERE id = ?`, id)
	repo, err := scanRepository(row)
	if err != nil {
		return Repository{}, fmt.Errorf("get repository: %w", err)
	}
	return repo, nil
}

func (s *SQLiteStore) ListRepositoriesWithSummary(ctx context.Context, opts ListOptions) ([]RepositorySummary, error) {
	opts = NormalizeListOptions(opts)
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.forge_type, r.owner, r.name, r.full_name, r.clone_url, r.default_branch,
			r.connected_repo, r.created_at, r.updated_at,
			ls.started_at, ls.status,
			COALESCE(fc.open_count, 0), COALESCE(fc.total_count, 0)
		FROM repositories r
		LEFT JOIN (
			SELECT repository_id, started_at, status
			FROM scans s1
			WHERE started_at = (
				SELECT MAX(started_at) FROM scans s2 WHERE s2.repository_id = s1.repository_id
			)
		) ls ON ls.repository_id = r.id
		LEFT JOIN (
			SELECT repository_id,
				SUM(CASE WHEN status = 'open' THEN 1 ELSE 0 END) AS open_count,
				COUNT(1) AS total_count
			FROM findings GROUP BY repository_id
		) fc ON fc.repository_id = r.id
		ORDER BY r.full_name
		LIMIT ? OFFSET ?
	`, opts.Limit, opts.Offset)
	if err != nil {
		return nil, fmt.Errorf("list repositories with summary: %w", err)
	}
	defer rows.Close()

	var out []RepositorySummary
	for rows.Next() {
		var summary RepositorySummary
		var connected int
		var createdAt, updatedAt string
		var lastScanAt sql.NullString
		var lastScanStatus sql.NullString
		if err := rows.Scan(
			&summary.ID, &summary.ForgeType, &summary.Owner, &summary.Name, &summary.FullName,
			&summary.CloneURL, &summary.DefaultBranch, &connected, &createdAt, &updatedAt,
			&lastScanAt, &lastScanStatus, &summary.OpenFindingsCount, &summary.TotalFindingsCount,
		); err != nil {
			return nil, fmt.Errorf("scan repository summary: %w", err)
		}
		summary.ConnectedRepo = intToBool(connected)
		summary.CreatedAt = parseTime(createdAt)
		summary.UpdatedAt = parseTime(updatedAt)
		if lastScanAt.Valid {
			t := parseTime(lastScanAt.String)
			summary.LastScanAt = &t
		}
		if lastScanStatus.Valid {
			summary.LastScanStatus = lastScanStatus.String
		}
		out = append(out, summary)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ListScansByRepository(ctx context.Context, repositoryID int64, opts ListOptions) ([]Scan, error) {
	opts = NormalizeListOptions(opts)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, repository_id, trigger_type, ref, commit_sha, pr_number,
			workspace_mode_used, commit_pinned, status, started_at, finished_at, summary_json, error
		FROM scans WHERE repository_id = ?
		ORDER BY started_at DESC
		LIMIT ? OFFSET ?
	`, repositoryID, opts.Limit, opts.Offset)
	if err != nil {
		return nil, fmt.Errorf("list scans: %w", err)
	}
	defer rows.Close()
	return scanScanRows(rows)
}

func (s *SQLiteStore) ListScannerResultsByScan(ctx context.Context, scanID string) ([]ScannerResultRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, scan_id, scanner_name, status, findings_count, duration_ms, detail, error
		FROM scanner_results WHERE scan_id = ?
		ORDER BY scanner_name
	`, scanID)
	if err != nil {
		return nil, fmt.Errorf("list scanner results: %w", err)
	}
	defer rows.Close()

	var out []ScannerResultRecord
	for rows.Next() {
		var rec ScannerResultRecord
		var duration sql.NullInt64
		if err := rows.Scan(&rec.ID, &rec.ScanID, &rec.ScannerName, &rec.Status, &rec.FindingsCount, &duration, &rec.Detail, &rec.Error); err != nil {
			return nil, fmt.Errorf("scan scanner result: %w", err)
		}
		if duration.Valid {
			rec.DurationMS = duration.Int64
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// ListRecentScannerFailures returns recent scanner_results rows that indicate tool/run failures.
func (s *SQLiteStore) ListRecentScannerFailures(ctx context.Context, limit int) ([]ScannerFailureEvent, error) {
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT sr.scanner_name, sr.status, COALESCE(sr.error, ''), COALESCE(sr.detail, ''),
			sr.scan_id, s.repository_id, r.full_name, s.started_at, COALESCE(sr.duration_ms, 0)
		FROM scanner_results sr
		JOIN scans s ON s.id = sr.scan_id
		JOIN repositories r ON r.id = s.repository_id
		WHERE sr.status IN ('failed', 'timed_out', 'parse_failed', 'error')
		ORDER BY s.started_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent scanner failures: %w", err)
	}
	defer rows.Close()

	var out []ScannerFailureEvent
	for rows.Next() {
		var ev ScannerFailureEvent
		var started string
		if err := rows.Scan(&ev.ScannerName, &ev.Status, &ev.Error, &ev.Detail, &ev.ScanID,
			&ev.RepositoryID, &ev.RepoFullName, &started, &ev.DurationMS); err != nil {
			return nil, fmt.Errorf("scan scanner failure row: %w", err)
		}
		ev.StartedAt = parseTime(started)
		out = append(out, ev)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ListFindings(ctx context.Context, filter FindingFilter) ([]FindingListItem, error) {
	filter.Limit = NormalizeListOptions(ListOptions{Limit: filter.Limit, Offset: filter.Offset}).Limit
	filter.Offset = NormalizeListOptions(ListOptions{Offset: filter.Offset}).Offset

	query := `
		SELECT f.id, f.repository_id, f.fingerprint, f.category, f.severity, f.confidence,
			f.source, f.rule_id, f.package_name, f.file_path, f.line, f.title, f.status,
			f.first_seen_scan_id, f.last_seen_scan_id, f.first_seen_at, f.last_seen_at,
			r.full_name,
			COALESCE(ei.issue_number, 0), COALESCE(ei.issue_url, '')
		FROM findings f
		JOIN repositories r ON r.id = f.repository_id
		LEFT JOIN external_issues ei ON ei.finding_id = f.id
		WHERE 1=1`
	args := []any{}

	if filter.RepositoryID > 0 {
		query += ` AND f.repository_id = ?`
		args = append(args, filter.RepositoryID)
	}
	if filter.Severity != "" {
		query += ` AND LOWER(f.severity) = LOWER(?)`
		args = append(args, filter.Severity)
	}
	if filter.Category != "" {
		query += ` AND LOWER(f.category) = LOWER(?)`
		args = append(args, filter.Category)
	}
	if filter.Status != "" {
		query += ` AND LOWER(f.status) = LOWER(?)`
		args = append(args, filter.Status)
	}
	if filter.Source != "" {
		query += ` AND LOWER(f.source) = LOWER(?)`
		args = append(args, filter.Source)
	}
	if filter.OnlySuppressed {
		query += ` AND f.status IN ('suppressed', 'false_positive')`
	} else if !filter.IncludeSuppressed {
		query += ` AND f.status NOT IN ('suppressed', 'false_positive')`
	}
	query += ` ORDER BY f.last_seen_at DESC LIMIT ? OFFSET ?`
	args = append(args, filter.Limit, filter.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list findings: %w", err)
	}
	defer rows.Close()

	var out []FindingListItem
	for rows.Next() {
		item, err := scanFindingListItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return s.annotateFindingSuppression(ctx, out), nil
}

func (s *SQLiteStore) CountFindings(ctx context.Context, filter FindingFilter) (int, error) {
	query := `SELECT COUNT(1) FROM findings f WHERE 1=1`
	args := []any{}

	if filter.RepositoryID > 0 {
		query += ` AND f.repository_id = ?`
		args = append(args, filter.RepositoryID)
	}
	if filter.Severity != "" {
		query += ` AND LOWER(f.severity) = LOWER(?)`
		args = append(args, filter.Severity)
	}
	if filter.Category != "" {
		query += ` AND LOWER(f.category) = LOWER(?)`
		args = append(args, filter.Category)
	}
	if filter.Status != "" {
		query += ` AND LOWER(f.status) = LOWER(?)`
		args = append(args, filter.Status)
	}
	if filter.Source != "" {
		query += ` AND LOWER(f.source) = LOWER(?)`
		args = append(args, filter.Source)
	}
	if filter.OnlySuppressed {
		query += ` AND f.status IN ('suppressed', 'false_positive')`
	} else if !filter.IncludeSuppressed {
		query += ` AND f.status NOT IN ('suppressed', 'false_positive')`
	}

	var n int
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("count findings: %w", err)
	}
	return n, nil
}

func (s *SQLiteStore) OpenFindingsBySeverityForRepository(ctx context.Context, repositoryID int64) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT LOWER(severity), COUNT(1) FROM findings
		WHERE repository_id = ? AND status = 'open'
		GROUP BY LOWER(severity)
	`, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("open findings by severity: %w", err)
	}
	defer rows.Close()

	out := map[string]int{}
	for rows.Next() {
		var sev string
		var count int
		if err := rows.Scan(&sev, &count); err != nil {
			return nil, err
		}
		out[sev] = count
	}
	return out, rows.Err()
}

func (s *SQLiteStore) OpenFindingsByCategoryForRepository(ctx context.Context, repositoryID int64) (map[string]int, error) {
	byRepo, err := s.OpenFindingsByCategoryForRepositories(ctx, []int64{repositoryID})
	if err != nil {
		return nil, err
	}
	out := byRepo[repositoryID]
	if out == nil {
		out = map[string]int{}
	}
	return out, nil
}

// OpenFindingsByCategoryForRepositories returns open finding counts by category for many repos in one query.
func (s *SQLiteStore) OpenFindingsByCategoryForRepositories(ctx context.Context, repositoryIDs []int64) (map[int64]map[string]int, error) {
	out := make(map[int64]map[string]int, len(repositoryIDs))
	if len(repositoryIDs) == 0 {
		return out, nil
	}
	placeholders, args := inClauseInt64(repositoryIDs)
	rows, err := s.db.QueryContext(ctx, `
		SELECT repository_id, LOWER(COALESCE(category, 'unknown')), COUNT(1) FROM findings
		WHERE repository_id IN (`+placeholders+`) AND status = 'open'
		GROUP BY repository_id, LOWER(COALESCE(category, 'unknown'))
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("open findings by category batch: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var repoID int64
		var cat string
		var count int
		if err := rows.Scan(&repoID, &cat, &count); err != nil {
			return nil, err
		}
		m := out[repoID]
		if m == nil {
			m = map[string]int{}
			out[repoID] = m
		}
		m[cat] = count
	}
	return out, rows.Err()
}

func (s *SQLiteStore) OpenFindingsConfidenceBandsForRepository(ctx context.Context, repositoryID int64, confidenceGate float64) (map[string]int, error) {
	if confidenceGate <= 0 {
		confidenceGate = 0.7
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT
			SUM(CASE WHEN confidence >= ? OR LOWER(severity) IN ('critical', 'high') AND confidence >= 0.5 THEN 1 ELSE 0 END),
			SUM(CASE WHEN NOT (confidence >= ? OR LOWER(severity) IN ('critical', 'high') AND confidence >= 0.5) THEN 1 ELSE 0 END),
			COUNT(1)
		FROM findings
		WHERE repository_id = ? AND status = 'open'
	`, confidenceGate, confidenceGate, repositoryID)
	var actionable, review, total int
	if err := row.Scan(&actionable, &review, &total); err != nil {
		return nil, fmt.Errorf("open findings confidence bands: %w", err)
	}
	return map[string]int{
		"actionable": actionable,
		"review":     review,
		"total":      total,
	}, nil
}

func (s *SQLiteStore) GetFindingDetail(ctx context.Context, id int64) (FindingDetail, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT f.id, f.repository_id, f.fingerprint, f.category, f.severity, f.confidence,
			f.source, f.rule_id, f.package_name, f.file_path, f.line, f.title, f.status,
			f.first_seen_scan_id, f.last_seen_scan_id, f.first_seen_at, f.last_seen_at,
			r.full_name,
			COALESCE(ei.issue_number, 0), COALESCE(ei.issue_url, '')
		FROM findings f
		JOIN repositories r ON r.id = f.repository_id
		LEFT JOIN external_issues ei ON ei.finding_id = f.id
		WHERE f.id = ?
	`, id)

	item, err := scanFindingListItem(row)
	if err != nil {
		return FindingDetail{}, fmt.Errorf("get finding: %w", err)
	}
	annotated := s.annotateFindingSuppression(ctx, []FindingListItem{item})
	if len(annotated) > 0 {
		item = annotated[0]
	}

	instances, err := s.listFindingInstances(ctx, id)
	if err != nil {
		return FindingDetail{}, err
	}
	external, err := s.ListExternalIssuesByFinding(ctx, id)
	if err != nil {
		return FindingDetail{}, err
	}
	lifecycle, err := s.ListLifecycleEventsByFinding(ctx, id)
	if err != nil {
		return FindingDetail{}, err
	}

	return FindingDetail{
		FindingListItem: item,
		Instances:       instances,
		ExternalIssues:  external,
		LifecycleEvents: lifecycle,
	}, nil
}

func (s *SQLiteStore) listFindingInstances(ctx context.Context, findingID int64) ([]FindingInstance, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, finding_id, scan_id, evidence_redacted, location_json, raw_metadata_json, created_at
		FROM finding_instances WHERE finding_id = ?
		ORDER BY created_at DESC
	`, findingID)
	if err != nil {
		return nil, fmt.Errorf("list finding instances: %w", err)
	}
	defer rows.Close()

	var out []FindingInstance
	for rows.Next() {
		var inst FindingInstance
		var location, meta, createdAt string
		if err := rows.Scan(&inst.ID, &inst.FindingID, &inst.ScanID, &inst.EvidenceRedacted, &location, &meta, &createdAt); err != nil {
			return nil, fmt.Errorf("scan finding instance: %w", err)
		}
		inst.LocationJSON = json.RawMessage(location)
		inst.RawMetadataJSON = json.RawMessage(meta)
		inst.CreatedAt = parseTime(createdAt)
		out = append(out, inst)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ListLifecycleEventsByFinding(ctx context.Context, findingID int64) ([]LifecycleEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, finding_id, scan_id, event_type, message, metadata_json, created_at
		FROM lifecycle_events WHERE finding_id = ?
		ORDER BY created_at DESC
	`, findingID)
	if err != nil {
		return nil, fmt.Errorf("list lifecycle events: %w", err)
	}
	defer rows.Close()
	return scanLifecycleRows(rows)
}

func (s *SQLiteStore) ListExternalIssuesByRepository(ctx context.Context, repositoryID int64, opts ListOptions) ([]ExternalIssue, error) {
	opts = NormalizeListOptions(opts)
	rows, err := s.db.QueryContext(ctx, `
		SELECT ei.id, ei.finding_id, ei.forge_type, ei.issue_number, ei.issue_url, ei.state, ei.created_at, ei.updated_at
		FROM external_issues ei
		JOIN findings f ON f.id = ei.finding_id
		WHERE f.repository_id = ?
		ORDER BY ei.updated_at DESC
		LIMIT ? OFFSET ?
	`, repositoryID, opts.Limit, opts.Offset)
	if err != nil {
		return nil, fmt.Errorf("list external issues by repo: %w", err)
	}
	defer rows.Close()
	return scanExternalIssueRows(rows)
}

func (s *SQLiteStore) ListExternalIssuesByFinding(ctx context.Context, findingID int64) ([]ExternalIssue, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, finding_id, forge_type, issue_number, issue_url, state, created_at, updated_at
		FROM external_issues WHERE finding_id = ?
		ORDER BY updated_at DESC
	`, findingID)
	if err != nil {
		return nil, fmt.Errorf("list external issues: %w", err)
	}
	defer rows.Close()
	return scanExternalIssueRows(rows)
}

func (s *SQLiteStore) ListRecentScans(ctx context.Context, opts ListOptions) ([]ScanWithRepo, error) {
	opts = NormalizeListOptions(opts)
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.repository_id, s.trigger_type, s.ref, s.commit_sha, s.pr_number,
			s.workspace_mode_used, s.commit_pinned, s.status, s.started_at, s.finished_at, s.summary_json, s.error,
			r.full_name
		FROM scans s
		JOIN repositories r ON r.id = s.repository_id
		ORDER BY s.started_at DESC
		LIMIT ? OFFSET ?
	`, opts.Limit, opts.Offset)
	if err != nil {
		return nil, fmt.Errorf("list recent scans: %w", err)
	}
	defer rows.Close()

	var out []ScanWithRepo
	for rows.Next() {
		var item ScanWithRepo
		var commitPinned int
		var startedAt string
		var finishedAt sql.NullString
		var summaryJSON string
		if err := rows.Scan(
			&item.ID, &item.RepositoryID, &item.TriggerType, &item.Ref, &item.CommitSHA, &item.PRNumber,
			&item.WorkspaceModeUsed, &commitPinned, &item.Status, &startedAt, &finishedAt, &summaryJSON, &item.Error,
			&item.RepoFullName,
		); err != nil {
			return nil, fmt.Errorf("scan recent scan row: %w", err)
		}
		item.CommitPinned = intToBool(commitPinned)
		item.StartedAt = parseTime(startedAt)
		item.FinishedAt = parseTimePtr(finishedAt)
		item.SummaryJSON = json.RawMessage(summaryJSON)
		out = append(out, item)
	}
	return out, rows.Err()
}

// CountCompletedScansByDay returns completed scan counts keyed by UTC day (YYYY-MM-DD)
// for scans started on or after since (inclusive, UTC calendar day).
func (s *SQLiteStore) CountCompletedScansByDay(ctx context.Context, since time.Time) (map[string]int, error) {
	if since.IsZero() {
		since = time.Now().UTC().AddDate(0, 0, -13)
	}
	sinceUTC := since.UTC()
	sinceDay := time.Date(sinceUTC.Year(), sinceUTC.Month(), sinceUTC.Day(), 0, 0, 0, 0, time.UTC)
	sinceStr := formatTime(sinceDay)
	rows, err := s.db.QueryContext(ctx, `
		SELECT substr(started_at, 1, 10) AS day, COUNT(1)
		FROM scans
		WHERE lower(status) = 'completed'
		  AND started_at >= ?
		GROUP BY substr(started_at, 1, 10)
		ORDER BY day
	`, sinceStr)
	if err != nil {
		return nil, fmt.Errorf("count completed scans by day: %w", err)
	}
	defer rows.Close()

	out := map[string]int{}
	for rows.Next() {
		var day string
		var n int
		if err := rows.Scan(&day, &n); err != nil {
			return nil, fmt.Errorf("scan completed-by-day row: %w", err)
		}
		day = strings.TrimSpace(day)
		if day == "" {
			continue
		}
		out[day] = n
	}
	return out, rows.Err()
}

// CountAutoRemediatedFindingsByDay returns distinct findings that landed an auto-remediation
// PR (opened or merged), keyed by UTC day of merge/update/create.
func (s *SQLiteStore) CountAutoRemediatedFindingsByDay(ctx context.Context, since time.Time) (map[string]int, error) {
	if since.IsZero() {
		since = time.Now().UTC().AddDate(0, 0, -13)
	}
	sinceUTC := since.UTC()
	sinceDay := time.Date(sinceUTC.Year(), sinceUTC.Month(), sinceUTC.Day(), 0, 0, 0, 0, time.UTC)
	sinceStr := formatTime(sinceDay)
	rows, err := s.db.QueryContext(ctx, `
		SELECT substr(
			CASE
				WHEN lower(status) = 'pr_merged' AND COALESCE(merged_at, '') != '' THEN merged_at
				ELSE created_at
			END, 1, 10) AS day,
			COUNT(DISTINCT COALESCE(finding_id, id))
		FROM patch_attempts
		WHERE lower(status) IN ('pr_opened', 'pr_merged')
		  AND CASE
				WHEN lower(status) = 'pr_merged' AND COALESCE(merged_at, '') != '' THEN merged_at
				ELSE created_at
			  END >= ?
		GROUP BY day
		ORDER BY day
	`, sinceStr)
	if err != nil {
		return nil, fmt.Errorf("count auto-remediated findings by day: %w", err)
	}
	defer rows.Close()

	out := map[string]int{}
	for rows.Next() {
		var day string
		var n int
		if err := rows.Scan(&day, &n); err != nil {
			return nil, fmt.Errorf("scan auto-remediated-by-day row: %w", err)
		}
		day = strings.TrimSpace(day)
		if day == "" {
			continue
		}
		out[day] = n
	}
	return out, rows.Err()
}

// CountRemediationPlansByDay returns remediation plans created per UTC day.
func (s *SQLiteStore) CountRemediationPlansByDay(ctx context.Context, since time.Time) (map[string]int, error) {
	if since.IsZero() {
		since = time.Now().UTC().AddDate(0, 0, -13)
	}
	sinceUTC := since.UTC()
	sinceDay := time.Date(sinceUTC.Year(), sinceUTC.Month(), sinceUTC.Day(), 0, 0, 0, 0, time.UTC)
	sinceStr := formatTime(sinceDay)
	rows, err := s.db.QueryContext(ctx, `
		SELECT substr(created_at, 1, 10) AS day, COUNT(1)
		FROM remediation_plans
		WHERE created_at >= ?
		GROUP BY substr(created_at, 1, 10)
		ORDER BY day
	`, sinceStr)
	if err != nil {
		return nil, fmt.Errorf("count remediation plans by day: %w", err)
	}
	defer rows.Close()

	out := map[string]int{}
	for rows.Next() {
		var day string
		var n int
		if err := rows.Scan(&day, &n); err != nil {
			return nil, fmt.Errorf("scan remediation-plans-by-day row: %w", err)
		}
		day = strings.TrimSpace(day)
		if day == "" {
			continue
		}
		out[day] = n
	}
	return out, rows.Err()
}

func (s *SQLiteStore) CountActiveScans(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM scans WHERE status IN ('running', 'started')
	`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active scans: %w", err)
	}
	return count, nil
}

func (s *SQLiteStore) DashboardSummary(ctx context.Context, recentLimit int) (DashboardSummary, error) {
	if recentLimit <= 0 {
		recentLimit = 10
	}
	if recentLimit > 50 {
		recentLimit = 50
	}

	if cached, ok := s.getCachedDashboardSummary(recentLimit); ok {
		return cached, nil
	}

	summary, err := s.buildDashboardSummary(ctx, recentLimit)
	if err != nil {
		return summary, err
	}
	s.putCachedDashboardSummary(recentLimit, summary)
	return summary, nil
}

func (s *SQLiteStore) getCachedDashboardSummary(recentLimit int) (DashboardSummary, bool) {
	s.dashboardSummaryMu.Lock()
	defer s.dashboardSummaryMu.Unlock()
	if s.dashboardSummaryCache == nil {
		return DashboardSummary{}, false
	}
	entry, ok := s.dashboardSummaryCache[recentLimit]
	if !ok || time.Now().After(entry.expiresAt) {
		return DashboardSummary{}, false
	}
	return entry.summary, true
}

func (s *SQLiteStore) putCachedDashboardSummary(recentLimit int, summary DashboardSummary) {
	s.dashboardSummaryMu.Lock()
	defer s.dashboardSummaryMu.Unlock()
	if s.dashboardSummaryCache == nil {
		s.dashboardSummaryCache = make(map[int]dashboardSummaryCacheEntry)
	}
	s.dashboardSummaryCache[recentLimit] = dashboardSummaryCacheEntry{
		summary:   summary,
		expiresAt: time.Now().Add(dashboardSummaryCacheTTL),
	}
}

func (s *SQLiteStore) buildDashboardSummary(ctx context.Context, recentLimit int) (DashboardSummary, error) {
	var summary DashboardSummary
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM repositories`).Scan(&summary.TotalRepositories); err != nil {
		return summary, fmt.Errorf("count repositories: %w", err)
	}

	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM scans WHERE status = 'failed'`).Scan(&summary.FailedScansCount); err != nil {
		return summary, fmt.Errorf("count failed scans: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM scans WHERE status = 'failed'
		  AND (error LIKE '%stale%reaped%' OR error LIKE '%interrupted by process restart%')
	`).Scan(&summary.StaleReapedScansCount); err != nil {
		return summary, fmt.Errorf("count stale reaped scans: %w", err)
	}
	// Actionable = recent non-noise failures that have not been superseded by a
	// later successful scan for the same repo. Lifetime totals stay in FailedScansCount.
	// Historical forge outages produced hundreds of "no valid ref" rows that no longer
	// reflect current fleet health once repos scan successfully again.
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM scans s
		WHERE s.status = 'failed'
		  AND s.started_at >= datetime('now', '-14 days')
		  AND NOT (COALESCE(s.error, '') LIKE '%stale%reaped%'
		           OR COALESCE(s.error, '') LIKE '%interrupted by process restart%')
		  AND NOT EXISTS (
			SELECT 1 FROM scans later
			WHERE later.repository_id = s.repository_id
			  AND later.status = 'completed'
			  AND later.started_at > s.started_at
		  )
	`).Scan(&summary.ActionableFailedScansCount); err != nil {
		return summary, fmt.Errorf("count actionable failed scans: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM repositories r
		WHERE (
			SELECT s.status FROM scans s
			WHERE s.repository_id = r.id
			ORDER BY s.started_at DESC LIMIT 1
		) = 'failed'
		AND NOT (
			SELECT COALESCE(s.error, '') FROM scans s
			WHERE s.repository_id = r.id
			ORDER BY s.started_at DESC LIMIT 1
		) LIKE '%stale%reaped%'
		AND NOT (
			SELECT COALESCE(s.error, '') FROM scans s
			WHERE s.repository_id = r.id
			ORDER BY s.started_at DESC LIMIT 1
		) LIKE '%interrupted by process restart%'
	`).Scan(&summary.UnhealthyReposCount); err != nil {
		return summary, fmt.Errorf("count unhealthy repos: %w", err)
	}

	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM findings WHERE status = 'open'
	`).Scan(&summary.OpenFindingsCount); err != nil {
		return summary, fmt.Errorf("count open findings: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM findings WHERE status IN ('suppressed', 'false_positive')
	`).Scan(&summary.SuppressedFindingsCount); err != nil {
		return summary, fmt.Errorf("count suppressed findings: %w", err)
	}

	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(CAST(json_extract(summary_json, '$.issues_found') AS INTEGER)), 0)
		FROM scans
		WHERE status = 'completed' AND started_at >= datetime('now', '-7 days')
	`).Scan(&summary.IssuesDetectedInScans); err != nil {
		return summary, fmt.Errorf("sum issues in scans: %w", err)
	}

	summary.OpenFindingsBySeverity = map[string]int{}
	rows, err := s.db.QueryContext(ctx, `
		SELECT LOWER(severity), COUNT(1) FROM findings WHERE status = 'open' GROUP BY LOWER(severity)
	`)
	if err != nil {
		return summary, fmt.Errorf("findings by severity: %w", err)
	}
	for rows.Next() {
		var sev string
		var count int
		if err := rows.Scan(&sev, &count); err != nil {
			rows.Close()
			return summary, err
		}
		summary.OpenFindingsBySeverity[sev] = count
	}
	rows.Close()

	summary.OpenFindingsByCategory = map[string]int{}
	catRows, err := s.db.QueryContext(ctx, `
		SELECT LOWER(category), COUNT(1) FROM findings WHERE status = 'open' GROUP BY LOWER(category)
	`)
	if err != nil {
		return summary, fmt.Errorf("findings by category: %w", err)
	}
	for catRows.Next() {
		var cat string
		var count int
		if err := catRows.Scan(&cat, &count); err != nil {
			catRows.Close()
			return summary, err
		}
		if cat == "" {
			cat = "unknown"
		}
		summary.OpenFindingsByCategory[cat] = count
	}
	catRows.Close()

	recentRows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.repository_id, s.trigger_type, s.ref, s.commit_sha, s.pr_number,
			s.workspace_mode_used, s.commit_pinned, s.status, s.started_at, s.finished_at, s.summary_json, s.error,
			r.full_name
		FROM scans s
		JOIN repositories r ON r.id = s.repository_id
		ORDER BY s.started_at DESC
		LIMIT ?
	`, recentLimit)
	if err != nil {
		return summary, fmt.Errorf("recent scans: %w", err)
	}
	defer recentRows.Close()

	for recentRows.Next() {
		var item ScanWithRepo
		var commitPinned int
		var startedAt string
		var finishedAt sql.NullString
		var summaryJSON string
		if err := recentRows.Scan(
			&item.ID, &item.RepositoryID, &item.TriggerType, &item.Ref, &item.CommitSHA, &item.PRNumber,
			&item.WorkspaceModeUsed, &commitPinned, &item.Status, &startedAt, &finishedAt, &summaryJSON, &item.Error,
			&item.RepoFullName,
		); err != nil {
			return summary, fmt.Errorf("scan recent scan: %w", err)
		}
		item.CommitPinned = intToBool(commitPinned)
		item.StartedAt = parseTime(startedAt)
		item.FinishedAt = parseTimePtr(finishedAt)
		item.SummaryJSON = json.RawMessage(summaryJSON)
		summary.RecentScans = append(summary.RecentScans, item)
	}

	lifecycleRows, err := s.db.QueryContext(ctx, `
		SELECT id, finding_id, scan_id, event_type, message, metadata_json, created_at
		FROM lifecycle_events ORDER BY created_at DESC LIMIT ?
	`, recentLimit)
	if err != nil {
		return summary, fmt.Errorf("recent lifecycle: %w", err)
	}
	defer lifecycleRows.Close()
	summary.RecentLifecycleEvents, err = scanLifecycleRows(lifecycleRows)
	if err != nil {
		return summary, err
	}

	since := time.Now().UTC().Add(-7 * 24 * time.Hour)
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM scans WHERE trigger_type = ? AND started_at >= ?
	`, TriggerScheduled, since.Format(time.RFC3339)).Scan(&summary.ScheduledScansCount); err != nil {
		return summary, fmt.Errorf("count scheduled scans: %w", err)
	}

	var lastScheduled sql.NullString
	if err := s.db.QueryRowContext(ctx, `
		SELECT MAX(started_at) FROM scans WHERE trigger_type = ?
	`, TriggerScheduled).Scan(&lastScheduled); err != nil {
		return summary, fmt.Errorf("last scheduled scan: %w", err)
	}
	if lastScheduled.Valid {
		t := parseTime(lastScheduled.String)
		summary.LastScheduledScanAt = &t
	}

	scheduledRows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.repository_id, s.trigger_type, s.ref, s.commit_sha, s.pr_number,
			s.workspace_mode_used, s.commit_pinned, s.status, s.started_at, s.finished_at, s.summary_json, s.error,
			r.full_name
		FROM scans s
		JOIN repositories r ON r.id = s.repository_id
		WHERE s.trigger_type = ?
		ORDER BY s.started_at DESC
		LIMIT ?
	`, TriggerScheduled, recentLimit)
	if err != nil {
		return summary, fmt.Errorf("recent scheduled scans: %w", err)
	}
	defer scheduledRows.Close()
	for scheduledRows.Next() {
		var item ScanWithRepo
		var commitPinned int
		var startedAt string
		var finishedAt sql.NullString
		var summaryJSON string
		if err := scheduledRows.Scan(
			&item.ID, &item.RepositoryID, &item.TriggerType, &item.Ref, &item.CommitSHA, &item.PRNumber,
			&item.WorkspaceModeUsed, &commitPinned, &item.Status, &startedAt, &finishedAt, &summaryJSON, &item.Error,
			&item.RepoFullName,
		); err != nil {
			return summary, fmt.Errorf("scan scheduled scan row: %w", err)
		}
		item.CommitPinned = intToBool(commitPinned)
		item.StartedAt = parseTime(startedAt)
		item.FinishedAt = parseTimePtr(finishedAt)
		item.SummaryJSON = json.RawMessage(summaryJSON)
		summary.RecentScheduledScans = append(summary.RecentScheduledScans, item)
	}

	summary.RunnerJobsByStatus, err = s.CountRunnerJobsByStatus(ctx)
	if err != nil {
		return summary, fmt.Errorf("runner jobs by status: %w", err)
	}
	summary.Remediation, err = s.RemediationSummary(ctx)
	if err != nil {
		return summary, fmt.Errorf("remediation summary: %w", err)
	}
	summary.Closure, err = s.ClosureSummary(ctx)
	if err != nil {
		return summary, fmt.Errorf("closure summary: %w", err)
	}
	summary.Lifecycle, err = s.LifecycleSummary(ctx)
	if err != nil {
		return summary, fmt.Errorf("lifecycle summary: %w", err)
	}

	if err := s.enrichOperatorDashboard(ctx, &summary); err != nil {
		return summary, err
	}

	return summary, nil
}

func scanScanRows(rows *sql.Rows) ([]Scan, error) {
	var out []Scan
	for rows.Next() {
		var scan Scan
		var commitPinned int
		var startedAt string
		var finishedAt sql.NullString
		var summary string
		if err := rows.Scan(
			&scan.ID, &scan.RepositoryID, &scan.TriggerType, &scan.Ref, &scan.CommitSHA, &scan.PRNumber,
			&scan.WorkspaceModeUsed, &commitPinned, &scan.Status, &startedAt, &finishedAt, &summary, &scan.Error,
		); err != nil {
			return nil, fmt.Errorf("scan scan row: %w", err)
		}
		scan.CommitPinned = intToBool(commitPinned)
		scan.StartedAt = parseTime(startedAt)
		scan.FinishedAt = parseTimePtr(finishedAt)
		scan.SummaryJSON = json.RawMessage(summary)
		out = append(out, scan)
	}
	return out, rows.Err()
}

func scanFindingListItem(row scanner) (FindingListItem, error) {
	var item FindingListItem
	var firstSeen, lastSeen string
	err := row.Scan(
		&item.ID, &item.RepositoryID, &item.Fingerprint, &item.Category, &item.Severity, &item.Confidence,
		&item.Source, &item.RuleID, &item.PackageName, &item.FilePath, &item.Line, &item.Title, &item.Status,
		&item.FirstSeenScanID, &item.LastSeenScanID, &firstSeen, &lastSeen,
		&item.RepoFullName, &item.ExternalIssueNumber, &item.ExternalIssueURL,
	)
	if err != nil {
		return FindingListItem{}, err
	}
	item.FirstSeenAt = parseTime(firstSeen)
	item.LastSeenAt = parseTime(lastSeen)
	return item, nil
}

func scanLifecycleRows(rows *sql.Rows) ([]LifecycleEvent, error) {
	var out []LifecycleEvent
	for rows.Next() {
		var ev LifecycleEvent
		var findingID sql.NullInt64
		var meta, createdAt string
		if err := rows.Scan(&ev.ID, &findingID, &ev.ScanID, &ev.EventType, &ev.Message, &meta, &createdAt); err != nil {
			return nil, fmt.Errorf("scan lifecycle event: %w", err)
		}
		if findingID.Valid {
			id := findingID.Int64
			ev.FindingID = &id
		}
		ev.MetadataJSON = json.RawMessage(meta)
		ev.CreatedAt = parseTime(createdAt)
		out = append(out, ev)
	}
	return out, rows.Err()
}

func scanExternalIssueRows(rows *sql.Rows) ([]ExternalIssue, error) {
	var out []ExternalIssue
	for rows.Next() {
		var issue ExternalIssue
		var createdAt, updatedAt string
		if err := rows.Scan(&issue.ID, &issue.FindingID, &issue.ForgeType, &issue.IssueNumber, &issue.IssueURL, &issue.State, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan external issue: %w", err)
		}
		issue.CreatedAt = parseTime(createdAt)
		issue.UpdatedAt = parseTime(updatedAt)
		out = append(out, issue)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ListScheduledRepositories(ctx context.Context) ([]ScheduledRepository, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.forge_type, r.owner, r.name, r.full_name, r.clone_url, r.default_branch,
			r.connected_repo, r.created_at, r.updated_at, rs.schedule_cron
		FROM repositories r
		INNER JOIN repo_settings rs ON rs.repository_id = r.id
		WHERE r.connected_repo = 1
		AND (rs.enabled IS NULL OR rs.enabled = 1)
		AND rs.schedule_enabled = 1
		AND rs.schedule_cron IS NOT NULL AND TRIM(rs.schedule_cron) != ''
		ORDER BY r.full_name
	`)
	if err != nil {
		return nil, fmt.Errorf("list scheduled repositories: %w", err)
	}
	defer rows.Close()

	var out []ScheduledRepository
	for rows.Next() {
		var item ScheduledRepository
		var connected int
		var createdAt, updatedAt string
		var cron string
		if err := rows.Scan(
			&item.ID, &item.ForgeType, &item.Owner, &item.Name, &item.FullName,
			&item.CloneURL, &item.DefaultBranch, &connected, &createdAt, &updatedAt, &cron,
		); err != nil {
			return nil, fmt.Errorf("scan scheduled repository: %w", err)
		}
		item.ConnectedRepo = intToBool(connected)
		item.CreatedAt = parseTime(createdAt)
		item.UpdatedAt = parseTime(updatedAt)
		item.ScheduleCron = cron
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) HasRunningScanForRepository(ctx context.Context, repositoryID int64) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM scans WHERE repository_id = ? AND status = ?
	`, repositoryID, ScanStatusStarted).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("has running scan: %w", err)
	}
	return count > 0, nil
}

func (s *SQLiteStore) GetLastScanStartedAt(ctx context.Context, repositoryID int64) (*time.Time, error) {
	var startedAt sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT started_at FROM scans
		WHERE repository_id = ?
		ORDER BY started_at DESC
		LIMIT 1
	`, repositoryID).Scan(&startedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("last scan started: %w", err)
	}
	if !startedAt.Valid || strings.TrimSpace(startedAt.String) == "" {
		return nil, nil
	}
	t := parseTime(startedAt.String)
	return &t, nil
}

func (s *SQLiteStore) GetLastScheduledScanFinishedAt(ctx context.Context, repositoryID int64) (*time.Time, error) {
	var finishedAt sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT MAX(finished_at) FROM scans
		WHERE repository_id = ? AND trigger_type = ?
		AND status IN (?, ?, ?)
	`, repositoryID, TriggerScheduled, ScanStatusCompleted, ScanStatusFailed, ScanStatusCancelled).Scan(&finishedAt)
	if err != nil {
		return nil, fmt.Errorf("last scheduled scan finished: %w", err)
	}
	if !finishedAt.Valid {
		return nil, nil
	}
	t := parseTime(finishedAt.String)
	return &t, nil
}

func (s *SQLiteStore) ListRecentScheduledScans(ctx context.Context, limit int) ([]ScanWithRepo, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.repository_id, s.trigger_type, s.ref, s.commit_sha, s.pr_number,
			s.workspace_mode_used, s.commit_pinned, s.status, s.started_at, s.finished_at, s.summary_json, s.error,
			r.full_name
		FROM scans s
		JOIN repositories r ON r.id = s.repository_id
		WHERE s.trigger_type = ?
		ORDER BY s.started_at DESC
		LIMIT ?
	`, TriggerScheduled, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent scheduled scans: %w", err)
	}
	defer rows.Close()

	var out []ScanWithRepo
	for rows.Next() {
		var item ScanWithRepo
		var commitPinned int
		var startedAt string
		var finishedAt sql.NullString
		var summaryJSON string
		if err := rows.Scan(
			&item.ID, &item.RepositoryID, &item.TriggerType, &item.Ref, &item.CommitSHA, &item.PRNumber,
			&item.WorkspaceModeUsed, &commitPinned, &item.Status, &startedAt, &finishedAt, &summaryJSON, &item.Error,
			&item.RepoFullName,
		); err != nil {
			return nil, fmt.Errorf("scan scheduled scan: %w", err)
		}
		item.CommitPinned = intToBool(commitPinned)
		item.StartedAt = parseTime(startedAt)
		item.FinishedAt = parseTimePtr(finishedAt)
		item.SummaryJSON = json.RawMessage(summaryJSON)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) CountScheduledScansSince(ctx context.Context, since time.Time) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM scans WHERE trigger_type = ? AND started_at >= ?
	`, TriggerScheduled, since.UTC().Format(time.RFC3339)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count scheduled scans since: %w", err)
	}
	return count, nil
}

// ReapStaleScans marks long-running "started" scans as failed (e.g. after process restart or context cancel).
func (s *SQLiteStore) ReapStaleScans(ctx context.Context, olderThan time.Duration) (int, error) {
	if olderThan <= 0 {
		olderThan = 2 * time.Hour
	}
	cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		UPDATE scans
		SET status = ?, finished_at = ?, error = ?
		WHERE status = ? AND started_at < ?
	`, ScanStatusFailed, formatTime(time.Now().UTC()),
		"scan did not complete (stale — reaped on startup)", ScanStatusStarted, cutoff)
	if err != nil {
		return 0, fmt.Errorf("reap stale scans: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("reap stale scans rows affected: %w", err)
	}
	return int(n), nil
}
