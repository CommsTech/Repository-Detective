package graph

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// FindingDetail is structured, safe context for dashboard drill-down (no raw secrets).
type FindingDetail struct {
	RuleID              string   `json:"rule_id"`
	NodeType            string   `json:"node_type"`
	FilePath            string   `json:"file_path,omitempty"`
	FunctionName        string   `json:"function_name,omitempty"`
	PackageName         string   `json:"package_name,omitempty"`
	Language            string   `json:"language,omitempty"`
	GraphNodeID         string   `json:"graph_node_id,omitempty"`
	InboundEdgeCount    int      `json:"inbound_edge_count"`
	OutboundEdgeCount   int      `json:"outbound_edge_count"`
	EntrypointReachable string   `json:"entrypoint_reachable"` // yes, no, unknown
	NearestEntrypoints  []string `json:"nearest_entrypoints,omitempty"`
	ImportsFrom         []string `json:"imports_from,omitempty"`
	ImportedBy          []string `json:"imported_by,omitempty"`
	PathClassification  string   `json:"path_classification,omitempty"`
	ExclusionReason     string   `json:"exclusion_reason,omitempty"`
	WhyFlagged          string   `json:"why_flagged"`
	Troubleshooting     []string `json:"troubleshooting"`
	SuggestedAction     string   `json:"suggested_action"`
	ClusterSize         int      `json:"cluster_size,omitempty"`
	FindingsInCluster   int      `json:"findings_in_cluster,omitempty"`
	TestsPresent        string   `json:"tests_present,omitempty"`
	Exported            *bool    `json:"exported,omitempty"`
	CallerCount         int      `json:"caller_count,omitempty"`
	CalibrationNote     string   `json:"calibration_note,omitempty"`
	ConfidenceBasis     string   `json:"confidence_basis,omitempty"`
}

const (
	entryYes    = "yes"
	entryNo     = "no"
	entryUnknown = "unknown"
)

func (d FindingDetail) JSON() string {
	b, err := json.Marshal(d)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func defaultTroubleshooting() []string {
	return []string{
		"Confirm whether this file is an entrypoint, plugin, generated file, migration, CLI, or test fixture.",
		"If intentionally standalone, suppress this fingerprint or rule for this repo.",
		"If unused, remove it or document why it remains.",
		"If it should be connected, add the missing import, registration, or wiring.",
	}
}

func classifyPath(path string, info *parsedFile) string {
	if info != nil && info.isTest {
		return "test"
	}
	lower := strings.ToLower(filepath.ToSlash(path))
	switch {
	case strings.Contains(lower, "/vendor/"):
		return "vendor"
	case strings.Contains(lower, "/examples/"), strings.Contains(lower, "/example/"):
		return "example"
	case strings.Contains(lower, "/docs/"), strings.HasSuffix(lower, ".md"):
		return "documentation"
	case strings.Contains(lower, "_gen.go"), strings.Contains(lower, "/generated/"):
		return "generated"
	default:
		return "source"
	}
}

func (b *builder) countImportEdges(nodeID string) (inbound, outbound int) {
	for _, e := range b.edges {
		if e.Type != "imports" {
			continue
		}
		if e.To == nodeID {
			inbound++
		}
		if e.From == nodeID {
			outbound++
		}
	}
	return inbound, outbound
}

func (b *builder) importTargetsFrom(path string) []string {
	info := b.fileInfos[path]
	if info == nil {
		return nil
	}
	out := make([]string, 0, len(info.imports))
	for _, imp := range info.imports {
		if imp.external {
			out = append(out, imp.target+ " (external)")
			continue
		}
		if t := b.resolveImport(path, imp.target); t != "" {
			out = append(out, t)
		} else if imp.target != "" {
			out = append(out, imp.target)
		}
	}
	if len(out) > 8 {
		return append(out[:8], fmt.Sprintf("…+%d more", len(out)-8))
	}
	return out
}

func (b *builder) importedByFiles(fileID string) []string {
	var out []string
	for _, e := range b.edges {
		if e.Type == "imports" && e.To == fileID {
			from := strings.TrimPrefix(e.From, "file:")
			if from != "" {
				out = append(out, from)
			}
		}
	}
	if len(out) > 8 {
		return append(out[:8], fmt.Sprintf("…+%d more", len(out)-8))
	}
	return out
}

func (b *builder) nearestEntrypoints(fileID string, limit int) []string {
	if b.entrypoints[fileID] {
		return []string{filepath.ToSlash(strings.TrimPrefix(fileID, "file:"))}
	}
	// Reverse BFS on import edges
	prev := map[string]string{}
	queue := []string{fileID}
	seen := map[string]bool{fileID: true}
	var found []string
	for len(queue) > 0 && len(found) < limit {
		cur := queue[0]
		queue = queue[1:]
		if b.entrypoints[cur] && cur != fileID {
			found = append(found, strings.TrimPrefix(cur, "file:"))
			continue
		}
		for _, e := range b.edges {
			if e.Type != "imports" || e.To != cur {
				continue
			}
			if seen[e.From] {
				continue
			}
			seen[e.From] = true
			prev[e.From] = cur
			queue = append(queue, e.From)
		}
	}
	return found
}

func entryReachability(b *builder, fileID string) string {
	if b.entrypoints[fileID] {
		return entryYes
	}
	if len(b.nearestEntrypoints(fileID, 1)) > 0 {
		return entryYes
	}
	if len(b.entrypoints) == 0 {
		return entryUnknown
	}
	return entryNo
}

func formatOrphanFileFinding(b *builder, path string, info *parsedFile) GraphFinding {
	fileID := nodeIDFile(path)
	inbound, outbound := b.countImportEdges(fileID)
	reach := entryReachability(b, fileID)
	why := "This file appears in the repository map but has no inbound import/reference edges from detected entrypoints or other source files. It may be unused, dynamically loaded, generated, or intentionally standalone."
	detail := FindingDetail{
		RuleID:              "GRAPH-ORPHAN-FILE",
		NodeType:            "file",
		FilePath:            path,
		Language:            info.language,
		GraphNodeID:         fileID,
		InboundEdgeCount:    inbound,
		OutboundEdgeCount:   outbound,
		EntrypointReachable: reach,
		NearestEntrypoints:  b.nearestEntrypoints(fileID, 5),
		ImportsFrom:         b.importTargetsFrom(path),
		ImportedBy:          b.importedByFiles(fileID),
		PathClassification: classifyPath(path, info),
		ExclusionReason:     "Not excluded: not marked test/entrypoint/generated/vendor/example; no inbound import edges detected.",
		WhyFlagged:          why,
		Troubleshooting:     defaultTroubleshooting(),
		SuggestedAction:     "Review recommended — confirm whether this file should remain disconnected from the import graph.",
	}
	desc := buildDescription(detail)
	return GraphFinding{
		Category:    "maintainability",
		Source:      "graph",
		RuleID:      "GRAPH-ORPHAN-FILE",
		Severity:    "low",
		Confidence:  0.72,
		Title:       fmt.Sprintf("Possible disconnected file: %s (no inbound imports)", path),
		Description: desc,
		File:        path,
		Line:        1,
		Evidence:    path,
		Detail:      detail,
	}
}

func formatOrphanFunctionFinding(b *builder, path string, info *parsedFile, fn funcRef) GraphFinding {
	fileID := nodeIDFile(path)
	inbound, outbound := b.countImportEdges(fileID)
	exported := fn.exported
	why := "This function is in a file with no inbound import edges; the function itself has no detected callers in the static import/call graph."
	detail := FindingDetail{
		RuleID:              "GRAPH-ORPHAN-FUNCTION",
		NodeType:            "function",
		FilePath:            path,
		FunctionName:        fn.name,
		Language:            info.language,
		GraphNodeID:         nodeIDFunction(path, fn.name),
		InboundEdgeCount:    inbound,
		OutboundEdgeCount:   outbound,
		EntrypointReachable: entryReachability(b, fileID),
		PathClassification:  classifyPath(path, info),
		Exported:            &exported,
		CallerCount:         0,
		ExclusionReason:     "Not excluded: non-exported/non-test function in disconnected file.",
		WhyFlagged:          why,
		Troubleshooting:     defaultTroubleshooting(),
		SuggestedAction:     "Review recommended — confirm whether this function is invoked via reflection, build tags, or external wiring.",
	}
	return GraphFinding{
		Category:    "maintainability",
		Source:      "graph",
		RuleID:      "GRAPH-ORPHAN-FUNCTION",
		Severity:    "low",
		Confidence:  0.65,
		Title:       fmt.Sprintf("Possible unused function %s in %s", fn.name, path),
		Description: buildDescription(detail),
		File:        path,
		Line:        fn.line,
		Evidence:    fn.name,
		Detail:      detail,
	}
}

func formatDisconnectedPackageFinding(b *builder, pkg string, files []string, inboundPkg, outboundPkg int) GraphFinding {
	why := "This package/module appears isolated from detected entrypoints and import paths in the repository map."
	detail := FindingDetail{
		RuleID:              "GRAPH-DISCONNECTED-PACKAGE",
		NodeType:            "package",
		PackageName:         pkg,
		GraphNodeID:         nodeIDPackage(pkg),
		InboundEdgeCount:    inboundPkg,
		OutboundEdgeCount:   outboundPkg,
		EntrypointReachable: entryNo,
		WhyFlagged:          why,
		Troubleshooting:     defaultTroubleshooting(),
		SuggestedAction:     "Review recommended — package may be a library leaf, tooling, or missing wiring.",
	}
	if len(files) > 0 {
		detail.FilePath = files[0]
	}
	desc := buildDescription(detail)
	if len(files) > 0 {
		desc += fmt.Sprintf("\n\nFiles in cluster (%d): %s", len(files), strings.Join(truncateList(files, 6), ", "))
	}
	return GraphFinding{
		Category:    "architecture",
		Source:      "graph",
		RuleID:      "GRAPH-DISCONNECTED-PACKAGE",
		Severity:    "medium",
		Confidence:  0.7,
		Title:       fmt.Sprintf("Possible disconnected package: %s", pkg),
		Description: desc,
		File:        detail.FilePath,
		Line:        1,
		Evidence:    pkg,
		Detail:      detail,
	}
}

func formatSuspiciousIslandFinding(b *builder, path string, info *parsedFile, n Node) GraphFinding {
	fileID := nodeIDFile(path)
	inbound, outbound := b.countImportEdges(fileID)
	tests := "no"
	if strings.HasSuffix(path, "_test.go") || (info != nil && info.isTest) {
		tests = "yes"
	}
	why := "This disconnected file/cluster also has security or quality findings overlay — review recommended."
	detail := FindingDetail{
		RuleID:              "GRAPH-SUSPICIOUS-ISLAND",
		NodeType:            "cluster",
		FilePath:            path,
		GraphNodeID:         fileID,
		InboundEdgeCount:    inbound,
		OutboundEdgeCount:   outbound,
		EntrypointReachable: entryNo,
		FindingsInCluster:   1,
		ClusterSize:         1,
		TestsPresent:        tests,
		WhyFlagged:          why,
		Troubleshooting:     defaultTroubleshooting(),
		SuggestedAction:     "Review recommended — isolated code with findings may need wiring or suppression.",
	}
	if n.Category != "" {
		detail.FindingsInCluster = 1
	}
	return GraphFinding{
		Category:    "architecture",
		Source:      "graph",
		RuleID:      "GRAPH-SUSPICIOUS-ISLAND",
		Severity:    "medium",
		Confidence:  0.68,
		Title:       fmt.Sprintf("Possible suspicious island: %s (%s finding)", path, n.Severity),
		Description: buildDescription(detail),
		File:        path,
		Line:        1,
		Evidence:    n.Category,
		Detail:      detail,
	}
}

func buildDescription(d FindingDetail) string {
	var b strings.Builder
	b.WriteString("Why Repository Detective flagged this:\n")
	b.WriteString(d.WhyFlagged)
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("File: %s\n", d.FilePath))
	if d.FunctionName != "" {
		b.WriteString(fmt.Sprintf("Function: %s\n", d.FunctionName))
	}
	if d.PackageName != "" {
		b.WriteString(fmt.Sprintf("Package: %s\n", d.PackageName))
	}
	b.WriteString(fmt.Sprintf("Rule: %s · Node type: %s\n", d.RuleID, d.NodeType))
	b.WriteString(fmt.Sprintf("Inbound import edges: %d · Outbound import edges: %d\n", d.InboundEdgeCount, d.OutboundEdgeCount))
	b.WriteString(fmt.Sprintf("Entrypoint reachable: %s\n", d.EntrypointReachable))
	if len(d.NearestEntrypoints) > 0 {
		b.WriteString("Nearest entrypoints: " + strings.Join(d.NearestEntrypoints, ", ") + "\n")
	}
	if len(d.ImportsFrom) > 0 {
		b.WriteString("Imports from this file: " + strings.Join(d.ImportsFrom, ", ") + "\n")
	}
	if len(d.ImportedBy) > 0 {
		b.WriteString("Imported by: " + strings.Join(d.ImportedBy, ", ") + "\n")
	}
	if d.PathClassification != "" {
		b.WriteString(fmt.Sprintf("Path classification: %s\n", d.PathClassification))
	}
	if d.ExclusionReason != "" {
		b.WriteString(d.ExclusionReason + "\n")
	}
	b.WriteString("\nTroubleshooting:\n")
	for i, step := range d.Troubleshooting {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
	}
	return strings.TrimSpace(b.String())
}

func truncateList(in []string, max int) []string {
	if len(in) <= max {
		return in
	}
	return append(in[:max], "…")
}
