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
		SELECT SUM(CASE WHEN status = ? OR status = '' THEN 1 ELSE 0 END)
		FROM findings
	`, FindingStatusOpen)
	if err := row.Scan(&summary.OpenFindings); err != nil && err != sql.ErrNoRows {
		return summary, fmt.Errorf("lifecycle open findings: %w", err)
	}

	rem, _ := s.RemediationSummary(ctx)
	summary.RemediationCandidates = rem.Candidates
	summary.ApprovedPlans = rem.ApprovedWaiting

	patchRow := s.db.QueryRowContext(ctx, `
		SELECT
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END)
		FROM patch_attempts
	`, PatchAttemptStatusPROpened, PatchAttemptStatusPRMerged)
	_ = patchRow.Scan(&summary.PROpened, &summary.PRMerged)

	closureRow := s.db.QueryRowContext(ctx, `
		SELECT
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END)
		FROM closure_evidence
	`, ClosureStatusPendingRescan, ClosureStatusBlocked, ClosureStatusStillPresent, ClosureStatusVerified)
	_ = closureRow.Scan(&summary.PendingRescan, &summary.ClosureBlocked, &summary.StillPresent, &summary.ResolvedVerified)

	return summary, nil
}
