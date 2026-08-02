package findinglearn

import "strings"

// ReachabilityInput describes graph-informed priority hints.
type ReachabilityInput struct {
	FromEntrypoint bool
	EntrypointRef  string
	TestOnlyPath   bool
	DocsOnlyPath   bool
	VendorPath     bool
	Unknown        bool
}

// ActionabilityAdjust returns severity/confidence deltas from reachability (never hides findings).
func ActionabilityAdjust(severity string, confidence float64, in ReachabilityInput) (string, float64, string) {
	note := ""
	if in.FromEntrypoint && !in.TestOnlyPath {
		if confidence < 0.85 {
			confidence += 0.05
		}
		note = "Reachable from detected entrypoint — raised actionability."
	}
	if in.TestOnlyPath || in.DocsOnlyPath {
		switch severity {
		case "critical", "high":
			// Keep secrets/CVE visible, but stop docs/archive noise from dominating the high queue.
			severity = "medium"
			confidence = min(confidence, 0.7)
		case "medium", "low":
			severity = "info"
			confidence = min(confidence, 0.55)
		}
		note = "Test/docs/archive path — lowered actionability (finding remains visible)."
	}
	if in.VendorPath {
		confidence = min(confidence, 0.5)
		note = "Vendor/generated path — review before issue filing."
	}
	if in.Unknown {
		note = "Reachability not proven — do not treat as safe."
	}
	return severity, confidence, note
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// ClassifyPath heuristics for reachability input.
func ClassifyPath(path string) ReachabilityInput {
	p := strings.ToLower(path)
	in := ReachabilityInput{}
	if strings.Contains(p, "/test") || strings.Contains(p, "_test.") || strings.HasSuffix(p, "_test.go") ||
		strings.Contains(p, "/benchmark/fixture/") || strings.HasSuffix(p, ".go.src") ||
		strings.Contains(p, "/testdata/") || strings.Contains(p, "/fixtures/") {
		in.TestOnlyPath = true
	}
	if strings.Contains(p, "/docs/") || strings.Contains(p, "readme") ||
		strings.HasPrefix(p, "wiki/") || strings.Contains(p, "/wiki/") ||
		strings.Contains(p, "/archive/") || strings.Contains(p, "/session_summaries/") ||
		strings.HasSuffix(p, ".md") || strings.HasSuffix(p, ".mdx") ||
		strings.HasSuffix(p, ".example") || strings.HasSuffix(p, ".sample") ||
		strings.Contains(p, ".example.") || strings.Contains(p, "/examples/") ||
		strings.Contains(p, "/test_generated_apps/") || strings.Contains(p, "/generated_apps/") {
		in.DocsOnlyPath = true
	}
	if strings.HasPrefix(p, "web/static/") || strings.HasSuffix(p, "pdf.js") ||
		strings.Contains(p, "/min.js") || strings.HasSuffix(p, ".min.js") {
		in.VendorPath = true
	}
	if strings.Contains(p, "/vendor/") || strings.Contains(p, "node_modules") ||
		strings.Contains(p, "/ansible_collections/") {
		in.VendorPath = true
	}
	return in
}
