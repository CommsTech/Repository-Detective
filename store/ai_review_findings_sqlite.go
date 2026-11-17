package store

import (
	"context"
	"encoding/json"
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
		ORDER BY CASE LOWER(f.severity)
			WHEN 'critical' THEN 5
			WHEN 'crit' THEN 5
			WHEN 'high' THEN 4
			WHEN 'error' THEN 4
			WHEN 'medium' THEN 3
			WHEN 'warning' THEN 3
			WHEN 'warn' THEN 3
			WHEN 'low' THEN 2
			WHEN 'info' THEN 2
			WHEN 'note' THEN 2
			WHEN 'informational' THEN 1
			ELSE 0
		END DESC, f.confidence DESC
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

// ListFindingsForAIReview returns a small pool ordered so security, secret, and
// vulnerability findings come before repeated lint. The scan page keeps
// severity order via ListFindingsForScan; this order exists so a tight LIMIT
// still contains something worth a model call.
func (s *SQLiteStore) ListFindingsForAIReview(ctx context.Context, scanID string, limit int) ([]Finding, error) {
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
		ORDER BY CASE
			WHEN LOWER(f.source) IN ('gitleaks','gosec','semgrep','grype','trivy','govulncheck') THEN 3
			WHEN LOWER(f.category) IN ('secret','security','secrets','dependency','dependency_vulnerability','misconfiguration')
				OR UPPER(f.rule_id) LIKE 'SEC-%' OR UPPER(f.rule_id) LIKE 'CVE-%' OR UPPER(f.rule_id) LIKE 'TRIVY-%' THEN 2
			WHEN UPPER(f.rule_id) LIKE 'LINT-%' OR UPPER(f.rule_id) LIKE 'HEALTH-%'
				OR LOWER(f.category) IN ('maintainability','tech_debt','code_smell')
				OR LOWER(f.source) IN ('ruff','tech_debt','maintainability','graph') THEN 0
			ELSE 1
		END DESC,
		CASE LOWER(f.severity)
			WHEN 'critical' THEN 5
			WHEN 'crit' THEN 5
			WHEN 'high' THEN 4
			WHEN 'error' THEN 4
			WHEN 'medium' THEN 3
			WHEN 'warning' THEN 3
			WHEN 'warn' THEN 3
			WHEN 'low' THEN 2
			WHEN 'info' THEN 2
			WHEN 'note' THEN 2
			WHEN 'informational' THEN 1
			ELSE 0
		END DESC, f.confidence DESC
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
		var rawLocation, rawMeta string
		var created string
		// SQLite drivers return TEXT columns as string; scan into strings and
		// convert so json.RawMessage ([]byte) fields receive byte slices.
		if err := rows.Scan(&inst.ID, &inst.FindingID, &inst.ScanID, &inst.EvidenceRedacted,
			&rawLocation, &rawMeta, &created); err != nil {
			return nil, err
		}
		if rawLocation != "" {
			inst.LocationJSON = json.RawMessage(rawLocation)
		}
		if rawMeta != "" {
			inst.RawMetadataJSON = json.RawMessage(rawMeta)
		}
		inst.CreatedAt = parseTime(created)
		out[inst.FindingID] = inst
	}
	return out, rows.Err()
}
