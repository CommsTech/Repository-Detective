package store

import (
	"encoding/json"
	"strings"
)

// Graph state labels returned by graph APIs and the Repository Map UI.
const (
	GraphStateAvailable    = "available"
	GraphStateMissing      = "missing"
	GraphStateDisabled     = "disabled"
	GraphStateFailed       = "failed"
	GraphStatePending      = "pending"
	GraphStateTruncated    = "truncated"
	GraphStateScanNotFound = "scan_not_found"
	GraphStateRepoNotFound = "repo_not_found"
)

// GraphStatus is the structured repository map state for a scan or repository.
type GraphStatus struct {
	State         string          `json:"state"`
	ScanID        string          `json:"scan_id,omitempty"`
	RepoID        int64           `json:"repo_id,omitempty"`
	GraphEnabled  bool            `json:"graph_enabled"`
	AnalysisDepth int             `json:"analysis_depth"`
	NodeCount     int             `json:"node_count"`
	EdgeCount     int             `json:"edge_count"`
	Truncated     bool            `json:"truncated"`
	FailureReason string          `json:"failure_reason,omitempty"`
	NextAction    string          `json:"next_action,omitempty"`
	Graph         json.RawMessage `json:"graph"`
}

// GraphStatusInput collects scan, settings, and persistence facts for state resolution.
type GraphStatusInput struct {
	ScanFound     bool
	RepoFound     bool
	ScanID        string
	RepoID        int64
	ScanStatus    string
	GraphEnabled  bool
	AnalysisDepth int
	GraphJSON     []byte
	NodeCount     int
	EdgeCount     int
	Truncated     bool
	GraphError    string
	SummaryJSON   json.RawMessage
}

// ResolveGraphStatus returns a single truthful graph state with optional graph payload.
func ResolveGraphStatus(in GraphStatusInput) GraphStatus {
	out := GraphStatus{
		ScanID:        in.ScanID,
		RepoID:        in.RepoID,
		GraphEnabled:  in.GraphEnabled,
		AnalysisDepth: in.AnalysisDepth,
		NodeCount:     in.NodeCount,
		EdgeCount:     in.EdgeCount,
		Truncated:     in.Truncated,
		Graph:         json.RawMessage("null"),
	}
	if !in.RepoFound {
		out.State = GraphStateRepoNotFound
		out.FailureReason = "repository not found"
		out.NextAction = "Return to the repositories list and pick a connected repository."
		return out
	}
	if !in.ScanFound {
		out.State = GraphStateScanNotFound
		out.FailureReason = "scan not found"
		out.NextAction = "Open scan history and select a completed scan."
		return out
	}
	if isGraphPendingScan(in.ScanStatus) {
		out.State = GraphStatePending
		out.NextAction = "Graph generation runs during analysis. Refresh when the scan completes."
		return out
	}
	if !in.GraphEnabled || in.AnalysisDepth < 2 {
		out.State = GraphStateDisabled
		out.GraphEnabled = in.GraphEnabled
		if in.AnalysisDepth < 2 {
			out.FailureReason = "analysis depth is below 2"
		} else {
			out.FailureReason = "code graph is disabled for this repository or profile"
		}
		out.NextAction = "Enable code graph in repository settings or choose a deeper scan profile, then run a new scan."
		return out
	}
	if graphErr := strings.TrimSpace(in.GraphError); graphErr != "" {
		out.State = GraphStateFailed
		out.FailureReason = graphErr
		out.NextAction = "Review scan logs, adjust graph limits if truncated, and run a new scan."
		return out
	}
	if len(in.GraphJSON) > 0 {
		out.Graph = append(json.RawMessage(nil), in.GraphJSON...)
		if in.Truncated {
			out.State = GraphStateTruncated
			out.NextAction = "Graph rendered with limits applied. Increase graph node/edge limits and rescan for full detail."
		} else {
			out.State = GraphStateAvailable
		}
		return out
	}
	out.State = GraphStateMissing
	out.NextAction = "Run a new scan with code graph enabled and analysis depth ≥ 2."
	return out
}

func isGraphPendingScan(status string) bool {
	switch strings.TrimSpace(status) {
	case ScanStatusStarted, ScanStatusAnalysisComplete:
		return true
	default:
		return false
	}
}

// GraphMetaFromSummary extracts graph fields persisted on scan summary JSON.
func GraphMetaFromSummary(raw json.RawMessage) (enabled bool, depth int, truncated bool, graphErr string) {
	if len(raw) == 0 {
		return false, 0, false, ""
	}
	var summary map[string]any
	if json.Unmarshal(raw, &summary) != nil {
		return false, 0, false, ""
	}
	if b, ok := summary["graph_enabled"].(bool); ok {
		enabled = b
	}
	if v, ok := summary["analysis_depth"].(float64); ok {
		depth = int(v)
	}
	if b, ok := summary["graph_truncated"].(bool); ok && b {
		truncated = true
	}
	if s, ok := summary["graph_error"].(string); ok {
		graphErr = s
	}
	if settings, ok := summary["effective_settings"].(map[string]any); ok {
		if !enabled {
			if b, ok := settings["enable_code_graph"].(bool); ok {
				enabled = b
			}
		}
		if depth == 0 {
			if v, ok := settings["analysis_depth"].(float64); ok {
				depth = int(v)
			}
		}
	}
	return enabled, depth, truncated, graphErr
}
