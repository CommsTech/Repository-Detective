package main

import (
	"context"

	"git.commsnet.org/commstech/repository-detective/internal/scanid"
	"git.commsnet.org/commstech/repository-detective/ui"
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
