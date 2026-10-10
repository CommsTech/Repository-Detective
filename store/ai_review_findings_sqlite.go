package store

import (
	"context"
	"fmt"
)

// ListFindingsForScan returns findings that have instances in the given scan.
func (s *SQLiteStore) ListFindingsForScan(ctx context.Context, scanID string, limit int) ([]Finding, error) {
	if scanID == "" {
		return nil, fmt.Errorf("scan_id required")
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT f.id, f.repository_id, f.fingerprint, f.category, f.severity, f.confidence,
			f.source, f.rule_id, f.package_name, f.file_path, f.line, f.title, f.status,
			f.first_seen_scan_id, f.last_seen_scan_id, f.first_seen_at, f.last_seen_at
		FROM findings f
		INNER JOIN finding_instances fi ON fi.finding_id = f.id
		WHERE fi.scan_id = ?
		ORDER BY f.severity DESC, f.confidence DESC
		LIMIT ?`, scanID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Finding
	for rows.Next() {
		var f Finding
		var firstAt, lastAt string
		if err := rows.Scan(&f.ID, &f.RepositoryID, &f.Fingerprint, &f.Category, &f.Severity, &f.Confidence,
			&f.Source, &f.RuleID, &f.PackageName, &f.FilePath, &f.Line, &f.Title, &f.Status,
			&f.FirstSeenScanID, &f.LastSeenScanID, &firstAt, &lastAt); err != nil {
			return nil, err
		}
		f.FirstSeenAt = parseTime(firstAt)
		f.LastSeenAt = parseTime(lastAt)
		out = append(out, f)
	}
	return out, rows.Err()
}

// ListFindingInstancesByScan returns instances for a scan keyed by finding_id.
func (s *SQLiteStore) ListFindingInstancesByScan(ctx context.Context, scanID string) (map[int64]FindingInstance, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, finding_id, scan_id, evidence_redacted, location_json, raw_metadata_json, created_at
		FROM finding_instances WHERE scan_id = ?`, scanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int64]FindingInstance)
	for rows.Next() {
		var inst FindingInstance
		var created string
		if err := rows.Scan(&inst.ID, &inst.FindingID, &inst.ScanID, &inst.EvidenceRedacted,
			&inst.LocationJSON, &inst.RawMetadataJSON, &created); err != nil {
			return nil, err
		}
		inst.CreatedAt = parseTime(created)
		out[inst.FindingID] = inst
	}
	return out, rows.Err()
}
