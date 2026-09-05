package main

import (
	"context"
	"strings"

	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/store"
)

// FIRST_SCAN_PROVEN means a scan initiated through a supported production path
// reached terminal completion and persisted repository identity, trigger,
// required scanner coverage counts, result state, and timestamp.
// It is distinct from WEBHOOK_DELIVERY_E2E_PROVEN (validated webhook accepted).
func maybeRecordFirstScanEvidence(ctx context.Context, scanCtx *store.ScanContext, repositoryID int64, result *analyzers.AnalysisResult, analysisErr error) {
	if rdStore == nil || scanCtx == nil || scanCtx.ScanID == "" || analysisErr != nil || result == nil {
		return
	}
	// Only record once — keep the earliest successful proof.
	if _, ok, _ := rdStore.GetFirstScanEvidence(ctx); ok {
		return
	}
	reqOK, reqTotal := 0, 0
	for _, sr := range result.ScannerResults {
		// Heuristic: count scanners that ran with success or N/A as completed evidence rows.
		st := strings.ToLower(string(sr.Status))
		if st == "" {
			continue
		}
		reqTotal++
		switch st {
		case "success", "ok", "passed", "clean", "not_applicable", "skipped":
			reqOK++
		}
	}
	viaWebhook := scanCtx.TriggerType == store.TriggerPush || scanCtx.TriggerType == store.TriggerPR
	_ = rdStore.RecordFirstScanProven(ctx, store.FirstScanEvidence{
		ScanID:         scanCtx.ScanID,
		RepositoryID:   repositoryID,
		RepositoryName: strings.Trim(scanCtx.Owner+"/"+scanCtx.Repo, "/"),
		TriggerType:    scanCtx.TriggerType,
		Status:         "completed",
		RequiredOK:     reqOK,
		RequiredTotal:  reqTotal,
		ViaWebhook:     viaWebhook,
	})
}
