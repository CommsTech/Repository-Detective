package graph

import (
	"path/filepath"
	"regexp"
	"strings"
)

var (
	makefileRunRE     = regexp.MustCompile(`(?m)^[\w.-]+:\s*$`)
	makefileRecipeRE  = regexp.MustCompile(`(?m)^\t(?:@)?(?:\./|\./\./|bash\s+|sh\s+|python3?\s+|\./)?([\w./_-]+\.(?:sh|py|go|js|ts|ps1))\b`)
	dockerfileRunRE   = regexp.MustCompile(`(?i)(?:^|\s)(?:COPY|ADD)\s+([\w./_-]+)|(?:ENTRYPOINT|CMD)\s+\[?["']?([\w./_-]+\.(?:sh|py|go|js|ts))`)
	composeCommandRE  = regexp.MustCompile(`(?m)(?:command|entrypoint):\s*(?:\[)?["']?([\w./_-]+\.(?:sh|py|go|js|ts))`)
	readmeScriptRE    = regexp.MustCompile(`(?m)(?:^|\s)(?:\./|\./\./|bash\s+|sh\s+|python3?\s+)([\w./_-]+\.(?:sh|py|go|js|ts))\b`)
	ciRunScriptRE     = regexp.MustCompile(`(?m)(?:run:|uses:.*\n\s+with:\s*\n\s+script:)\s*["']?(?:\./|bash\s+|sh\s+|python3?\s+)?([\w./_-]+\.(?:sh|py|go|js|ts))\b`)
	pyProjectScriptRE = regexp.MustCompile(`(?m)^[\w.-]+\s*=\s*["']([\w.]+:[\w.]+)["']`)
	setupConsoleRE    = regexp.MustCompile(`(?m)entry_points\s*=|console_scripts\s*=|scripts\s*=\s*\[`)
)

func detectOperationalEntrypoints(b *builder, files []FileInput) {
	refs := collectOperationalReferences(files)
	if len(refs) == 0 {
		return
	}
	for path := range b.fileInfos {
		if matchesOperationalRef(path, refs) {
			fileID := nodeIDFile(path)
			b.entrypoints[fileID] = true
			n := b.nodes[fileID]
			n.Entrypoint = true
			b.nodes[fileID] = n
		}
	}
}

func collectOperationalReferences(files []FileInput) map[string]struct{} {
	refs := make(map[string]struct{})
	for _, f := range files {
		base := strings.ToLower(filepath.Base(f.Path))
		content := f.Content
		switch {
		case base == "makefile" || base == "makefile.am":
			addPathRefs(refs, extractMakefileRefs(content)...)
		case base == "dockerfile" || strings.HasPrefix(base, "dockerfile."):
			addPathRefs(refs, extractDockerfileRefs(content)...)
		case strings.HasPrefix(base, "docker-compose") && strings.HasSuffix(base, ".yml") ||
			strings.HasPrefix(base, "docker-compose") && strings.HasSuffix(base, ".yaml") ||
			base == "compose.yml" || base == "compose.yaml":
			addPathRefs(refs, extractComposeRefs(content)...)
		case base == "readme.md" || strings.HasPrefix(base, "readme"):
			addPathRefs(refs, extractReadmeRefs(content)...)
		case strings.Contains(f.Path, "/.github/workflows/") || strings.Contains(f.Path, "/.gitea/workflows/"):
			addPathRefs(refs, extractCIRefs(content)...)
		case base == "pyproject.toml":
			addPathRefs(refs, extractPyProjectScriptRefs(content)...)
		case base == "setup.py" || base == "setup.cfg":
			markOperationalFile(refs, f.Path)
		}
	}
	return refs
}

func extractMakefileRefs(content string) []string {
	var out []string
	for _, m := range makefileRecipeRE.FindAllStringSubmatch(content, -1) {
		if m[1] != "" {
			out = append(out, m[1])
		}
	}
	for _, line := range strings.Split(content, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "./") || strings.HasPrefix(trim, "bash ") || strings.HasPrefix(trim, "sh ") {
			for _, m := range readmeScriptRE.FindAllStringSubmatch(trim, -1) {
				if m[1] != "" {
					out = append(out, m[1])
				}
			}
		}
	}
	_ = makefileRunRE
	return out
}

func extractDockerfileRefs(content string) []string {
	var out []string
	for _, m := range dockerfileRunRE.FindAllStringSubmatch(content, -1) {
		if m[1] != "" {
			out = append(out, m[1])
		}
		if len(m) > 2 && m[2] != "" {
			out = append(out, m[2])
		}
	}
	return out
}

func extractComposeRefs(content string) []string {
	var out []string
	for _, m := range composeCommandRE.FindAllStringSubmatch(content, -1) {
		if m[1] != "" {
			out = append(out, m[1])
		}
	}
	return out
}

func extractReadmeRefs(content string) []string {
	var out []string
	for _, m := range readmeScriptRE.FindAllStringSubmatch(content, -1) {
		if m[1] != "" {
			out = append(out, m[1])
		}
	}
	return out
}

func extractCIRefs(content string) []string {
	var out []string
	for _, m := range ciRunScriptRE.FindAllStringSubmatch(content, -1) {
		if m[1] != "" {
			out = append(out, m[1])
		}
	}
	return out
}

func extractPyProjectScriptRefs(content string) []string {
	var out []string
	for _, m := range pyProjectScriptRE.FindAllStringSubmatch(content, -1) {
		if m[1] != "" {
			parts := strings.SplitN(m[1], ":", 2)
			if len(parts) == 2 {
				out = append(out, strings.ReplaceAll(parts[0], ".", "/")+".py")
			}
		}
	}
	if setupConsoleRE.MatchString(content) {
		out = append(out, "__main__.py", "main.py", "cli.py")
	}
	return out
}

func addPathRefs(refs map[string]struct{}, paths ...string) {
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		p = filepath.ToSlash(p)
		p = strings.TrimPrefix(p, "./")
		refs[p] = struct{}{}
		refs[filepath.Base(p)] = struct{}{}
	}
}

func markOperationalFile(refs map[string]struct{}, path string) {
	path = filepath.ToSlash(path)
	refs[path] = struct{}{}
	refs[filepath.Base(path)] = struct{}{}
}

func matchesOperationalRef(path string, refs map[string]struct{}) bool {
	path = filepath.ToSlash(path)
	if _, ok := refs[path]; ok {
		return true
	}
	base := filepath.Base(path)
	if _, ok := refs[base]; ok {
		return true
	}
	for ref := range refs {
		if strings.HasSuffix(path, "/"+ref) || path == ref {
			return true
		}
	}
	return false
}
