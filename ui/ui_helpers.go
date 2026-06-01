package ui

import (
	"encoding/json"
	"fmt"
	"html/template"
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
	}
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
	case "security", "misconfiguration":
		return "security"
	case "secret":
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
