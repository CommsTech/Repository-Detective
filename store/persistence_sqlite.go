package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// UpdateScanPipelineState merges pipeline fields into scan summary and optionally updates status.
func (s *SQLiteStore) UpdateScanPipelineState(ctx context.Context, scanID string, status string, fields map[string]any) error {
	if scanID == "" {
		return fmt.Errorf("scan id is required")
	}
	scan, err := s.GetScan(ctx, scanID)
	if err != nil {
		return err
	}
	summary, err := MergeSummaryPipelineFields(scan.SummaryJSON, fields)
	if err != nil {
		return fmt.Errorf("merge scan summary: %w", err)
	}
	if status == "" {
		status = scan.Status
	}
	finishedAt := scan.FinishedAt
	if finishedAt == nil {
		now := time.Now().UTC()
		finishedAt = &now
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE scans SET status = ?, finished_at = ?, summary_json = ? WHERE id = ?
	`, status, formatTime(*finishedAt), string(summary), scanID)
	if err != nil {
		return fmt.Errorf("update scan pipeline state: %w", err)
	}
	return nil
}

// GetLatestReconcilableScanForRepository returns the newest scan safe for reconciliation.
func (s *SQLiteStore) GetLatestReconcilableScanForRepository(ctx context.Context, repositoryID int64) (Scan, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, repository_id, trigger_type, status, started_at, finished_at, summary_json
		FROM scans
		WHERE repository_id = ?
		  AND status IN (?, ?, ?)
		ORDER BY started_at DESC
		LIMIT 20
	`, repositoryID, ScanStatusCompleted, ScanStatusAnalysisComplete, ScanStatusPersistenceIncomplete)
	if err != nil {
		return Scan{}, fmt.Errorf("list recent scans: %w", err)
	}
	candidates, err := scanReconcilableCandidatesFromRows(rows)
	if err != nil {
		return Scan{}, err
	}

	for _, scan := range candidates {
		pipeline := PipelineStateFromSummary(scan.SummaryJSON)
		count, err := s.CountFindingInstancesForScan(ctx, scan.ID)
		if err != nil {
			return Scan{}, err
		}
		if !pipeline.IsReconcilable(count) {
			continue
		}
		if scan.Status != ScanStatusCompleted {
			// Upgrade analysis_complete → completed once verified reconcilable.
			_ = s.UpdateScanPipelineState(ctx, scan.ID, ScanStatusCompleted, map[string]any{
				"persistence_status":          PersistenceStatusComplete,
				"persistence_persisted_count": count,
			})
			scan.Status = ScanStatusCompleted
		}
		// Stale scans may leave issue_sync pending after successful persistence + zero new filings.
		if pipeline.PersistenceStatus == PersistenceStatusComplete &&
			pipeline.IssueSyncStatus == IssueSyncStatusPending {
			_ = s.UpdateScanPipelineState(ctx, scan.ID, ScanStatusCompleted, map[string]any{
				"issue_sync_status": IssueSyncStatusComplete,
			})
			if refreshed, err := s.GetScan(ctx, scan.ID); err == nil {
				scan = refreshed
			}
		}
		return scan, nil
	}
	return Scan{}, sql.ErrNoRows
}

func scanReconcilableCandidatesFromRows(rows *sql.Rows) ([]Scan, error) {
	defer rows.Close()
	var candidates []Scan
	for rows.Next() {
		scan, err := scanReconcilableCandidateFromRow(rows)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, scan)
	}
	return candidates, rows.Err()
}

func scanReconcilableCandidateFromRow(rows *sql.Rows) (Scan, error) {
	var scan Scan
	var started string
	var finished sql.NullString
	var summary []byte
	if err := rows.Scan(&scan.ID, &scan.RepositoryID, &scan.TriggerType, &scan.Status, &started, &finished, &summary); err != nil {
		return Scan{}, err
	}
	scan.StartedAt = parseTime(started)
	if finished.Valid {
		t := parseTime(finished.String)
		scan.FinishedAt = &t
	}
	if len(summary) > 0 {
		scan.SummaryJSON = summary
	}
	return scan, nil
}
