package store

import (
	"context"
	"database/sql"
	"fmt"
)

func (s *SQLiteStore) listFleetAuditRows(ctx context.Context) ([]fleetAuditBase, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.full_name, r.forge_type, r.connected_repo,
			ls.started_at, COALESCE(ls.trigger_type, ''), COALESCE(ls.commit_sha, ''),
			COALESCE(ls.dry_run, 0),
			lw.started_at,
			COALESCE(fc.open_findings, 0),
			COALESCE(fc.no_mapped, 0),
			COALESCE(fc.mapped_issues, 0)
		FROM repositories r
		LEFT JOIN (
			SELECT s.repository_id, s.started_at, s.trigger_type, s.commit_sha,
				CASE WHEN json_extract(s.summary_json, '$.dry_run_report_only') = true THEN 1 ELSE 0 END AS dry_run
			FROM scans s
			INNER JOIN (
				SELECT repository_id, MAX(started_at) AS max_started FROM scans GROUP BY repository_id
			) m ON s.repository_id = m.repository_id AND s.started_at = m.max_started
		) ls ON ls.repository_id = r.id
		LEFT JOIN (
			SELECT repository_id, MAX(started_at) AS started_at
			FROM scans WHERE trigger_type = ?
			GROUP BY repository_id
		) lw ON lw.repository_id = r.id
		LEFT JOIN (
			SELECT f.repository_id,
				COUNT(1) AS open_findings,
				SUM(CASE WHEN NOT EXISTS (
					SELECT 1 FROM external_issues e WHERE e.finding_id = f.id AND e.state = 'open'
				) THEN 1 ELSE 0 END) AS no_mapped,
				SUM(CASE WHEN EXISTS (
					SELECT 1 FROM external_issues e WHERE e.finding_id = f.id AND e.state = 'open'
				) THEN 1 ELSE 0 END) AS mapped_issues
			FROM findings f
			WHERE f.status = ?
			GROUP BY f.repository_id
		) fc ON fc.repository_id = r.id
		ORDER BY r.full_name
	`, TriggerPush, FindingStatusOpen)
	if err != nil {
		return nil, fmt.Errorf("fleet audit rows: %w", err)
	}
	defer rows.Close()
	var out []fleetAuditBase
	for rows.Next() {
		var b fleetAuditBase
		var connected int
		var lastScan, lastWebhook sql.NullString
		var dryRun int
		if err := rows.Scan(
			&b.ID, &b.FullName, &b.ForgeType, &connected,
			&lastScan, &b.LastScanTrigger, &b.LastScanCommit, &dryRun,
			&lastWebhook,
			&b.OpenFindings, &b.NoMappedFindings, &b.MappedForgeIssues,
		); err != nil {
			return nil, err
		}
		b.ConnectedRepo = connected == 1
		b.DryRunReportOnly = dryRun == 1
		if lastScan.Valid && lastScan.String != "" {
			t := parseTime(lastScan.String)
			b.LastScanAt = &t
		}
		if lastWebhook.Valid && lastWebhook.String != "" {
			t := parseTime(lastWebhook.String)
			b.LastWebhookAt = &t
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// EnableNightlyScheduleForRepos turns on staggered cron schedules for the given repo IDs.
func (s *SQLiteStore) EnableNightlyScheduleForRepos(ctx context.Context, repoIDs []int64) (int, error) {
	if len(repoIDs) == 0 {
		return 0, nil
	}
	updated := 0
	for _, id := range repoIDs {
		cron := StaggeredNightlyCron(id)
		on := true
		settings, err := s.GetRepoSettings(ctx, id)
		if err != nil {
			return updated, err
		}
		settings.RepositoryID = id
		settings.ScheduleEnabled = &on
		settings.ScheduleCron = &cron
		if err := s.SaveRepoSettings(ctx, settings); err != nil {
			return updated, err
		}
		updated++
	}
	return updated, nil
}

// StaggeredNightlyCron returns a cron expression after calibration learner (02:17).
// Spreads repos between 03:30 and 04:25 UTC in 5-minute steps.
func StaggeredNightlyCron(repositoryID int64) string {
	minute := 30 + int(repositoryID%12)*5
	hour := 3
	if minute >= 60 {
		hour++
		minute -= 60
	}
	return fmt.Sprintf("%d %d * * *", minute, hour)
}
