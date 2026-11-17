package store

import (
	"encoding/json"
)

// Scan pipeline persistence / issue-sync states stored in scans.summary_json.
const (
	PersistenceStatusPending    = "pending"
	PersistenceStatusComplete   = "complete"
	PersistenceStatusFailed     = "failed"
	PersistenceStatusIncomplete = "incomplete"

	IssueSyncStatusPending  = "pending"
	IssueSyncStatusComplete = "complete"
	IssueSyncStatusSkipped  = "skipped"
	IssueSyncStatusFailed   = "failed"
)

// ScanPipelineState tracks analysis → persistence → issue sync progress for a scan.
type ScanPipelineState struct {
	PersistenceStatus         string
	IssueSyncStatus           string
	PersistenceExpectedCount  int
	PersistencePersistedCount int
	PersistenceError          string
	IssuesFound               int
}

// PipelineStateFromSummary parses pipeline fields from summary JSON.
func PipelineStateFromSummary(raw json.RawMessage) ScanPipelineState {
	out := ScanPipelineState{}
	if len(raw) == 0 {
		return out
	}
	var summary map[string]any
	if err := json.Unmarshal(raw, &summary); err != nil {
		return out
	}
	out.IssuesFound = intFromSummary(summary["issues_found"])
	out.PersistenceStatus = stringFromSummary(summary["persistence_status"])
	out.IssueSyncStatus = stringFromSummary(summary["issue_sync_status"])
	out.PersistenceExpectedCount = intFromSummary(summary["persistence_expected_count"])
	out.PersistencePersistedCount = intFromSummary(summary["persistence_persisted_count"])
	out.PersistenceError = stringFromSummary(summary["persistence_error"])
	return out
}

// IsPersistenceComplete reports whether finding_instances persistence finished successfully.
func (p ScanPipelineState) IsPersistenceComplete() bool {
	return p.PersistenceStatus == PersistenceStatusComplete
}

// IsReconcilable reports whether reconciliation/issue decisions may use this scan.
func (p ScanPipelineState) IsReconcilable(persistedInstances int) bool {
	switch p.PersistenceStatus {
	case PersistenceStatusComplete:
		if p.PersistenceExpectedCount > 0 {
			return persistedInstances >= p.PersistenceExpectedCount
		}
		return persistedInstances > 0 || p.IssuesFound == 0
	case PersistenceStatusPending, PersistenceStatusFailed, PersistenceStatusIncomplete:
		return false
	default:
		// Legacy scans before pipeline fields: require instance count to match summary.
		if p.IssuesFound == 0 {
			return true
		}
		return persistedInstances >= p.IssuesFound
	}
}

func intFromSummary(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}

func stringFromSummary(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// MergeSummaryPipelineFields returns updated summary JSON with pipeline fields merged in.
func MergeSummaryPipelineFields(raw json.RawMessage, fields map[string]any) (json.RawMessage, error) {
	summary := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &summary)
	}
	for k, v := range fields {
		summary[k] = v
	}
	b, err := json.Marshal(summary)
	if err != nil {
		return nil, err
	}
	return b, nil
}
