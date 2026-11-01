package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Claim states for issue_filing_claims.
const (
	IssueFilingClaimStateClaimed = "claimed"
	IssueFilingClaimStateFiled   = "filed"
)

// ErrIssueFilingClaimNotHeld is returned when MarkIssueFilingClaimFiled targets
// a claim row no longer held by the caller.
var ErrIssueFilingClaimNotHeld = errors.New("issue filing claim not held")

// IssueFilingClaim is one durable row per (repository, forge, fingerprint) that
// records which process instance is filing (or just filed) the forge issue. It
// is kept in SQLite so independent manager processes converge on exactly one
// issue and recover when a previous run filed remotely but never linked the
// issue locally.
type IssueFilingClaim struct {
	RepositoryID int64
	ForgeType    string
	Fingerprint  string
	Holder       string
	State        string
	IssueNumber  int
	IssueURL     string
	ClaimedAt    time.Time
	ExpiresAt    time.Time
}

const acquireIssueFilingClaimSQL = `
	INSERT INTO issue_filing_claims
		(repository_id, forge_type, fingerprint, holder, state, issue_number, issue_url, claimed_at, expires_at)
	VALUES (?, ?, ?, ?, 'claimed', 0, '', ?, ?)
	ON CONFLICT(repository_id, forge_type, fingerprint) DO UPDATE SET
		holder = excluded.holder,
		state = 'claimed',
		issue_number = 0,
		issue_url = '',
		claimed_at = excluded.claimed_at,
		expires_at = excluded.expires_at
	WHERE issue_filing_claims.state = 'claimed'
	  AND (issue_filing_claims.expires_at <= ? OR issue_filing_claims.holder = excluded.holder)
`

const markIssueFilingClaimFiledSQL = `
	UPDATE issue_filing_claims
	SET state = 'filed', issue_number = ?, issue_url = ?, expires_at = ?
	WHERE repository_id = ? AND forge_type = ? AND fingerprint = ? AND holder = ?
`

const releaseIssueFilingClaimSQL = `
	DELETE FROM issue_filing_claims
	WHERE repository_id = ? AND forge_type = ? AND fingerprint = ?
	  AND (holder = ? OR state = 'filed')
`

const selectIssueFilingClaimSQL = `
	SELECT repository_id, forge_type, fingerprint, holder, state, issue_number, issue_url, claimed_at, expires_at
	FROM issue_filing_claims
	WHERE repository_id = ? AND forge_type = ? AND fingerprint = ?
`

// AcquireIssueFilingClaim durably claims the filing slot for a fingerprint.
// The return value is the resulting claim row; the claim was acquired iff the
// returned row is held by us. An existing claim is taken over only when it is
// expired (crashed holder) and still unfiled, or when the caller already holds
// it. A filed claim is never overwritten: its issue number is the recovery
// source for linking the remote issue without creating a duplicate.
func (s *SQLiteStore) AcquireIssueFilingClaim(ctx context.Context, repositoryID int64, forgeType, fingerprint, holder string, lease time.Duration) (IssueFilingClaim, error) {
	if repositoryID <= 0 {
		return IssueFilingClaim{}, fmt.Errorf("repository_id is required")
	}
	fingerprint = strings.TrimSpace(fingerprint)
	holder = strings.TrimSpace(holder)
	if fingerprint == "" || holder == "" {
		return IssueFilingClaim{}, fmt.Errorf("fingerprint and holder are required")
	}
	if lease <= 0 {
		lease = 2 * time.Minute
	}
	forgeType = normalizeForgeType(forgeType)
	now := time.Now().UTC()
	args := []any{
		repositoryID, forgeType, fingerprint, holder,
		issueFilingTimestamp(now), issueFilingTimestamp(now.Add(lease)),
		issueFilingTimestamp(now),
	}
	if _, err := s.db.ExecContext(ctx, acquireIssueFilingClaimSQL, args...); err != nil {
		return IssueFilingClaim{}, fmt.Errorf("acquire issue filing claim: %w", err)
	}
	return s.getIssueFilingClaimRow(ctx, repositoryID, forgeType, fingerprint)
}

// GetIssueFilingClaim returns the current claim row for a fingerprint.
// Returns sql.ErrNoRows when no claim exists.
func (s *SQLiteStore) GetIssueFilingClaim(ctx context.Context, repositoryID int64, forgeType, fingerprint string) (IssueFilingClaim, error) {
	if repositoryID <= 0 || strings.TrimSpace(fingerprint) == "" {
		return IssueFilingClaim{}, sql.ErrNoRows
	}
	return s.getIssueFilingClaimRow(ctx, repositoryID, normalizeForgeType(forgeType), strings.TrimSpace(fingerprint))
}

// MarkIssueFilingClaimFiled records the issue that was created under our
// claim; the lease is extended so the filed row (holding the remote issue
// number) stays stable while the local mapping propagates.
func (s *SQLiteStore) MarkIssueFilingClaimFiled(ctx context.Context, repositoryID int64, forgeType, fingerprint, holder string, issueNumber int, issueURL string, lease time.Duration) error {
	if repositoryID <= 0 || issueNumber <= 0 {
		return fmt.Errorf("repository_id and issue_number are required")
	}
	fingerprint = strings.TrimSpace(fingerprint)
	holder = strings.TrimSpace(holder)
	if fingerprint == "" || holder == "" {
		return fmt.Errorf("fingerprint and holder are required")
	}
	if lease <= 0 {
		lease = 2 * time.Minute
	}
	forgeType = normalizeForgeType(forgeType)
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, markIssueFilingClaimFiledSQL,
		issueNumber, issueURL, issueFilingTimestamp(now.Add(lease)),
		repositoryID, forgeType, fingerprint, holder)
	if err != nil {
		return fmt.Errorf("mark issue filing claim filed: %w", err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return ErrIssueFilingClaimNotHeld
	}
	return nil
}

// ReleaseIssueFilingClaim deletes a claim row. The recorded holder may release
// its claim at any time; a filed claim may also be cleared by another holder
// after it linked the mapped issue locally (the mapping now carries the link).
// Reports whether a row was deleted.
func (s *SQLiteStore) ReleaseIssueFilingClaim(ctx context.Context, repositoryID int64, forgeType, fingerprint, holder string) (bool, error) {
	if repositoryID <= 0 || strings.TrimSpace(fingerprint) == "" {
		return false, fmt.Errorf("repository_id and fingerprint are required")
	}
	forgeType = normalizeForgeType(forgeType)
	res, err := s.db.ExecContext(ctx, releaseIssueFilingClaimSQL,
		repositoryID, forgeType, strings.TrimSpace(fingerprint), strings.TrimSpace(holder))
	if err != nil {
		return false, fmt.Errorf("release issue filing claim: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *SQLiteStore) getIssueFilingClaimRow(ctx context.Context, repositoryID int64, forgeType, fingerprint string) (IssueFilingClaim, error) {
	var row IssueFilingClaim
	var claimedAt, expiresAt string
	rowQ := s.db.QueryRowContext(ctx, selectIssueFilingClaimSQL, repositoryID, forgeType, fingerprint)
	err := rowQ.Scan(
		&row.RepositoryID, &row.ForgeType, &row.Fingerprint, &row.Holder, &row.State,
		&row.IssueNumber, &row.IssueURL, &claimedAt, &expiresAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return IssueFilingClaim{}, err
		}
		return IssueFilingClaim{}, fmt.Errorf("get issue filing claim: %w", err)
	}
	row.ClaimedAt = parseIssueFilingTimestamp(claimedAt)
	row.ExpiresAt = parseIssueFilingTimestamp(expiresAt)
	return row, nil
}

// issueFilingTimestamp formats fixed-width UTC so lexicographic string
// comparison in SQL matches chronological order (expiry checks compare strings).
func issueFilingTimestamp(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000000000")
}

func parseIssueFilingTimestamp(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	if t, err := time.Parse("2006-01-02T15:04:05.000000000", value); err == nil {
		return t
	}
	return parseTime(value)
}
