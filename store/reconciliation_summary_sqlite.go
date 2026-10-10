package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ReconciliationSummaryForRepository computes issue/finding reconciliation for the latest reconcilable scan.
func (s *SQLiteStore) ReconciliationSummaryForRepository(ctx context.Context, repositoryID int64, issueFilingEnabled bool) (ReconciliationSummary, error) {
	scan, err := s.GetLatestReconcilableScanForRepository(ctx, repositoryID)
	if err != nil {
		return ReconciliationSummary{}, err
	}
	return s.reconciliationSummary(ctx, repositoryID, scan, issueFilingEnabled)
}

// ReconciliationSummaryForScan computes reconciliation scoped to a specific scan.
func (s *SQLiteStore) ReconciliationSummaryForScan(ctx context.Context, repositoryID int64, scanID string, issueFilingEnabled bool) (ReconciliationSummary, error) {
	scan, err := s.GetScan(ctx, scanID)
	if err != nil {
		return ReconciliationSummary{}, err
	}
	if scan.RepositoryID != repositoryID {
		return ReconciliationSummary{}, fmt.Errorf("scan %s does not belong to repository %d", scanID, repositoryID)
	}
	return s.reconciliationSummary(ctx, repositoryID, scan, issueFilingEnabled)
}

func (s *SQLiteStore) reconciliationSummary(ctx context.Context, repositoryID int64, scan Scan, issueFilingEnabled bool) (ReconciliationSummary, error) {
	out := ReconciliationSummary{
		RepositoryID:       repositoryID,
		LatestScanID:       scan.ID,
		IssueFilingEnabled: issueFilingEnabled,
	}
	if scan.ID != "" {
		out.ScanID = scan.ID
	}

	pipeline := PipelineStateFromSummary(scan.SummaryJSON)
	out.IssueSyncStatus = pipeline.IssueSyncStatus
	out.PersistenceStatus = pipeline.PersistenceStatus
	out.DryRunReportOnly = dryRunFromSummary(scan.SummaryJSON)

	if scan.ID != "" {
		n, _ := s.CountFindingInstancesForScan(ctx, scan.ID)
		out.ScanFindingsTotal = n
		if out.ScanFindingsTotal == 0 && pipeline.IssuesFound > 0 {
			out.ScanFindingsTotal = pipeline.IssuesFound
		}
	}

	if err := s.fillFindingCounts(ctx, repositoryID, scan.ID, &out); err != nil {
		return out, err
	}
	if err := s.fillForgeCounts(ctx, repositoryID, scan.ID, &out); err != nil {
		return out, err
	}

	out.ReportOnlyExplanation = buildReportOnlyExplanation(out, issueFilingEnabled)
	out.SkippedDueReportOnly = skippedDueReportOnly(out, issueFilingEnabled)
	out.SkippedDueBacklogControl = skippedDueBacklogControl(pipeline, issueFilingEnabled)
	out.CountsDifferExpected = !issueFilingEnabled || out.DryRunReportOnly ||
		out.IssueSyncStatus == IssueSyncStatusSkipped ||
		out.FindingsWithoutIssue > 0
	out.MismatchWarning = buildMismatchWarning(out, pipeline)

	lastRecon, _ := s.lastReconciliationRunAt(ctx, repositoryID)
	out.LastReconciliationAt = lastRecon

	return out, nil
}

func dryRunFromSummary(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return false
	}
	v, _ := m["dry_run_report_only"].(bool)
	return v
}

func (s *SQLiteStore) fillFindingCounts(ctx context.Context, repositoryID int64, scanID string, out *ReconciliationSummary) error {
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM findings
		WHERE repository_id = ? AND status = ?`,
		repositoryID, FindingStatusOpen).Scan(&out.OpenFindingsTotal); err != nil {
		return err
	}

	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM findings f
		INNER JOIN external_issues e ON e.finding_id = f.id AND e.state = 'open'
		WHERE f.repository_id = ? AND f.status = ?`,
		repositoryID, FindingStatusResolvedVerified).Scan(&out.ResolvedVerifiedOpen); err != nil {
		return err
	}

	if scanID != "" {
		if err := s.db.QueryRowContext(ctx, `
			SELECT COUNT(DISTINCT f.id) FROM findings f
			INNER JOIN finding_instances fi ON fi.finding_id = f.id AND fi.scan_id = ?
			WHERE f.repository_id = ? AND f.status = ?`,
			scanID, repositoryID, FindingStatusOpen).Scan(&out.ActivePresentOpen); err != nil {
			return err
		}
		if err := s.db.QueryRowContext(ctx, `
			SELECT COUNT(DISTINCT f.id) FROM findings f
			INNER JOIN finding_instances fi ON fi.finding_id = f.id AND fi.scan_id = ?
			WHERE f.repository_id = ? AND f.status = ? AND f.severity IN ('high','critical','medium')`,
			scanID, repositoryID, FindingStatusOpen).Scan(&out.ActionableActiveOpen); err != nil {
			return err
		}
		if err := s.db.QueryRowContext(ctx, `
			SELECT COUNT(DISTINCT f.id) FROM findings f
			INNER JOIN finding_instances fi ON fi.finding_id = f.id AND fi.scan_id = ?
			WHERE f.repository_id = ? AND f.status = ? AND f.severity IN ('low','info')`,
			scanID, repositoryID, FindingStatusOpen).Scan(&out.InformationalActiveOpen); err != nil {
			return err
		}
	}

	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM findings f
		WHERE f.repository_id = ? AND f.status = ?
		AND NOT EXISTS (
			SELECT 1 FROM external_issues e
			WHERE e.finding_id = f.id AND e.state = 'open'
		)`,
		repositoryID, FindingStatusOpen).Scan(&out.ReportOnlyFindings); err != nil {
		return err
	}

	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM findings
		WHERE repository_id = ? AND canonical_finding_id IS NOT NULL`,
		repositoryID).Scan(&out.DuplicateFindings); err != nil {
		return err
	}

	return nil
}

func (s *SQLiteStore) fillForgeCounts(ctx context.Context, repositoryID int64, scanID string, out *ReconciliationSummary) error {
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM external_issues
		WHERE finding_id IN (SELECT id FROM findings WHERE repository_id = ?)
		AND state = 'open'`,
		repositoryID).Scan(&out.ForgeOpenIssues); err != nil {
		return err
	}
	out.MappedOpenIssues = out.ForgeOpenIssues

	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT f.id) FROM findings f
		INNER JOIN external_issues e ON e.finding_id = f.id AND e.state = 'open'
		WHERE f.repository_id = ? AND f.status = ?`,
		repositoryID, FindingStatusOpen).Scan(&out.FindingsWithIssue); err != nil {
		return err
	}
	out.FindingsWithoutIssue = out.OpenFindingsTotal - out.FindingsWithIssue
	if out.FindingsWithoutIssue < 0 {
		out.FindingsWithoutIssue = 0
	}

	if scanID != "" {
		var unmapped int
		_ = s.db.QueryRowContext(ctx, `
			SELECT COUNT(1) FROM external_issues e
			INNER JOIN findings f ON f.id = e.finding_id
			WHERE f.repository_id = ? AND e.state = 'open'
			AND NOT EXISTS (
				SELECT 1 FROM finding_instances fi
				WHERE fi.finding_id = f.id AND fi.scan_id = ?
			)`,
			repositoryID, scanID).Scan(&unmapped)
		out.UnmappedOpenIssues = unmapped
	}

	return nil
}

func (s *SQLiteStore) lastReconciliationRunAt(ctx context.Context, repositoryID int64) (*time.Time, error) {
	var ts sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT MAX(created_at) FROM reconciliation_runs WHERE repository_id = ?`,
		repositoryID).Scan(&ts)
	if err != nil || !ts.Valid {
		return nil, err
	}
	t := ts.Time.UTC()
	return &t, nil
}

func buildReportOnlyExplanation(out ReconciliationSummary, issueFilingEnabled bool) string {
	parts := []string{
		"Scan findings are everything detected in the scan pipeline.",
		"Open forge issues are tracked mappings in Gitea/GitHub (synced by fingerprint when filing is enabled).",
	}
	if out.DryRunReportOnly {
		parts = append(parts, "This scan was report-only — findings were persisted but issue filing was skipped.")
	} else if !issueFilingEnabled {
		parts = append(parts, "Issue filing is disabled — findings persist without creating forge issues.")
	} else if out.IssueSyncStatus == IssueSyncStatusSkipped {
		parts = append(parts, "Issue sync was skipped for this scan (policy or backlog-control).")
	}
	if out.FindingsWithoutIssue > 0 {
		parts = append(parts, fmt.Sprintf("%d open finding(s) have no linked forge issue — this is expected when filing is off or gates block creation.", out.FindingsWithoutIssue))
	}
	return strings.Join(parts, " ")
}

func buildMismatchWarning(out ReconciliationSummary, pipeline ScanPipelineState) string {
	if pipeline.PersistenceStatus != "" && pipeline.PersistenceStatus != PersistenceStatusComplete {
		return "Latest scan persistence is incomplete — counts may change until persistence finishes."
	}
	if out.IssueSyncStatus == IssueSyncStatusPending {
		return "Issue sync is pending — forge issue counts may not reflect the latest scan yet."
	}
	if out.CountsDifferExpected {
		return ""
	}
	if out.OpenFindingsTotal != out.ForgeOpenIssues && out.ForgeOpenIssues > 0 {
		return "Open finding count differs from mapped forge issues — review mappings or run reconciliation."
	}
	return ""
}

func skippedDueReportOnly(out ReconciliationSummary, issueFilingEnabled bool) int {
	if out.DryRunReportOnly || !issueFilingEnabled {
		return out.FindingsWithoutIssue
	}
	return 0
}

func skippedDueBacklogControl(pipeline ScanPipelineState, issueFilingEnabled bool) int {
	if !issueFilingEnabled {
		return 0
	}
	if pipeline.IssueSyncStatus == IssueSyncStatusSkipped {
		return 0
	}
	return 0
}
