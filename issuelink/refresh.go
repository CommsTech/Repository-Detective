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

// RefreshResult summarizes inbound forge state sync for tracked mappings.
type RefreshResult struct {
	Examined int
	Updated  int
	Skipped  int
	Errors   []string
}

// RefreshExternalIssueStates pulls live forge issue state for locally-open mappings
// and updates SQLite when Gitea/GitHub already closed (or deleted) the issue.
// Finding status is intentionally left unchanged — that remains reconcile/evidence-owned.
func RefreshExternalIssueStates(ctx context.Context, db *Store, forge issues.IssueForge, owner, repo string, repositoryID int64, forgeType, scanID string, logger *logrus.Logger) (RefreshResult, error) {
	result := RefreshResult{}
	if db == nil || db.Query == nil || forge == nil || repositoryID <= 0 {
		return result, nil
	}
	if logger == nil {
		logger = logrus.New()
	}
	forgeType = normalizeForgeType(forgeType)

	fetcher, ok := forge.(issues.IssueStateFetcher)
	if !ok {
		logger.Debugf("forge adapter does not support GetIssue; skipping state refresh for %s/%s", owner, repo)
		return result, nil
	}

	locals, err := db.Query.ListExternalIssuesByRepository(ctx, repositoryID, store.ListOptions{Limit: 500})
	if err != nil {
		return result, fmt.Errorf("list external issues: %w", err)
	}

	now := time.Now().UTC()
	for _, ei := range locals {
		if !strings.EqualFold(strings.TrimSpace(ei.State), "open") {
			result.Skipped++
			continue
		}
		if ei.IssueNumber <= 0 {
			result.Skipped++
			continue
		}
		result.Examined++

		live, err := fetcher.GetIssue(ctx, owner, repo, ei.IssueNumber)
		if err != nil {
			// Missing issue is treated as closed mapping (deleted / unavailable).
			if isNotFoundErr(err) {
				if syncErr := markExternalIssueClosed(ctx, db, ei, forgeType, scanID, now, "forge issue not found"); syncErr != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("#%d: %v", ei.IssueNumber, syncErr))
					continue
				}
				result.Updated++
				logger.Infof("Marked external issue #%d closed (forge issue missing)", ei.IssueNumber)
				continue
			}
			result.Errors = append(result.Errors, fmt.Sprintf("#%d: get issue: %v", ei.IssueNumber, err))
			continue
		}

		state := strings.ToLower(strings.TrimSpace(live.State))
		if state == "" || state == "open" {
			result.Skipped++
			continue
		}

		url := live.HTMLURL
		if url == "" {
			url = ei.IssueURL
		}
		updated := store.ExternalIssue{
			FindingID:   ei.FindingID,
			ForgeType:   forgeType,
			IssueNumber: ei.IssueNumber,
			IssueURL:    url,
			State:       "closed",
			CreatedAt:   ei.CreatedAt,
			UpdatedAt:   now,
		}
		if _, err := db.Query.UpsertExternalIssue(ctx, updated); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("#%d: upsert: %v", ei.IssueNumber, err))
			continue
		}
		if err := db.Query.AddLifecycleEvent(ctx, store.LifecycleEvent{
			FindingID: findingIDPtr(ei.FindingID),
			ScanID:    scanID,
			EventType: store.LifecycleEventExternalIssueStateSynced,
			Message:   fmt.Sprintf("Synced forge issue #%d state to closed", ei.IssueNumber),
			CreatedAt: now,
		}); err != nil {
			logger.Warnf("lifecycle event for issue #%d: %v", ei.IssueNumber, err)
		}
		result.Updated++
		logger.Infof("Synced external issue #%d state open→closed from forge", ei.IssueNumber)
	}
	return result, nil
}

func markExternalIssueClosed(ctx context.Context, db *Store, ei store.ExternalIssue, forgeType, scanID string, now time.Time, reason string) error {
	updated := store.ExternalIssue{
		FindingID:   ei.FindingID,
		ForgeType:   forgeType,
		IssueNumber: ei.IssueNumber,
		IssueURL:    ei.IssueURL,
		State:       "closed",
		CreatedAt:   ei.CreatedAt,
		UpdatedAt:   now,
	}
	if _, err := db.Query.UpsertExternalIssue(ctx, updated); err != nil {
		return err
	}
	return db.Query.AddLifecycleEvent(ctx, store.LifecycleEvent{
		FindingID: findingIDPtr(ei.FindingID),
		ScanID:    scanID,
		EventType: store.LifecycleEventExternalIssueStateSynced,
		Message:   fmt.Sprintf("Synced forge issue #%d state to closed (%s)", ei.IssueNumber, reason),
		CreatedAt: now,
	})
}

func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status 404") || strings.Contains(msg, "not found")
}
