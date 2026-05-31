package graph

import (
	"path/filepath"
	"strings"
)

func analyzeOrphans(b *builder) []GraphFinding {
	var findings []GraphFinding

	importedBy := map[string]int{}
	for _, e := range b.edges {
		if e.Type == "imports" {
			importedBy[e.To]++
		}
	}

	for path, info := range b.fileInfos {
		if info.isTest || info.isEntry {
			continue
		}
		if isLikelyGeneratedOrExample(path) {
			continue
		}
		fileID := nodeIDFile(path)
		if b.entrypoints[fileID] {
			continue
		}
		if importedBy[fileID] > 0 {
			continue
		}
		// Check package-level import via pkg node
		if info.packageName != "" {
			pkgID := nodeIDPackage(info.packageName)
			if importedBy[pkgID] > 0 {
				continue
			}
		}
		n := b.nodes[fileID]
		n.Disconnected = true
		b.nodes[fileID] = n
		findings = append(findings, GraphFinding{
			Category: "maintainability", Source: "graph", RuleID: "GRAPH-ORPHAN-FILE",
			Severity: "low", Confidence: 0.72,
			Title:    "Possible disconnected code path: file not referenced by imports",
			Description: "Review whether this file is intentionally unused or potentially built but not wired into the application.",
			File: path, Line: 1, Evidence: filepath.Base(path),
		})
	}

	if b.cfg.IncludeFunctions {
		for path, info := range b.fileInfos {
			if info.isTest {
				continue
			}
			for _, fn := range info.functions {
				if fn.exported || fn.name == "init" || fn.name == "main" || fn.name == "TestMain" {
					continue
				}
				if strings.HasPrefix(fn.name, "Test") || strings.HasPrefix(fn.name, "Benchmark") {
					continue
				}
				findings = append(findings, GraphFinding{
					Category: "maintainability", Source: "graph", RuleID: "GRAPH-ORPHAN-FUNCTION",
					Severity: "low", Confidence: 0.65,
					Title:    "Possible disconnected code path: function may be unused",
					Description: "Review recommended — function appears defined but not referenced in the import/call graph.",
					File: path, Line: fn.line, Evidence: fn.name,
				})
			}
		}
	}

	// Disconnected packages
	pkgHasEntry := map[string]bool{}
	for _, info := range b.fileInfos {
		if info.isEntry && info.packageName != "" {
			pkgHasEntry[info.packageName] = true
		}
	}
	pkgImported := map[string]bool{}
	for _, e := range b.edges {
		if e.Type == "imports" && strings.HasPrefix(e.To, "pkg:") {
			pkgImported[strings.TrimPrefix(e.To, "pkg:")] = true
		}
	}
	seenPkg := map[string]bool{}
	for _, info := range b.fileInfos {
		if info.packageName == "" || info.isTest || seenPkg[info.packageName] {
			continue
		}
		seenPkg[info.packageName] = true
		if pkgHasEntry[info.packageName] || info.packageName == "main" {
			continue
		}
		if pkgImported[info.packageName] {
			continue
		}
		findings = append(findings, GraphFinding{
			Category: "architecture", Source: "graph", RuleID: "GRAPH-DISCONNECTED-PACKAGE",
			Severity: "medium", Confidence: 0.7,
			Title:    "Potentially disconnected package/module — review recommended",
			Description: "Package appears isolated from entrypoints and import paths in the repository map.",
			File: info.path, Line: 1, Evidence: info.packageName,
		})
	}

	// Suspicious islands: disconnected files with findings and no tests
	for path, info := range b.fileInfos {
		fileID := nodeIDFile(path)
		n := b.nodes[fileID]
		if !n.Disconnected || info.isTest {
			continue
		}
		if n.Severity == "" {
			continue
		}
		findings = append(findings, GraphFinding{
			Category: "architecture", Source: "graph", RuleID: "GRAPH-SUSPICIOUS-ISLAND",
			Severity: "medium", Confidence: 0.68,
			Title:    "Possible suspicious code island — disconnected with findings",
			Description: "Cluster of code appears isolated from main application paths and has findings; review recommended.",
			File: path, Line: 1, Evidence: n.Category,
		})
	}

	return findings
}

func isLikelyGeneratedOrExample(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "/vendor/") || strings.Contains(lower, "/examples/") ||
		strings.Contains(lower, "/example/") || strings.HasSuffix(lower, "_gen.go")
}
