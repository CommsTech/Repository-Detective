package store

import (
	"context"
	"fmt"
	"time"
)

// SaveSBOMArtifact persists SBOM metadata for a scan.
func (s *SQLiteStore) SaveSBOMArtifact(ctx context.Context, rec SBOMArtifact) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sbom_artifacts (repository_id, scan_id, format, package_count, vuln_count, status, detail, artifact_path, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, rec.RepositoryID, rec.ScanID, rec.Format, rec.PackageCount, rec.VulnCount, rec.Status, rec.Detail, rec.ArtifactPath, now)
	if err != nil {
		return fmt.Errorf("save sbom artifact: %w", err)
	}
	return nil
}

// GetSBOMArtifactForScan returns the latest SBOM artifact for a scan.
func (s *SQLiteStore) GetSBOMArtifactForScan(ctx context.Context, scanID string) (SBOMArtifact, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, repository_id, scan_id, format, package_count, vuln_count, status, detail, artifact_path, created_at
		FROM sbom_artifacts WHERE scan_id = ? ORDER BY id DESC LIMIT 1
	`, scanID)
	var rec SBOMArtifact
	var created string
	if err := row.Scan(&rec.ID, &rec.RepositoryID, &rec.ScanID, &rec.Format, &rec.PackageCount, &rec.VulnCount, &rec.Status, &rec.Detail, &rec.ArtifactPath, &created); err != nil {
		return SBOMArtifact{}, err
	}
	rec.CreatedAt = parseTime(created)
	return rec, nil
}

// GetLatestSBOMArtifactForRepository returns the newest SBOM artifact for a repository.
func (s *SQLiteStore) GetLatestSBOMArtifactForRepository(ctx context.Context, repoID int64) (SBOMArtifact, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, repository_id, scan_id, format, package_count, vuln_count, status, detail, artifact_path, created_at
		FROM sbom_artifacts WHERE repository_id = ? ORDER BY id DESC LIMIT 1
	`, repoID)
	var rec SBOMArtifact
	var created string
	if err := row.Scan(&rec.ID, &rec.RepositoryID, &rec.ScanID, &rec.Format, &rec.PackageCount, &rec.VulnCount, &rec.Status, &rec.Detail, &rec.ArtifactPath, &created); err != nil {
		return SBOMArtifact{}, err
	}
	rec.CreatedAt = parseTime(created)
	return rec, nil
}
