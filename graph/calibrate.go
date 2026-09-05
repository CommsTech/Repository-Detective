package graph

import (
	"strings"
)

const (
	severityInfo = "info"
)

// RepoContext carries repository traits used to calibrate graph findings.
type RepoContext struct {
	FileCount        int
	Layout           string
	PrimaryEcosystem string
	HomelabInfra     bool
}

func calibrateGraphFindings(b *builder, findings []GraphFinding, ctx RepoContext) []GraphFinding {
	if len(findings) == 0 {
		return findings
	}
	smallRepo := ctx.FileCount > 0 && ctx.FileCount <= 100
	largeRepo := ctx.FileCount > 500
	out := make([]GraphFinding, 0, len(findings))
	for _, f := range findings {
		cal := calibrateOneFinding(b, f, ctx, smallRepo)
		if cal == nil {
			continue
		}
		if largeRepo && strings.EqualFold(cal.Source, "graph") &&
			(strings.EqualFold(cal.Severity, severityInfo) || strings.EqualFold(cal.Severity, "low")) {
			continue
		}
		out = append(out, *cal)
	}
	return out
}

func calibrateOneFinding(b *builder, f GraphFinding, ctx RepoContext, smallRepo bool) *GraphFinding {
	rule := strings.ToUpper(strings.TrimSpace(f.RuleID))
	info := b.fileInfos[f.File]
	classification := classifyPath(f.File, info)

	switch rule {
	case "GRAPH-ORPHAN-FILE":
		if classification == "test" || classification == "example" || classification == "documentation" {
			return downgradeGraphFinding(f, severityInfo, 0.42,
				"Downgraded: file classified as "+classification+" — orphan graph edges are often expected.",
				"Actionable when the file is production source with no operational references.")
		}
		if b.entrypoints[nodeIDFile(f.File)] {
			return nil
		}
		if ctx.FileCount > 500 {
			return downgradeGraphFinding(f, severityInfo, 0.44,
				"Downgraded: large repository — static orphan-file graph findings are informational; CLI, generated, and script entrypoints are often invisible to import-only analysis.",
				"Actionable when the file is production code with no documented entrypoint and carries security findings.")
		}
		if smallRepo || ctx.HomelabInfra {
			reach := entryReachability(b, nodeIDFile(f.File))
			if reach != entryYes {
				return downgradeGraphFinding(f, severityInfo, 0.48,
					"Downgraded: small/homelab repo — static import graph may miss CLI, compose, and script entrypoints.",
					"Actionable when the file is unreachable from any documented entrypoint and is not intentionally standalone.")
			}
		}
	case "GRAPH-ORPHAN-FUNCTION":
		if classification == "test" || classification == "example" {
			return downgradeGraphFinding(f, severityInfo, 0.42,
				"Downgraded: function in "+classification+" path — orphan call graph edges are often expected.",
				"Actionable when the function is production code on a reachable path.")
		}
		if ctx.FileCount > 500 {
			return downgradeGraphFinding(f, severityInfo, 0.44,
				"Downgraded: large repository — static orphan-function graph findings are informational.",
				"Actionable when the function is security-sensitive and reachable from production entrypoints.")
		}
	case "GRAPH-SUSPICIOUS-ISLAND":
		if smallRepo || ctx.HomelabInfra {
			return downgradeGraphFinding(f, severityInfo, 0.45,
				"Downgraded: isolated cluster in small/homelab repo — overlay findings may reflect script-style layout.",
				"Actionable when disconnected code contains high-severity security findings with clear reachability.")
		}
		if strings.EqualFold(f.Detail.EntrypointReachable, entryNo) && f.Confidence < 0.7 {
			return downgradeGraphFinding(f, severityInfo, 0.5,
				"Downgraded: low-confidence suspicious island without entrypoint reachability evidence.",
				"Actionable when security findings in the cluster are confirmed reachable from production paths.")
		}
	case "GRAPH-DISCONNECTED-PACKAGE":
		if ctx.FileCount > 500 {
			return downgradeGraphFinding(f, severityInfo, 0.46,
				"Downgraded: large repository — disconnected package clusters are often script-style layouts.",
				"Actionable when the package contains confirmed security findings with production reachability.")
		}
		if smallRepo && ctx.PrimaryEcosystem == "python" {
			return downgradeGraphFinding(f, severityInfo, 0.5,
				"Downgraded: Python utility repos often use script-style modules without package import wiring.",
				"Actionable when an entire package is dead code with no README/Makefile/compose references.")
		}
	}
	return &f
}

func downgradeGraphFinding(f GraphFinding, severity string, confidence float64, note, actionable string) *GraphFinding {
	out := f
	out.Severity = severity
	out.Confidence = confidence
	out.Detail.CalibrationNote = note
	out.Detail.ConfidenceBasis = actionable
	out.Detail.SuggestedAction = "Informational — " + note + " Actionable when: " + actionable
	if strings.HasPrefix(out.Title, "Possible ") {
		out.Title = "Informational: " + strings.TrimPrefix(out.Title, "Possible ")
	}
	out.Description = buildDescription(out.Detail)
	return &out
}
