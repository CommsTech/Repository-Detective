package ui

import (
	"context"
	"fmt"
)

// ScanTriggerRequest starts a manual repository scan from the UI.
type ScanTriggerRequest struct {
	ForgeType        string
	Owner            string
	Repository       string
	Ref              string
	ScanProfile      string
	ReportOnlyDryRun bool
}

// ScanTriggerResult is returned when a manual scan is queued.
type ScanTriggerResult struct {
	ScanID string
}

// ScanTrigger queues a manual scan (implemented in main).
type ScanTrigger func(ctx context.Context, req ScanTriggerRequest) (ScanTriggerResult, error)

// SetScanTrigger wires manual scan from the UI.
func (h *Handler) SetScanTrigger(fn ScanTrigger) {
	if h != nil {
		h.scanTrigger = fn
	}
}

// ScanTriggerEnabled reports whether manual scans can be started.
func (h *Handler) ScanTriggerEnabled() bool {
	return h != nil && h.scanTrigger != nil
}

// triggerManualScan queues a scan for a repository.
func (h *Handler) triggerManualScan(ctx context.Context, req ScanTriggerRequest) (ScanTriggerResult, error) {
	if h.scanTrigger == nil {
		return ScanTriggerResult{}, fmt.Errorf("manual scan is not configured")
	}
	return h.scanTrigger(ctx, req)
}
