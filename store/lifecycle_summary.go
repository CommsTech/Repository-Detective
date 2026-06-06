package store

import (
	"context"
	"database/sql"
	"fmt"
)

// LifecycleSummary counts findings and remediation/closure pipeline stages.
type LifecycleSummary struct {
	OpenFindings          int
	RemediationCandidates int
	ApprovedPlans         int
	PROpened              int
	PRMerged              int
	PendingRescan         int
	ClosureBlocked        int
	StillPresent          int
	ResolvedVerified      int
}

func (s *SQLiteStore) LifecycleSummary(ctx context.Context) (LifecycleSummary, error) {
	var summary LifecycleSummary
	row := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(CASE WHEN status = ? OR status = '' THEN 1 ELSE 0 END), 0)
		FROM findings
	`, FindingStatusOpen)
	if err := row.Scan(&summary.OpenFindings); err != nil && err != sql.ErrNoRows {
		return summary, fmt.Errorf("lifecycle open findings: %w", err)
	}

	rem, err := s.RemediationSummary(ctx)
	if err != nil {
		return summary, fmt.Errorf("lifecycle remediation summary: %w", err)
	}
	summary.RemediationCandidates = rem.Candidates
	summary.ApprovedPlans = rem.ApprovedWaiting

	patchRow := s.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0)
		FROM patch_attempts
	`, PatchAttemptStatusPROpened, PatchAttemptStatusPRMerged)
	if err := patchRow.Scan(&summary.PROpened, &summary.PRMerged); err != nil && err != sql.ErrNoRows {
		return summary, fmt.Errorf("lifecycle patch attempts: %w", err)
	}

	closureRow := s.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0)
		FROM closure_evidence
	`, ClosureStatusPendingRescan, ClosureStatusBlocked, ClosureStatusStillPresent, ClosureStatusVerified)
	if err := closureRow.Scan(&summary.PendingRescan, &summary.ClosureBlocked, &summary.StillPresent, &summary.ResolvedVerified); err != nil && err != sql.ErrNoRows {
		return summary, fmt.Errorf("lifecycle closure evidence: %w", err)
	}

	return summary, nil
}
