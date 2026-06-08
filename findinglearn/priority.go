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
		if severity == "medium" || severity == "low" {
			severity = "info"
			confidence = min(confidence, 0.55)
		}
		note = "Test/docs-only path — lowered actionability (finding remains visible)."
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
		strings.Contains(p, "/benchmark/fixture/") || strings.HasSuffix(p, ".go.src") {
		in.TestOnlyPath = true
	}
	if strings.Contains(p, "/docs/") || strings.Contains(p, "readme") {
		in.DocsOnlyPath = true
	}
	if strings.Contains(p, "/vendor/") || strings.Contains(p, "node_modules") {
		in.VendorPath = true
	}
	return in
}
