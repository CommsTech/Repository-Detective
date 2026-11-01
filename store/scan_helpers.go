package store

import (
	"context"
	"fmt"
)

// GetLatestCompletedScanForRepository returns the most recent fully persisted scan.
func (s *SQLiteStore) GetLatestCompletedScanForRepository(ctx context.Context, repositoryID int64) (Scan, error) {
	return s.GetLatestReconcilableScanForRepository(ctx, repositoryID)
}

// ListFingerprintsInScan returns fingerprints seen in a scan for a repository.
func (s *SQLiteStore) ListFingerprintsInScan(ctx context.Context, scanID string, repositoryID int64) (map[string]bool, error) {
	out := map[string]bool{}
	if scanID == "" {
		return out, nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT f.fingerprint
		FROM finding_instances fi
		JOIN findings f ON f.id = fi.finding_id
		WHERE fi.scan_id = ? AND f.repository_id = ?
	`, scanID, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("fingerprints in scan: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var fp string
		if err := rows.Scan(&fp); err != nil {
			return nil, err
		}
		out[fp] = true
	}
	return out, rows.Err()
}
