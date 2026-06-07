package graph

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type builder struct {
	cfg          Config
	skipPatterns []string
	nodes        map[string]Node
	edges        map[string]Edge
	fileInfos    map[string]*parsedFile
	entrypoints  map[string]bool
}

type parsedFile struct {
	path       string
	language   string
	packageName string
	imports    []importRef
	functions  []funcRef
	isTest     bool
	isEntry    bool
}

type importRef struct {
	target string
	external bool
}

type funcRef struct {
	name     string
	exported bool
	line     int
}

// Build generates a repository graph from workspace files.
func Build(ctx context.Context, input BuildInput, cfg Config, skipPatterns []string) (Graph, []GraphFinding) {
	cfg = cfg.normalized()
	if !cfg.Enabled {
		return Graph{}, nil
	}

	deadline := time.Now().Add(time.Duration(cfg.TimeoutSeconds) * time.Second)
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	files := filterFiles(input.Files, skipPatterns)
	b := &builder{
		cfg:          cfg,
		skipPatterns: skipPatterns,
		nodes:        make(map[string]Node),
		edges:        make(map[string]Edge),
		fileInfos:    make(map[string]*parsedFile),
		entrypoints:  make(map[string]bool),
	}

	repoNodeID := "repo:root"
	b.addNode(Node{ID: repoNodeID, Type: "repo", Label: "repository"})

	// Structural tree + parse files
	for _, f := range files {
		select {
		case <-ctx.Done():
			return b.finalize(input, true), nil
		default:
		}
		b.ensureDirectoryChain(repoNodeID, f.Path)
		b.parseAndIndex(f)
	}

	// Import/dependency edges
	for path, info := range b.fileInfos {
		fileID := nodeIDFile(path)
		for _, imp := range info.imports {
			if imp.external {
				extID := nodeIDExternal(imp.target)
				b.addNode(Node{ID: extID, Type: "external_dependency", Label: imp.target})
				b.addEdge(Edge{ID: edgeID(fileID, extID, "depends_on"), From: fileID, To: extID, Type: "depends_on"})
				continue
			}
			targetFile := b.resolveImport(path, imp.target)
			if targetFile != "" {
				b.addEdge(Edge{ID: edgeID(fileID, nodeIDFile(targetFile), "imports"), From: fileID, To: nodeIDFile(targetFile), Type: "imports", Weight: 1})
			}
			if pkg := imp.target; pkg != "" {
				pkgID := nodeIDPackage(pkg)
				b.addNode(Node{ID: pkgID, Type: "package", Label: pkg, PackageName: pkg})
				b.addEdge(Edge{ID: edgeID(fileID, pkgID, "imports"), From: fileID, To: pkgID, Type: "imports"})
			}
		}
		if info.packageName != "" {
			pkgID := nodeIDPackage(info.packageName)
			b.addNode(Node{ID: pkgID, Type: "package", Label: info.packageName, PackageName: info.packageName, Language: info.language})
			b.addEdge(Edge{ID: edgeID(fileID, pkgID, "defines"), From: fileID, To: pkgID, Type: "defines"})
		}
		if cfg.IncludeFunctions {
			for _, fn := range info.functions {
				fnID := nodeIDFunction(path, fn.name)
				b.addNode(Node{ID: fnID, Type: "function", Label: fn.name, Path: path, Language: info.language})
				b.addEdge(Edge{ID: edgeID(fileID, fnID, "defines"), From: fileID, To: fnID, Type: "defines"})
			}
		}
	}

	// Entrypoints
	detectEntrypoints(b, files)
	detectOperationalEntrypoints(b, files)

	// Finding overlay
	if cfg.IncludeFindings {
		overlayFindings(b, input.Findings)
	}

	// Orphan / disconnected analysis
	graphFindings := calibrateGraphFindings(b, analyzeOrphans(b), input.Repo)

	g := b.finalize(input, false)
	return g, graphFindings
}

func (b *builder) parseAndIndex(f FileInput) {
	lang := f.Language
	if lang == "" {
		lang = detectLanguage(f.Path)
	}
	info := &parsedFile{path: f.Path, language: lang, isTest: isTestFile(f.Path)}
	switch lang {
	case "go":
		parseGoFile(f.Content, f.Path, info)
	case "javascript", "typescript":
		parseJSFile(f.Content, info)
	case "python":
		parsePythonFile(f.Content, info)
	default:
		parseGenericFile(f.Content, f.Path, info)
	}
	b.fileInfos[f.Path] = info
	fileID := nodeIDFile(f.Path)
	b.addNode(Node{
		ID: fileID, Type: "file", Label: filepath.Base(f.Path),
		Path: f.Path, Language: lang, PackageName: info.packageName,
	})
}

func (b *builder) ensureDirectoryChain(repoID, filePath string) {
	filePath = filepath.ToSlash(filePath)
	dir := filepath.ToSlash(filepath.Dir(filePath))
	if dir == "." || dir == "" {
		return
	}
	parts := strings.Split(dir, "/")
	parent := repoID
	accum := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		if accum == "" {
			accum = part
		} else {
			accum += "/" + part
		}
		dirID := nodeIDDirectory(accum)
		b.addNode(Node{ID: dirID, Type: "directory", Label: part, Path: accum})
		b.addEdge(Edge{ID: edgeID(parent, dirID, "contains"), From: parent, To: dirID, Type: "contains"})
		parent = dirID
	}
	fileID := nodeIDFile(filePath)
	b.addEdge(Edge{ID: edgeID(parent, fileID, "contains"), From: parent, To: fileID, Type: "contains"})
}

func (b *builder) resolveImport(fromFile, target string) string {
	target = strings.Trim(target, "./")
	if strings.HasSuffix(target, ".go") || strings.HasSuffix(target, ".js") || strings.HasSuffix(target, ".ts") || strings.HasSuffix(target, ".py") {
		if _, ok := b.fileInfos[target]; ok {
			return target
		}
	}
	dir := filepath.ToSlash(filepath.Dir(fromFile))
	candidate := filepath.ToSlash(filepath.Join(dir, target))
	if _, ok := b.fileInfos[candidate]; ok {
		return candidate
	}
	for path := range b.fileInfos {
		if strings.HasSuffix(path, "/"+target+".go") || strings.HasSuffix(path, "/"+target+".js") ||
			strings.HasSuffix(path, "/"+target+".py") || strings.HasSuffix(path, "/"+target+"/__init__.py") {
			return path
		}
	}
	// Python module paths: utils.helper -> utils.py or utils/helper.py
	if strings.Contains(target, ".") {
		parts := strings.Split(target, ".")
		modPath := strings.Join(parts, "/") + ".py"
		if _, ok := b.fileInfos[modPath]; ok {
			return modPath
		}
		if _, ok := b.fileInfos[parts[0]+".py"]; ok {
			return parts[0] + ".py"
		}
	}
	if !strings.Contains(target, "/") && !strings.Contains(target, ".") {
		candidatePy := target + ".py"
		if _, ok := b.fileInfos[candidatePy]; ok {
			return candidatePy
		}
	}
	return ""
}

func (b *builder) addNode(n Node) {
	if existing, ok := b.nodes[n.ID]; ok {
		if n.Entrypoint {
			existing.Entrypoint = true
			b.nodes[n.ID] = existing
		}
		if n.Disconnected {
			existing.Disconnected = true
			b.nodes[n.ID] = existing
		}
		if n.Severity != "" {
			existing.Severity = n.Severity
			existing.Category = n.Category
			b.nodes[n.ID] = existing
		}
		return
	}
	b.nodes[n.ID] = n
}

func (b *builder) addEdge(e Edge) {
	if e.ID == "" {
		e.ID = edgeID(e.From, e.To, e.Type)
	}
	b.edges[e.ID] = e
}

func (b *builder) finalize(input BuildInput, truncated bool) Graph {
	nodes := make([]Node, 0, len(b.nodes))
	for _, n := range b.nodes {
		nodes = append(nodes, n)
	}
	edges := make([]Edge, 0, len(b.edges))
	for _, e := range b.edges {
		edges = append(edges, e)
	}

	agg := ""
	if len(nodes) > b.cfg.MaxNodes || len(edges) > b.cfg.MaxEdges {
		truncated = true
		nodes, edges = truncateGraph(nodes, edges, b.cfg.MaxNodes, b.cfg.MaxEdges)
		agg = "package_aggregate"
	}

	byType := map[string]int{}
	entryCount := 0
	for _, n := range nodes {
		byType[n.Type]++
		if n.Entrypoint {
			entryCount++
		}
	}

	return Graph{
		RepositoryID: input.RepositoryID,
		ScanID:       input.ScanID,
		AuditID:      input.AuditID,
		Nodes:        nodes,
		Edges:        edges,
		Metrics: GraphMetrics{
			NodeCount:       len(nodes),
			EdgeCount:       len(edges),
			EntrypointCount: entryCount,
			FindingsOverlay: countFindingNodes(nodes),
			Truncated:       truncated,
			AggregationMode: agg,
			ByType:          byType,
			GeneratedAt:     time.Now().UTC(),
		},
	}
}

func truncateGraph(nodes []Node, edges []Edge, maxNodes, maxEdges int) ([]Node, []Edge) {
	if len(nodes) <= maxNodes && len(edges) <= maxEdges {
		return nodes, edges
	}
	keep := map[string]bool{}
	priority := func(t string) int {
		switch t {
		case "repo", "directory", "package", "file":
			return 3
		case "finding", "external_dependency":
			return 2
		default:
			return 1
		}
	}
	// Keep high-priority nodes first
	for len(keep) < maxNodes && len(keep) < len(nodes) {
		bestIdx := -1
		bestPri := -1
		for i, n := range nodes {
			if keep[n.ID] {
				continue
			}
			p := priority(n.Type)
			if p > bestPri {
				bestPri = p
				bestIdx = i
			}
		}
		if bestIdx < 0 {
			break
		}
		keep[nodes[bestIdx].ID] = true
	}
	outNodes := make([]Node, 0, len(keep))
	for _, n := range nodes {
		if keep[n.ID] {
			outNodes = append(outNodes, n)
		}
	}
	outEdges := make([]Edge, 0, maxEdges)
	for _, e := range edges {
		if len(outEdges) >= maxEdges {
			break
		}
		if keep[e.From] && keep[e.To] {
			outEdges = append(outEdges, e)
		}
	}
	return outNodes, outEdges
}

func nodeIDFile(path string) string   { return "file:" + filepath.ToSlash(path) }
func nodeIDDirectory(path string) string { return "dir:" + filepath.ToSlash(path) }
func nodeIDPackage(name string) string   { return "pkg:" + name }
func nodeIDFunction(file, name string) string {
	return fmt.Sprintf("fn:%s:%s", filepath.ToSlash(file), name)
}
func nodeIDExternal(name string) string { return "ext:" + name }
func nodeIDFinding(id string) string    { return "finding:" + id }

func edgeID(from, to, typ string) string { return from + "->" + to + ":" + typ }

func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".py":
		return "python"
	default:
		return "generic"
	}
}

func isTestFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return strings.HasSuffix(base, "_test.go") || strings.Contains(base, ".test.") ||
		strings.HasPrefix(base, "test_") || strings.HasSuffix(path, "/tests/")
}

func countFindingNodes(nodes []Node) int {
	n := 0
	for _, node := range nodes {
		if node.Type == "finding" {
			n++
		}
	}
	return n
}
