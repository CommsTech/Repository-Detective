package main

import (
	"context"
	"fmt"

	"git.commsnet.org/commstech/repository-detective/internal/scanid"
	"git.commsnet.org/commstech/repository-detective/store"
	"git.commsnet.org/commstech/repository-detective/ui"
)

func wireScanTrigger() {
	if operatorUI == nil {
		return
	}
	operatorUI.SetScanTrigger(func(ctx context.Context, req ui.ScanTriggerRequest) (ui.ScanTriggerResult, error) {
		forgeType := normalizeForgeType(req.ForgeType)
		if skip, reason := checkScanAdmission(ctx, forgeType, req.Owner, req.Repository, store.TriggerManual); skip {
			return ui.ScanTriggerResult{}, fmt.Errorf("%s", reason)
		}
		scanID := scanid.New()
		if err := enqueueManualAnalysis(ctx, manualAnalysisRequest{
			ForgeType:        req.ForgeType,
			Owner:            req.Owner,
			Repository:       req.Repository,
			Ref:              req.Ref,
			ScanProfile:      req.ScanProfile,
			ReportOnlyDryRun: req.ReportOnlyDryRun,
			ScanID:           scanID,
		}); err != nil {
			return ui.ScanTriggerResult{}, err
		}
		return ui.ScanTriggerResult{ScanID: scanID}, nil
	})
}
