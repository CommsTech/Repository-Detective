package main

import (
	"context"
	"time"

	"git.commsnet.org/commstech/repository-detective/issues"
	"git.commsnet.org/commstech/repository-detective/store"
)

// issueClaimBridge adapts the SQLiteStore filing-claim API to the forge-agnostic
// issues.FilingClaimStore interface. It must target the concrete SQLiteStore so
// store interface fakes are not affected.
type issueClaimBridge struct {
	db *store.SQLiteStore
}

func (b *issueClaimBridge) AcquireIssueFilingClaim(ctx context.Context, repositoryID int64, forgeType, fingerprint, holder string, lease time.Duration) (issues.FilingClaim, error) {
	c, err := b.db.AcquireIssueFilingClaim(ctx, repositoryID, forgeType, fingerprint, holder, lease)
	if err != nil {
		return issues.FilingClaim{}, err
	}
	return claimToIssues(c), nil
}

func (b *issueClaimBridge) GetIssueFilingClaim(ctx context.Context, repositoryID int64, forgeType, fingerprint string) (issues.FilingClaim, error) {
	c, err := b.db.GetIssueFilingClaim(ctx, repositoryID, forgeType, fingerprint)
	if err != nil {
		return issues.FilingClaim{}, err
	}
	return claimToIssues(c), nil
}

func (b *issueClaimBridge) MarkIssueFilingClaimFiled(ctx context.Context, repositoryID int64, forgeType, fingerprint, holder string, issueNumber int, issueURL string, lease time.Duration) error {
	return b.db.MarkIssueFilingClaimFiled(ctx, repositoryID, forgeType, fingerprint, holder, issueNumber, issueURL, lease)
}

func (b *issueClaimBridge) ReleaseIssueFilingClaim(ctx context.Context, repositoryID int64, forgeType, fingerprint, holder string) (bool, error) {
	return b.db.ReleaseIssueFilingClaim(ctx, repositoryID, forgeType, fingerprint, holder)
}

func claimToIssues(c store.IssueFilingClaim) issues.FilingClaim {
	return issues.FilingClaim{
		Holder:      c.Holder,
		State:       c.State,
		IssueNumber: c.IssueNumber,
		IssueURL:    c.IssueURL,
		ExpiresAt:   c.ExpiresAt,
	}
}

// wireIssueFilingClaims attaches the durable claim store to the issue manager
// when a SQLite-backed store is present. Durable idempotency is the Beta 6 P0
// guarantee that two independent manager processes converge on exactly one
// forge issue per fingerprint.
func wireIssueFilingClaims() {
	if rdStore == nil || issueManager == nil {
		return
	}
	if s, ok := rdStore.(*store.SQLiteStore); ok {
		issueManager.SetFilingClaimStore(&issueClaimBridge{db: s})
	}
}
