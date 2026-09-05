package main

import (
	"context"
	"encoding/json"

	"git.commsnet.org/commstech/repository-detective/learning"
	"git.commsnet.org/commstech/repository-detective/store"
)

func learningStore() learning.EventRecorder {
	if rdStore == nil {
		return nil
	}
	return rdStore
}

func emitLearning(ctx context.Context, ev store.LearningEvent) {
	if err := learning.Emit(ctx, learningStore(), ev); err != nil {
		// Best-effort learning stream — never fail the primary request path.
		return
	}
}

func emitLearningEvidence(ctx context.Context, ev store.LearningEvent, evidence any) {
	if err := learning.EmitJSON(ctx, learningStore(), ev, evidence); err != nil {
		return
	}
}

func recordScannerHealthFromScan(ctx context.Context, repositoryID int64, scanID string, results []store.ScanCompletionScanner) {
	if rdStore == nil || repositoryID <= 0 || scanID == "" {
		return
	}
	for _, r := range results {
		_ = rdStore.RecordScannerHealth(ctx, store.ScannerHealthRecord{
			RepositoryID: repositoryID,
			ScanID:       scanID,
			Scanner:      r.Scanner,
			Status:       r.Status,
			FindingCount: r.FindingsCount,
			ErrorClass:   classifyScannerError(r.Status, r.Detail),
		})
		if isScannerFailureStatus(r.Status) {
			emitLearning(ctx, store.LearningEvent{
				RepositoryID:   repositoryID,
				ScanID:         scanID,
				Source:         r.Scanner,
				RuleID:         r.Scanner,
				EventType:      learning.EventScannerFailed,
				CreatedBy:      "scanner",
				IdempotencyKey: scanID + ":scanner_failed:" + r.Scanner,
			})
		}
	}
}

func isScannerFailureStatus(status string) bool {
	switch status {
	case "failed", "error", "timeout", "parse_failed", "timed_out", "binary_missing", "scanner_unavailable":
		return true
	default:
		return false
	}
}

func classifyScannerError(status, detail string) string {
	if detail != "" {
		return status + ":" + detail[:min(len(detail), 80)]
	}
	return status
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func emitReportOnlyDryRun(ctx context.Context, repositoryID int64, scanID string, findings int) {
	emitLearningEvidence(ctx, store.LearningEvent{
		RepositoryID:   repositoryID,
		ScanID:         scanID,
		EventType:      learning.EventReportOnlyDryRun,
		CreatedBy:      "dry-run",
		IdempotencyKey: scanID + ":report_only_dry_run",
	}, map[string]any{"findings": findings})
}

func emitClosureVerified(ctx context.Context, repositoryID int64, scanID string, findingID int64, fp, source, ruleID string) {
	fid := findingID
	emitLearning(ctx, store.LearningEvent{
		RepositoryID:   repositoryID,
		ScanID:         scanID,
		FindingID:      &fid,
		Fingerprint:    fp,
		Source:         source,
		RuleID:         ruleID,
		EventType:      learning.EventResolvedVerified,
		CreatedBy:      "closure",
		IdempotencyKey: scanID + ":resolved:" + fp,
	})
}

func emitFalsePositiveMarked(ctx context.Context, repositoryID int64, findingID int64, fp, source, ruleID, by string) {
	fid := findingID
	emitLearning(ctx, store.LearningEvent{
		RepositoryID:   repositoryID,
		FindingID:      &fid,
		Fingerprint:    fp,
		Source:         source,
		RuleID:         ruleID,
		EventType:      learning.EventUserMarkedFalsePositive,
		CreatedBy:      by,
		IdempotencyKey: "fp:" + fp,
	})
}

func emitRecommendationLearning(ctx context.Context, repositoryID int64, recID int64, accepted bool, source, ruleID string) {
	typ := learning.EventRecommendationRejected
	if accepted {
		typ = learning.EventRecommendationAccepted
	}
	emitLearningEvidence(ctx, store.LearningEvent{
		RepositoryID:   repositoryID,
		Source:         source,
		RuleID:         ruleID,
		EventType:      typ,
		CreatedBy:      "operator",
		IdempotencyKey: typ + ":rec:" + jsonString(recID),
	}, map[string]any{"recommendation_id": recID})
}

func emitDuplicateLinked(ctx context.Context, repositoryID int64, scanID string, findingID int64, fp, source, ruleID string, canonicalIssue int) {
	fid := findingID
	emitLearningEvidence(ctx, store.LearningEvent{
		RepositoryID:   repositoryID,
		ScanID:         scanID,
		FindingID:      &fid,
		Fingerprint:    fp,
		Source:         source,
		RuleID:         ruleID,
		EventType:      learning.EventDuplicateLinked,
		CreatedBy:      "reconcile",
		IdempotencyKey: "dup:" + fp + ":" + jsonString(findingID),
	}, map[string]any{"canonical_issue": canonicalIssue})
}

func jsonString(v int64) string {
	b, _ := json.Marshal(v)
	return string(b)
}
