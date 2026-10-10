package ui

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/preinstall"
	"git.commsnet.org/commstech/repository-detective/store"
)

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"join":             joinStrings,
		"scannerBadge":     scannerStatusBadgeClass,
		"scanBadge":        scanStatusBadgeClass,
		"severityBadge":    severityBadgeClass,
		"categoryBadge":    categoryBadgeClass,
		"statusBadge":      findingStatusBadgeClass,
		"formatTime":       formatTimePtr,
		"formatTimeValue":  formatTimeValue,
		"formatDuration":   formatDurationBetween,
		"jsonPretty":       jsonPretty,
		"shortID":          shortID,
		"mul":              func(a, b float64) float64 { return a * b },
		"profileLabel":     store.ScanProfileLabel,
		"profileDesc":      store.ScanProfileDescription,
		"enforcementLabel": store.EnforcementModeLabel,
		"pct":              func(f float64) int { return int(f*100 + 0.5) },
		"rateClass":        rateMeterClass,
		"rateWidth":        rateMeterWidth,
		"radarBarWidth":    radarBarWidth,
		"navActive":        navActiveClass,
		"apiKeyQS":         apiKeyQueryString,
		"apiKeySuffix":     apiKeyQuerySuffix,
		"issuesFromScan":   issuesFromScanSummary,
		"add":              func(a, b int) int { return a + b },
		"sub":              func(a, b int) int { return a - b },
		"min": func(a, b int) int {
			if a < b {
				return a
			}
			return b
		},
		"gt":                     func(a, b int) bool { return a > b },
		"lt":                     func(a, b int) bool { return a < b },
		"eq":                     func(a, b interface{}) bool { return a == b },
		"dict":                   templateDict,
		"jsonScript":             jsonScriptContent,
		"preinstallRiskDisplay":  preinstallRiskDisplay,
		"preinstallRecDisplay":   preinstallRecDisplay,
		"preinstallFailureStage": preinstallFailureStage,
		"displayBrand":           displayBrandText,
		"displayPath":            displayPath,
	}
}

// jsonScriptContent marks pre-encoded JSON safe for embedding in application/json script tags.
func jsonScriptContent(raw string) template.JS {
	if !json.Valid([]byte(raw)) {
		return template.JS("{}")
	}
	//nosec G203 -- raw is validated JSON; used only inside application/json script tags.
	return template.JS(raw)
}

func templateDict(kv ...any) (map[string]any, error) {
	if len(kv)%2 != 0 {
		return nil, fmt.Errorf("dict: expected key/value pairs")
	}
	out := make(map[string]any, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict: key at %d is not a string", i)
		}
		out[key] = kv[i+1]
	}
	return out, nil
}

func joinStrings(items []string, sep ...string) string {
	if len(items) == 0 {
		return ""
	}
	separator := ", "
	if len(sep) > 0 && sep[0] != "" {
		separator = sep[0]
	}
	out := items[0]
	for i := 1; i < len(items); i++ {
		out += separator + items[i]
	}
	return out
}

func scannerStatusBadgeClass(status string) string {
	switch status {
	case "found", "success", "completed":
		return "found"
	case "running", "started":
		return "running"
	case "timed_out", "failed", "parse_failed", "error":
		return "timed_out"
	default:
		return "binary_missing"
	}
}

func scanStatusBadgeClass(status string) string {
	switch status {
	case "completed":
		return "completed"
	case "analysis_complete":
		return "running"
	case "persistence_incomplete":
		return "failed"
	case "running", "started":
		return "running"
	case "failed":
		return "failed"
	default:
		return "started"
	}
}

func severityBadgeClass(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "medium", "warning", "warn":
		return "medium"
	case "low":
		return "low"
	case "info", "informational", "note":
		return "low"
	default:
		return "medium"
	}
}

func categoryBadgeClass(category string) string {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "security", "misconfiguration", "command_injection", "injection", "sql_injection", "xss":
		return "security"
	case "secret", "hardcoded_secret":
		return "secret"
	case "dependency":
		return "dependency"
	case "reliability":
		return "reliability"
	case "performance":
		return "performance"
	case "test_gap", "test-gap":
		return "test-gap"
	case "tech_debt", "tech-debt":
		return "tech-debt"
	case "maintainability", "code_quality", "code-quality":
		return "code-quality"
	case "architecture":
		return "architecture"
	case "ai_generated_risk", "ai-generated-risk":
		return "ai-risk"
	default:
		return "code-quality"
	}
}

func findingStatusBadgeClass(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "open", "detected":
		return "running"
	case "resolved", "verified", "closed":
		return "completed"
	case "needs_review", "needs-human-review":
		return "timed_out"
	case "suppressed", "false_positive":
		return "skipped"
	default:
		return "started"
	}
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return "—"
	}
	return t.UTC().Format("2006-01-02 15:04:05 UTC")
}

func formatTimeValue(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.UTC().Format("2006-01-02 15:04:05 UTC")
}

func formatDurationBetween(start time.Time, end *time.Time) string {
	if end == nil {
		return "in progress"
	}
	d := end.Sub(start)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
}

func jsonPretty(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(raw)
	}
	return string(b)
}

// GraphFindingView is parsed safe graph detail for dashboard drill-down.
type GraphFindingView struct {
	IsGraph             bool
	RuleID              string
	NodeType            string
	WhyFlagged          string
	Troubleshooting     []string
	SuggestedAction     string
	InboundEdgeCount    int
	OutboundEdgeCount   int
	EntrypointReachable string
	ImportsFrom         []string
	ImportedBy          []string
	PathClassification  string
	GraphNodeID         string
}

func buildGraphFindingView(detail store.FindingDetail) GraphFindingView {
	view := GraphFindingView{IsGraph: detail.Source == "graph", RuleID: detail.RuleID}
	if !view.IsGraph {
		return view
	}
	for _, inst := range detail.Instances {
		if len(inst.RawMetadataJSON) == 0 {
			continue
		}
		var meta map[string]json.RawMessage
		if err := json.Unmarshal(inst.RawMetadataJSON, &meta); err != nil {
			continue
		}
		raw, ok := meta["graph_detail"]
		if !ok || len(raw) == 0 {
			continue
		}
		var gd struct {
			RuleID              string   `json:"rule_id"`
			NodeType            string   `json:"node_type"`
			WhyFlagged          string   `json:"why_flagged"`
			Troubleshooting     []string `json:"troubleshooting"`
			SuggestedAction     string   `json:"suggested_action"`
			InboundEdgeCount    int      `json:"inbound_edge_count"`
			OutboundEdgeCount   int      `json:"outbound_edge_count"`
			EntrypointReachable string   `json:"entrypoint_reachable"`
			ImportsFrom         []string `json:"imports_from"`
			ImportedBy          []string `json:"imported_by"`
			PathClassification  string   `json:"path_classification"`
			GraphNodeID         string   `json:"graph_node_id"`
		}
		if err := json.Unmarshal(raw, &gd); err != nil {
			continue
		}
		view.RuleID = gd.RuleID
		view.NodeType = gd.NodeType
		view.WhyFlagged = gd.WhyFlagged
		view.Troubleshooting = gd.Troubleshooting
		view.SuggestedAction = gd.SuggestedAction
		view.InboundEdgeCount = gd.InboundEdgeCount
		view.OutboundEdgeCount = gd.OutboundEdgeCount
		view.EntrypointReachable = gd.EntrypointReachable
		view.ImportsFrom = gd.ImportsFrom
		view.ImportedBy = gd.ImportedBy
		view.PathClassification = gd.PathClassification
		view.GraphNodeID = gd.GraphNodeID
		break
	}
	if view.WhyFlagged == "" && len(detail.Instances) > 0 {
		view.WhyFlagged = detail.Instances[0].EvidenceRedacted
	}
	return view
}

type scanDetailView struct {
	IssuesFound               int
	FilesAnalyzed             int
	AnalysisTimeMS            int64
	GraphNodes                int
	GraphEdges                int
	ScanProfile               string
	PersistenceStatus         string
	PersistenceExpectedCount  int
	PersistencePersistedCount int
	PersistenceError          string
	IssueSyncStatus           string
	PersistenceIncomplete     bool
}

func sbomStatusFromSummary(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var summary map[string]any
	if err := json.Unmarshal(raw, &summary); err != nil {
		return ""
	}
	return stringFromAny(summary["sbom_status"])
}

func sbomDetailFromSummary(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var summary map[string]any
	if err := json.Unmarshal(raw, &summary); err != nil {
		return ""
	}
	return stringFromAny(summary["sbom_detail"])
}

func buildScanDetailView(raw json.RawMessage) scanDetailView {
	view := scanDetailView{}
	if len(raw) == 0 {
		return view
	}
	var summary map[string]any
	if err := json.Unmarshal(raw, &summary); err != nil {
		return view
	}
	view.IssuesFound = intFromAny(summary["issues_found"])
	view.FilesAnalyzed = intFromAny(summary["files_analyzed"])
	view.AnalysisTimeMS = int64FromAny(summary["analysis_time_ms"])
	view.GraphNodes = intFromAny(summary["graph_nodes"])
	view.GraphEdges = intFromAny(summary["graph_edges"])
	if settings, ok := summary["effective_settings"].(map[string]any); ok {
		view.ScanProfile = stringFromAny(settings["scan_profile"])
	}
	view.PersistenceStatus = stringFromAny(summary["persistence_status"])
	view.PersistenceExpectedCount = intFromAny(summary["persistence_expected_count"])
	view.PersistencePersistedCount = intFromAny(summary["persistence_persisted_count"])
	view.PersistenceError = stringFromAny(summary["persistence_error"])
	view.IssueSyncStatus = stringFromAny(summary["issue_sync_status"])
	if view.PersistenceStatus == store.PersistenceStatusPending ||
		view.PersistenceStatus == store.PersistenceStatusFailed ||
		view.PersistenceStatus == store.PersistenceStatusIncomplete {
		view.PersistenceIncomplete = true
	} else if view.PersistenceExpectedCount > 0 && view.PersistencePersistedCount < view.PersistenceExpectedCount {
		view.PersistenceIncomplete = true
	}
	return view
}

func intFromAny(v any) int {
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

func int64FromAny(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	default:
		return 0
	}
}

func stringFromAny(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func shortID(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// radarBarWidth returns a 0–100 width for category bar charts on the dashboard.
func radarBarWidth(count int, all map[string]int) int {
	if count <= 0 || len(all) == 0 {
		return 0
	}
	max := 0
	for _, v := range all {
		if v > max {
			max = v
		}
	}
	if max == 0 {
		return 0
	}
	w := (count * 100) / max
	if w < 8 && count > 0 {
		return 8
	}
	return w
}

func navActiveClass(section, current string) string {
	if section != "" && section == current {
		return "active"
	}
	return ""
}

func apiKeyQueryString(apiKey string) string {
	return apiKeyQuerySuffix("", apiKey)
}

// apiKeyQuerySuffix appends api_key using ? or & depending on whether path already has a query string.
func apiKeyQuerySuffix(path, apiKey string) string {
	if apiKey == "" {
		return ""
	}
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return sep + "api_key=" + url.QueryEscape(apiKey)
}

func issuesFromScanSummary(raw json.RawMessage) int {
	view := buildScanDetailView(raw)
	return view.IssuesFound
}

func preinstallRiskDisplay(a store.AuditRequest) string {
	return preinstall.RiskScoreDisplay(a)
}

func preinstallRecDisplay(a store.AuditRequest) string {
	return preinstall.RecommendationDisplay(a)
}

func preinstallFailureStage(a store.AuditRequest) string {
	return preinstall.FailureStageFromSummary(a.SummaryJSON)
}
