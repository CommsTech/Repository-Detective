package graph

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

func detectEntrypoints(b *builder, files []FileInput) {
	for _, f := range files {
		info := b.fileInfos[f.Path]
		if info == nil {
			continue
		}
		fileID := nodeIDFile(f.Path)
		if info.isEntry || info.packageName == "main" {
			b.entrypoints[fileID] = true
			n := b.nodes[fileID]
			n.Entrypoint = true
			b.nodes[fileID] = n
		}
		base := strings.ToLower(filepath.Base(f.Path))
		if base == "main.go" || base == "index.js" || base == "index.ts" || base == "main.py" ||
			base == "app.py" || base == "server.js" || base == "cli.go" || base == "manage.py" {
			b.entrypoints[fileID] = true
			n := b.nodes[fileID]
			n.Entrypoint = true
			b.nodes[fileID] = n
		}
		if strings.EqualFold(base, "package.json") {
			detectJSEntryFromPackageJSON(f.Content, b)
		}
	}
}

func detectJSEntryFromPackageJSON(content string, b *builder) {
	var pkg struct {
		Main   string `json:"main"`
		Module string `json:"module"`
		Bin    any    `json:"bin"`
	}
	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return
	}
	for _, entry := range []string{pkg.Main, pkg.Module} {
		if entry == "" {
			continue
		}
		for path := range b.fileInfos {
			if strings.HasSuffix(path, entry) || strings.HasSuffix(path, "/"+entry) {
				id := nodeIDFile(path)
				b.entrypoints[id] = true
				n := b.nodes[id]
				n.Entrypoint = true
				b.nodes[id] = n
			}
		}
	}
}

func overlayFindings(b *builder, findings []FindingOverlay) {
	for _, f := range findings {
		if f.File == "" {
			continue
		}
		fileID := nodeIDFile(f.File)
		if _, ok := b.nodes[fileID]; !ok {
			continue
		}
		fid := nodeIDFinding(f.ID)
		if fid == "finding:" {
			fid = nodeIDFinding(f.RuleID + f.File)
		}
		b.addNode(Node{
			ID: fid, Type: "finding", Label: f.Title,
			Path: f.File, Severity: f.Severity, Category: f.Category,
		})
		b.addEdge(Edge{From: fid, To: fileID, Type: "finding_on"})
		n := b.nodes[fileID]
		if severityRank(f.Severity) > severityRank(n.Severity) {
			n.Severity = f.Severity
			n.Category = f.Category
			b.nodes[fileID] = n
		}
	}
}

func severityRank(s string) int {
	switch strings.ToLower(s) {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	default:
		return 1
	}
}
