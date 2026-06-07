package main

import (
	"context"
	"strings"

	"git.commsnet.org/commstech/bugbot/internal/scanid"
	"git.commsnet.org/commstech/bugbot/ui"
)

func wireScanTrigger() {
	if operatorUI == nil {
		return
	}
	operatorUI.SetScanTrigger(func(ctx context.Context, req ui.ScanTriggerRequest) (ui.ScanTriggerResult, error) {
		scanID := scanid.New()
		enqueueManualAnalysis(ctx, manualAnalysisRequest{
			ForgeType:        req.ForgeType,
			Owner:            req.Owner,
			Repository:       req.Repository,
			Ref:              req.Ref,
			ScanProfile:      req.ScanProfile,
			ReportOnlyDryRun: req.ReportOnlyDryRun,
			ScanID:           scanID,
		})
		return ui.ScanTriggerResult{ScanID: scanID}, nil
	})
}

func manualScanReportOnlyDefault() bool {
	if config == nil {
		return true
	}
	return !config.AutoCreateIssues
}

func normalizeManualScanRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "main"
	}
	return ref
}
