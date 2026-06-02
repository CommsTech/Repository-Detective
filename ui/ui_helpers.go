package ui

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"time"
)

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"join":           joinStrings,
		"scannerBadge":   scannerStatusBadgeClass,
		"scanBadge":      scanStatusBadgeClass,
		"severityBadge":  severityBadgeClass,
		"categoryBadge":  categoryBadgeClass,
		"statusBadge":    findingStatusBadgeClass,
		"formatTime":     formatTimePtr,
		"formatDuration": formatDurationBetween,
		"jsonPretty":     jsonPretty,
		"shortID":        shortID,
		"mul":            func(a, b float64) float64 { return a * b },
		"radarBarWidth":  radarBarWidth,
		"navActive":      navActiveClass,
		"apiKeyQS":       apiKeyQueryString,
		"apiKeySuffix":   apiKeyQuerySuffix,
		"issuesFromScan": issuesFromScanSummary,
		"add":            func(a, b int) int { return a + b },
		"sub":            func(a, b int) int { return a - b },
		"min":            func(a, b int) int { if a < b { return a }; return b },
		"gt":             func(a, b int) bool { return a > b },
		"lt":             func(a, b int) bool { return a < b },
		"eq":             func(a, b interface{}) bool { return a == b },
		"dict":           templateDict,
		"jsonScript":     jsonScriptContent,
	}
}

// jsonScriptContent marks pre-encoded JSON safe for embedding in application/json script tags.
func jsonScriptContent(raw string) template.JS {
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

type scanDetailView struct {
	IssuesFound    int
	FilesAnalyzed  int
	AnalysisTimeMS int64
	GraphNodes     int
	GraphEdges     int
	ScanProfile    string
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
