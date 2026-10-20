package graph

import (
	"path/filepath"
	"strings"
)

func analyzeOrphans(b *builder) []GraphFinding {
	var findings []GraphFinding

	importedBy := map[string]int{}
	pkgInbound := map[string]int{}
	pkgOutbound := map[string]int{}
	for _, e := range b.edges {
		if e.Type == "imports" {
			importedBy[e.To]++
			if strings.HasPrefix(e.To, "pkg:") {
				pkgInbound[strings.TrimPrefix(e.To, "pkg:")]++
			}
			if strings.HasPrefix(e.From, "file:") && strings.HasPrefix(e.To, "pkg:") {
				pkgOutbound[strings.TrimPrefix(e.To, "pkg:")]++
			}
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
		if info.packageName != "" {
			pkgID := nodeIDPackage(info.packageName)
			if importedBy[pkgID] > 0 {
				continue
			}
		}
		n := b.nodes[fileID]
		n.Disconnected = true
		b.nodes[fileID] = n
		findings = append(findings, formatOrphanFileFinding(b, path, info))
	}

	if b.cfg.IncludeFunctions {
		for path, info := range b.fileInfos {
			if info.isTest {
				continue
			}
			fileID := nodeIDFile(path)
			if !b.nodes[fileID].Disconnected {
				continue
			}
			for _, fn := range info.functions {
				if fn.exported || fn.name == "init" || fn.name == "main" || fn.name == "TestMain" {
					continue
				}
				if strings.HasPrefix(fn.name, "Test") || strings.HasPrefix(fn.name, "Benchmark") {
					continue
				}
				findings = append(findings, formatOrphanFunctionFinding(b, path, info, fn))
			}
		}
	}

	pkgHasEntry := map[string]bool{}
	pkgFiles := map[string][]string{}
	for path, info := range b.fileInfos {
		if info.packageName != "" {
			pkgFiles[info.packageName] = append(pkgFiles[info.packageName], path)
		}
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
		nonTestFiles := nonTestPackageFiles(pkgFiles[info.packageName])
		if len(nonTestFiles) == 0 {
			continue
		}
		findings = append(findings, formatDisconnectedPackageFinding(
			b, info.packageName, nonTestFiles,
			pkgInbound[info.packageName], pkgOutbound[info.packageName],
		))
	}

	for path, info := range b.fileInfos {
		fileID := nodeIDFile(path)
		n := b.nodes[fileID]
		if !n.Disconnected || info.isTest {
			continue
		}
		if n.Severity == "" {
			continue
		}
		findings = append(findings, formatSuspiciousIslandFinding(b, path, info, n))
	}

	return findings
}

func nonTestPackageFiles(files []string) []string {
	out := make([]string, 0, len(files))
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		out = append(out, path)
	}
	return out
}

func isLikelyGeneratedOrExample(path string) bool {
	lower := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	if strings.Contains(lower, "/vendor/") || strings.Contains(lower, "/examples/") ||
		strings.Contains(lower, "/example/") || strings.HasSuffix(lower, "_gen.go") {
		return true
	}
	if strings.HasPrefix(lower, "ui/templates/") || strings.HasPrefix(lower, "ui/static/") ||
		strings.HasPrefix(lower, "web/static/") || strings.HasPrefix(lower, "docs/") ||
		strings.HasPrefix(lower, "scripts/") {
		return true
	}
	if strings.Contains(lower, "/tests/") || strings.HasSuffix(lower, "conftest.py") {
		return true
	}
	base := filepath.Base(lower)
	if base == "setup.py" || base == "setup.cfg" || base == "pyproject.toml" {
		return true
	}
	if strings.HasSuffix(lower, ".sh") || strings.HasSuffix(lower, ".bash") ||
		strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".md") ||
		strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") {
		return true
	}
	if lower == "deploy.sh" || lower == "docker-compose.yml" || strings.HasPrefix(lower, "docker-compose.") {
		return true
	}
	return false
}
