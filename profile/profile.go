package profile

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
)

// RepoProfile captures repository structure before scanners run.
type RepoProfile struct {
	Layout           string           `json:"layout"`
	PrimaryEcosystem string           `json:"primary_ecosystem"`
	Ecosystems       []string         `json:"ecosystems"`
	Manifests        []string         `json:"manifests"`
	IgnorePaths      []string         `json:"ignore_paths"`
	ReportingHints   ReportingHints   `json:"reporting_hints"`
	FileCount        int              `json:"file_count"`
	LanguageCounts   map[string]int   `json:"language_counts,omitempty"`
	Subpaths         []SubpathProfile `json:"subpaths,omitempty"`
	ProfileVersion   string           `json:"profile_version"`
}

// SubpathProfile describes a nested app/service within a monorepo.
type SubpathProfile struct {
	Path       string   `json:"path"`
	Layout     string   `json:"layout"`
	Ecosystems []string `json:"ecosystems"`
	Manifests  []string `json:"manifests"`
}

// ReportingHints captures repo-specific issue filing conventions.
type ReportingHints struct {
	GiteaIssueTemplates  []string `json:"gitea_issue_templates,omitempty"`
	GithubIssueTemplates []string `json:"github_issue_templates,omitempty"`
	HasContributing      bool     `json:"has_contributing"`
	HasSecurity          bool     `json:"has_security"`
	HasCodeowners        bool     `json:"has_codeowners"`
	DocsBugReport        bool     `json:"docs_bug_report"`
	DocsSecurity         bool     `json:"docs_security"`
}

// DetectProfile builds a repository profile from indexed file paths.
func DetectProfile(paths []string) RepoProfile {
	p := RepoProfile{
		ProfileVersion: "1",
		LanguageCounts: make(map[string]int),
		IgnorePaths:    defaultIgnorePaths(),
	}
	if len(paths) == 0 {
		p.Layout = LayoutMixed
		p.PrimaryEcosystem = EcosystemUnknown
		p.Ecosystems = []string{EcosystemUnknown}
		return p
	}

	manifestSet := map[string]struct{}{}
	ecoCounts := map[string]int{}
	var analyzable []string

	for _, raw := range paths {
		path := NormalizePath(raw)
		if path == "" {
			continue
		}
		p.FileCount++

		if IsIgnoredPath(path) {
			continue
		}
		analyzable = append(analyzable, path)

		base := strings.ToLower(filepath.Base(path))
		if KnownManifestBasenames[base] {
			manifestSet[path] = struct{}{}
		}
		if strings.Contains(path, "/.github/workflows/") || strings.Contains(path, "/.gitea/workflows/") {
			manifestSet[path] = struct{}{}
		}

		lang := detectLanguageFromPath(path)
		p.LanguageCounts[lang]++
		for _, eco := range ecosystemsFromPath(path, base) {
			ecoCounts[eco]++
		}

		detectReportingFile(path, &p.ReportingHints)
	}

	p.Manifests = sortedKeys(manifestSet)
	p.Ecosystems = sortedEcoKeys(ecoCounts)
	p.PrimaryEcosystem = primaryEcosystem(ecoCounts)
	p.Layout = detectLayout(analyzable, p.Manifests, ecoCounts, p.LanguageCounts)
	p.Subpaths = detectSubpaths(analyzable, p.Manifests)

	return p
}

// JSON returns a compact JSON representation for persistence.
func (p RepoProfile) JSON() []byte {
	b, _ := json.Marshal(p)
	return b
}

func defaultIgnorePaths() []string {
	out := make([]string, len(ignoreDirSegments))
	copy(out, ignoreDirSegments)
	sort.Strings(out)
	return out
}

func detectLanguageFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js", ".jsx":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".sh", ".bash":
		return "shell"
	case ".md", ".rst":
		return "markdown"
	case ".yaml", ".yml":
		return "yaml"
	case ".tf":
		return "terraform"
	default:
		return "other"
	}
}

func ecosystemsFromPath(path, base string) []string {
	var out []string
	lower := strings.ToLower(path)
	ext := strings.ToLower(filepath.Ext(path))

	switch {
	case ext == ".go" || base == "go.mod":
		out = append(out, EcosystemGo)
	case ext == ".py" || base == "pyproject.toml" || base == "requirements.txt":
		out = append(out, EcosystemPython)
	case ext == ".js" || ext == ".jsx" || ext == ".ts" || ext == ".tsx" || base == "package.json":
		out = append(out, EcosystemJavaScript)
		if ext == ".ts" || ext == ".tsx" {
			out = append(out, EcosystemTypeScript)
		}
	case ext == ".sh" || ext == ".bash":
		out = append(out, EcosystemShell)
	case base == "dockerfile" || strings.HasPrefix(base, "dockerfile.") ||
		base == "docker-compose.yml" || base == "docker-compose.yaml":
		out = append(out, EcosystemDocker)
	case base == "chart.yaml" || base == "kustomization.yaml" ||
		(strings.Contains(lower, "/deploy/") && (ext == ".yaml" || ext == ".yml")):
		out = append(out, EcosystemKubernetes)
	case ext == ".tf" || ext == ".tfvars":
		out = append(out, EcosystemTerraform)
	case strings.Contains(lower, "/ansible/") || base == "playbook.yml":
		out = append(out, EcosystemAnsible)
	case ext == ".md" || ext == ".rst":
		out = append(out, EcosystemMarkdownDocs)
	}
	if len(out) == 0 {
		out = append(out, EcosystemUnknown)
	}
	return out
}

func detectReportingFile(path string, hints *ReportingHints) {
	lower := strings.ToLower(NormalizePath(path))
	base := strings.ToLower(filepath.Base(lower))

	switch {
	case strings.Contains(lower, ".gitea/issue_template/"):
		hints.GiteaIssueTemplates = append(hints.GiteaIssueTemplates, path)
	case strings.Contains(lower, ".github/issue_template/"):
		hints.GithubIssueTemplates = append(hints.GithubIssueTemplates, path)
	case base == "contributing.md":
		hints.HasContributing = true
	case base == "security.md":
		hints.HasSecurity = true
	case base == "codeowners":
		hints.HasCodeowners = true
	case base == "bug_report.md" && strings.Contains(lower, "/docs/"):
		hints.DocsBugReport = true
	case base == "security.md" && strings.Contains(lower, "/docs/"):
		hints.DocsSecurity = true
	}
}

func detectLayout(paths []string, manifests []string, ecoCounts map[string]int, langCounts map[string]int) string {
	if len(paths) == 0 {
		return LayoutMixed
	}

	docsOnly := ecoCounts[EcosystemMarkdownDocs] > 0 &&
		ecoCounts[EcosystemGo]+ecoCounts[EcosystemPython]+ecoCounts[EcosystemJavaScript] == 0 &&
		len(manifests) == 0
	if docsOnly && langCounts["markdown"] >= len(paths)/2 {
		return LayoutDocumentation
	}

	infraScore := ecoCounts[EcosystemTerraform] + ecoCounts[EcosystemKubernetes] +
		ecoCounts[EcosystemAnsible] + ecoCounts[EcosystemDocker]
	appScore := ecoCounts[EcosystemGo] + ecoCounts[EcosystemPython] + ecoCounts[EcosystemJavaScript]

	if infraScore > 0 && appScore == 0 {
		return LayoutInfrastructure
	}

	rootManifests := 0
	nestedManifests := 0
	for _, m := range manifests {
		if strings.Count(m, "/") <= 1 {
			rootManifests++
		} else {
			nestedManifests++
		}
	}

	if nestedManifests >= 2 || (rootManifests >= 1 && nestedManifests >= 1) {
		return LayoutMonorepo
	}
	if nestedManifests == 1 && rootManifests == 0 {
		return LayoutNestedServices
	}

	if rootManifests == 0 && appScore == 0 && infraScore == 0 {
		if docsOnly {
			return LayoutDocumentation
		}
		return LayoutMixed
	}

	if rootManifests == 1 && nestedManifests == 0 {
		base := strings.ToLower(filepath.Base(manifests[0]))
		if base == "go.mod" {
			hasMain := false
			for _, p := range paths {
				norm := NormalizePath(p)
				if strings.Contains(norm, "/cmd/") || strings.HasSuffix(norm, "/main.go") || norm == "main.go" {
					hasMain = true
					break
				}
			}
			if !hasMain {
				return LayoutLibrary
			}
		}
		return LayoutSingleApp
	}

	return LayoutMixed
}

func detectSubpaths(paths []string, manifests []string) []SubpathProfile {
	byRoot := map[string]*SubpathProfile{}
	for _, m := range manifests {
		dir := filepath.Dir(m)
		if dir == "." {
			continue
		}
		root := strings.SplitN(NormalizePath(dir), "/", 2)[0]
		sp, ok := byRoot[root]
		if !ok {
			sp = &SubpathProfile{Path: root, Layout: LayoutSingleApp}
			byRoot[root] = sp
		}
		sp.Manifests = append(sp.Manifests, m)
	}

	for _, p := range paths {
		parts := strings.SplitN(NormalizePath(p), "/", 2)
		if len(parts) < 2 {
			continue
		}
		root := parts[0]
		sp, ok := byRoot[root]
		if !ok {
			continue
		}
		base := strings.ToLower(filepath.Base(p))
		for _, eco := range ecosystemsFromPath(p, base) {
			if eco != EcosystemUnknown && !containsString(sp.Ecosystems, eco) {
				sp.Ecosystems = append(sp.Ecosystems, eco)
			}
		}
	}

	out := make([]SubpathProfile, 0, len(byRoot))
	for _, sp := range byRoot {
		sort.Strings(sp.Manifests)
		sort.Strings(sp.Ecosystems)
		out = append(out, *sp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func primaryEcosystem(counts map[string]int) string {
	best := EcosystemUnknown
	bestCount := 0
	for eco, n := range counts {
		if eco == EcosystemUnknown || eco == EcosystemMarkdownDocs {
			continue
		}
		if n > bestCount {
			bestCount = n
			best = eco
		}
	}
	if bestCount == 0 {
		if counts[EcosystemMarkdownDocs] > 0 {
			return EcosystemMarkdownDocs
		}
		return EcosystemMixed
	}
	return best
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedEcoKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k, n := range m {
		if n > 0 && k != EcosystemUnknown {
			out = append(out, k)
		}
	}
	if len(out) == 0 {
		return []string{EcosystemUnknown}
	}
	sort.Strings(out)
	return out
}

func containsString(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

// IsDocsOnlyRepo reports whether the profile represents a documentation-only repository.
func (p RepoProfile) IsDocsOnlyRepo() bool {
	return p.Layout == LayoutDocumentation
}
