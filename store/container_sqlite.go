package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// UpsertContainerImageReference stores or updates a discovered image reference.
func (s *SQLiteStore) UpsertContainerImageReference(ctx context.Context, ref ContainerImageReference) (ContainerImageReference, error) {
	if ref.RepositoryID == 0 || ref.Image == "" {
		return ContainerImageReference{}, fmt.Errorf("repository_id and image required")
	}
	now := time.Now().UTC()
	if ref.CreatedAt.IsZero() {
		ref.CreatedAt = now
	}
	ref.UpdatedAt = now
	meta := ref.MetaJSON
	if meta == nil {
		meta = json.RawMessage(`{}`)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO container_image_references (
			repository_id, image, tag, digest, target_type, file_path, line, service_name,
			mutable_tag, private_registry, last_scan_id, last_digest, created_at, updated_at, meta_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repository_id, image, file_path, line) DO UPDATE SET
			tag=excluded.tag, digest=excluded.digest, target_type=excluded.target_type,
			service_name=excluded.service_name, mutable_tag=excluded.mutable_tag,
			private_registry=excluded.private_registry, updated_at=excluded.updated_at
	`, ref.RepositoryID, ref.Image, ref.Tag, ref.Digest, ref.TargetType, ref.FilePath, ref.Line,
		ref.ServiceName, boolToInt(ref.MutableTag), boolToInt(ref.PrivateRegistry),
		ref.LastScanID, ref.LastDigest, formatTime(ref.CreatedAt), formatTime(ref.UpdatedAt), meta)
	if err != nil {
		return ContainerImageReference{}, fmt.Errorf("upsert container image ref: %w", err)
	}
	return s.GetContainerImageReference(ctx, ref.RepositoryID, ref.Image, ref.FilePath, ref.Line)
}

func (s *SQLiteStore) GetContainerImageReference(ctx context.Context, repoID int64, image, filePath string, line int) (ContainerImageReference, error) {
	var ref ContainerImageReference
	var created, updated string
	var mutable, private int
	err := s.db.QueryRowContext(ctx, `
		SELECT id, repository_id, image, tag, digest, target_type, file_path, line, service_name,
			mutable_tag, private_registry, last_scan_id, last_digest, created_at, updated_at
		FROM container_image_references
		WHERE repository_id=? AND image=? AND file_path=? AND line=?
	`, repoID, image, filePath, line).Scan(
		&ref.ID, &ref.RepositoryID, &ref.Image, &ref.Tag, &ref.Digest, &ref.TargetType,
		&ref.FilePath, &ref.Line, &ref.ServiceName, &mutable, &private,
		&ref.LastScanID, &ref.LastDigest, &created, &updated,
	)
	if err != nil {
		return ContainerImageReference{}, err
	}
	ref.MutableTag = mutable == 1
	ref.PrivateRegistry = private == 1
	ref.CreatedAt = parseTime(created)
	ref.UpdatedAt = parseTime(updated)
	return ref, nil
}

// ListContainerImageReferences lists discovered images for a repository.
func (s *SQLiteStore) ListContainerImageReferences(ctx context.Context, repoID int64) ([]ContainerImageReference, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, repository_id, image, tag, digest, target_type, file_path, line, service_name,
			mutable_tag, private_registry, last_scan_id, last_digest, created_at, updated_at
		FROM container_image_references WHERE repository_id=? ORDER BY image, file_path
	`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ContainerImageReference
	for rows.Next() {
		var ref ContainerImageReference
		var created, updated string
		var mutable, private int
		if err := rows.Scan(
			&ref.ID, &ref.RepositoryID, &ref.Image, &ref.Tag, &ref.Digest, &ref.TargetType,
			&ref.FilePath, &ref.Line, &ref.ServiceName, &mutable, &private,
			&ref.LastScanID, &ref.LastDigest, &created, &updated,
		); err != nil {
			return nil, err
		}
		ref.MutableTag = mutable == 1
		ref.PrivateRegistry = private == 1
		ref.CreatedAt = parseTime(created)
		ref.UpdatedAt = parseTime(updated)
		out = append(out, ref)
	}
	return out, rows.Err()
}

// CreateContainerImageScan inserts a container image scan record.
func (s *SQLiteStore) CreateContainerImageScan(ctx context.Context, scan ContainerImageScan) (ContainerImageScan, error) {
	if scan.RepositoryID == 0 || scan.Image == "" {
		return ContainerImageScan{}, fmt.Errorf("repository_id and image required")
	}
	now := time.Now().UTC()
	if scan.CreatedAt.IsZero() {
		scan.CreatedAt = now
	}
	if scan.Status == "" {
		scan.Status = ContainerScanStatusQueued
	}
	cov := scan.CoverageJSON
	if cov == nil {
		cov = json.RawMessage(`{}`)
	}
	warn := scan.WarningsJSON
	if warn == nil {
		warn = json.RawMessage(`[]`)
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO container_image_scans (
			repository_id, scan_id, runner_job_id, image, image_digest, status, vuln_count,
			sbom_path, sbom_format, coverage_json, warnings_json, started_at, finished_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, scan.RepositoryID, scan.ScanID, scan.RunnerJobID, scan.Image, scan.ImageDigest, scan.Status,
		scan.VulnCount, scan.SBOMPath, scan.SBOMFormat, cov, warn,
		formatTimePtr(scan.StartedAt), formatTimePtrPtr(scan.FinishedAt), formatTime(scan.CreatedAt))
	if err != nil {
		return ContainerImageScan{}, fmt.Errorf("create container image scan: %w", err)
	}
	id, _ := res.LastInsertId()
	scan.ID = id
	return scan, nil
}

// UpdateContainerImageScan updates scan completion fields.
func (s *SQLiteStore) UpdateContainerImageScan(ctx context.Context, id int64, status, digest string, vulnCount int, coverage, warnings json.RawMessage, finished time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE container_image_scans SET status=?, image_digest=?, vuln_count=?, coverage_json=?, warnings_json=?, finished_at=?
		WHERE id=?
	`, status, digest, vulnCount, coverage, warnings, formatTime(finished), id)
	return err
}

// ListContainerImageScans lists image scans for a repository.
func (s *SQLiteStore) ListContainerImageScans(ctx context.Context, repoID int64, limit int) ([]ContainerImageScan, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, repository_id, scan_id, runner_job_id, image, image_digest, status, vuln_count,
			sbom_path, sbom_format, coverage_json, warnings_json, started_at, finished_at, created_at
		FROM container_image_scans WHERE repository_id=? ORDER BY created_at DESC LIMIT ?
	`, repoID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanContainerImageRows(rows)
}

func scanContainerImageRows(rows *sql.Rows) ([]ContainerImageScan, error) {
	var out []ContainerImageScan
	for rows.Next() {
		var sc ContainerImageScan
		var started, finished, created sql.NullString
		if err := rows.Scan(
			&sc.ID, &sc.RepositoryID, &sc.ScanID, &sc.RunnerJobID, &sc.Image, &sc.ImageDigest,
			&sc.Status, &sc.VulnCount, &sc.SBOMPath, &sc.SBOMFormat, &sc.CoverageJSON, &sc.WarningsJSON,
			&started, &finished, &created,
		); err != nil {
			return nil, err
		}
		if started.Valid {
			sc.StartedAt = parseTime(started.String)
		}
		if finished.Valid {
			t := parseTime(finished.String)
			sc.FinishedAt = &t
		}
		if created.Valid {
			sc.CreatedAt = parseTime(created.String)
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

func formatTimePtr(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return formatTime(t)
}

func formatTimePtrPtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return formatTime(*t)
}
