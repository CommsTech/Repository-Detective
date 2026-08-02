package issuelink

import (
	"context"
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/issues"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/sirupsen/logrus"
)

// Store links findings and external forge issues in SQLite.
type Store struct {
	Query store.QueryStore
}

// BackfillResult summarizes mapping repair for open forge issues.
type BackfillResult struct {
	Examined   int
	Backfilled int
	Skipped    int
	Errors     []string
}

// BackfillExternalIssueMappings links open forge issues with fingerprints to local findings.
func BackfillExternalIssueMappings(ctx context.Context, db *Store, forge issues.IssueForge, owner, repo string, repositoryID int64, forgeType, scanID string, logger *logrus.Logger) (BackfillResult, error) {
	result := BackfillResult{}
	if db == nil || db.Query == nil || forge == nil || repositoryID <= 0 {
		return result, nil
	}
	if logger == nil {
		logger = logrus.New()
	}
	forgeType = normalizeForgeType(forgeType)

	allIssues, err := issues.ListAllOpenIssues(ctx, forge, owner, repo)
	if err != nil {
		return result, fmt.Errorf("list forge issues: %w", err)
	}

	now := time.Now().UTC()
	for _, issue := range allIssues {
		fp := issues.ExtractFingerprintFromBody(issue.Body)
		if fp == "" {
			result.Skipped++
			continue
		}
		result.Examined++

		if _, err := db.Query.GetExternalIssueByIssueNumber(ctx, repositoryID, forgeType, issue.Number); err == nil {
			result.Skipped++
			continue
		}

		finding, err := db.Query.GetFindingByFingerprint(ctx, repositoryID, fp)
		if err != nil {
			meta := issues.ParseIssueBodyMetadata(issue.Body)
			finding, err = db.Query.UpsertFinding(ctx, store.Finding{
				RepositoryID:   repositoryID,
				Fingerprint:    fp,
				Title:          firstNonEmpty(meta.Title, fmt.Sprintf("Backfilled finding %s", fp)),
				Severity:       firstNonEmpty(meta.Severity, "medium"),
				Category:       firstNonEmpty(meta.Category, "reliability"),
				Source:         meta.Source,
				RuleID:         meta.RuleID,
				Status:         store.FindingStatusOpen,
				FirstSeenAt:    now,
				LastSeenAt:     now,
				LastSeenScanID: scanID,
			})
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("#%d: upsert finding: %v", issue.Number, err))
				continue
			}
		}

		if _, err := db.Query.UpsertExternalIssue(ctx, store.ExternalIssue{
			FindingID: finding.ID, ForgeType: forgeType,
			IssueNumber: issue.Number, IssueURL: issue.HTMLURL, State: "open",
		}); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("#%d: upsert external issue: %v", issue.Number, err))
			continue
		}

		if err := db.Query.AddLifecycleEvent(ctx, store.LifecycleEvent{
			FindingID: findingIDPtr(finding.ID), ScanID: scanID,
			EventType: store.LifecycleEventExternalIssueMappingBackfilled,
			Message:   fmt.Sprintf("Backfilled mapping to %s issue #%d", forgeType, issue.Number),
			CreatedAt: now,
		}); err != nil {
			logger.Warnf("backfill lifecycle event for #%d: %v", issue.Number, err)
		}
		result.Backfilled++
		logger.Infof("Backfilled external issue mapping: fingerprint=%s issue=#%d", fp, issue.Number)
	}
	return result, nil
}

// LinkForgeIssue records a forge issue mapping for a fingerprint if missing.
func LinkForgeIssue(ctx context.Context, db *Store, repositoryID int64, forgeType, fingerprint, scanID string, issueNumber int, issueURL string) {
	if db == nil || db.Query == nil || repositoryID <= 0 || fingerprint == "" || issueNumber <= 0 {
		return
	}
	forgeType = normalizeForgeType(forgeType)
	if _, err := db.Query.GetExternalIssueByFingerprint(ctx, repositoryID, forgeType, fingerprint); err == nil {
		return
	}
	finding, err := db.Query.GetFindingByFingerprint(ctx, repositoryID, fingerprint)
	if err != nil {
		return
	}
	if _, err := db.Query.UpsertExternalIssue(ctx, store.ExternalIssue{
		FindingID: finding.ID, ForgeType: forgeType,
		IssueNumber: issueNumber, IssueURL: issueURL, State: "open",
	}); err != nil {
		return
	}
	if err := db.Query.AddLifecycleEvent(ctx, store.LifecycleEvent{
		FindingID: findingIDPtr(finding.ID), ScanID: scanID,
		EventType: store.LifecycleEventExternalIssueMappingBackfilled,
		Message:   fmt.Sprintf("Linked forge issue #%d from live fingerprint search", issueNumber),
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return
	}
}

// MappedIssue returns a local mapping for a fingerprint if present.
func MappedIssue(ctx context.Context, db *Store, repositoryID int64, forgeType, fingerprint string) (issueNumber int, issueURL string, ok bool) {
	if db == nil || db.Query == nil {
		return 0, "", false
	}
	ext, err := db.Query.GetExternalIssueByFingerprint(ctx, repositoryID, normalizeForgeType(forgeType), fingerprint)
	if err != nil || ext.IssueNumber <= 0 {
		return 0, "", false
	}
	return ext.IssueNumber, ext.IssueURL, true
}

func normalizeForgeType(forgeType string) string {
	if strings.EqualFold(strings.TrimSpace(forgeType), store.ForgeTypeGitHub) {
		return store.ForgeTypeGitHub
	}
	return store.ForgeTypeGitea
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func findingIDPtr(id int64) *int64 {
	if id <= 0 {
		return nil
	}
	return &id
}
