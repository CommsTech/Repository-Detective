package main

import (
	"context"

	"git.commsnet.org/commstech/repository-detective/issuelink"
	"git.commsnet.org/commstech/repository-detective/issues"
)

type issueLinkBridge struct {
	store *issuelink.Store
}

func initIssueLinkBridge() {
	if rdStore == nil {
		return
	}
	bridge := &issueLinkBridge{store: &issuelink.Store{Query: rdStore}}
	if issueManager != nil {
		issueManager.SetIssueMappingLookup(bridge)
		issueManager.SetBackfillRunner(bridge)
	}
}

func (b *issueLinkBridge) MappedIssueNumber(ctx context.Context, repositoryID int64, forgeType, fingerprint string) (int, string, bool) {
	if b == nil || b.store == nil {
		return 0, "", false
	}
	return issuelink.MappedIssue(ctx, b.store, repositoryID, forgeType, fingerprint)
}

func (b *issueLinkBridge) LinkForgeIssue(ctx context.Context, repositoryID int64, forgeType, fingerprint, scanID string, issueNumber int, issueURL string) {
	if b == nil || b.store == nil {
		return
	}
	issuelink.LinkForgeIssue(ctx, b.store, repositoryID, forgeType, fingerprint, scanID, issueNumber, issueURL)
}

func (b *issueLinkBridge) BackfillMissingMappings(ctx context.Context, req *issues.IssueCreationRequest) (issues.BackfillOutcome, error) {
	if b == nil || b.store == nil || req == nil || issueManager == nil {
		return issues.BackfillOutcome{}, nil
	}
	forge := issueManager.ForgeFor(req.ForgeType)
	result, err := issuelink.BackfillExternalIssueMappings(ctx, b.store, forge, req.Owner, req.Repository, req.RepositoryID, req.ForgeType, req.ScanID, logger)
	if err != nil {
		return issues.BackfillOutcome{}, err
	}
	return issues.BackfillOutcome{
		Examined: result.Examined, Backfilled: result.Backfilled, Skipped: result.Skipped,
	}, nil
}
