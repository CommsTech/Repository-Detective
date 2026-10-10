package containers

import (
	"regexp"
	"strings"
)

var (
	dockerfileFrom = regexp.MustCompile(`(?i)^FROM\s+(?P<img>[^\s]+)`)
	composeImage   = regexp.MustCompile(`(?i)^\s*image:\s*["']?([^"'\s#]+)["']?`)
	k8sImage       = regexp.MustCompile(`(?i)image:\s*["']?([^"'\s#]+)["']?`)
)

// FileInput is one file for discovery.
type FileInput struct {
	Path    string
	Content string
}

// DiscoverImages extracts container image references from repository files.
func DiscoverImages(files []FileInput, repoID int64) []ImageReference {
	var out []ImageReference
	seen := make(map[string]struct{})
	for _, file := range files {
		path := strings.ReplaceAll(file.Path, "\\", "/")
		lower := strings.ToLower(path)
		switch {
		case strings.HasSuffix(lower, "dockerfile") || strings.Contains(lower, "/dockerfile"):
			out = append(out, discoverDockerfile(file, repoID, seen)...)
		case isComposeFile(lower):
			out = append(out, discoverCompose(file, repoID, seen)...)
		case isKubernetesYAML(lower):
			out = append(out, discoverKubernetes(file, repoID, seen)...)
		case strings.Contains(lower, ".gitea/workflows/") && strings.HasSuffix(lower, ".yml"):
			out = append(out, discoverWorkflowImages(file, repoID, seen)...)
		}
	}
	return out
}

func isComposeFile(path string) bool {
	base := path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		base = path[i+1:]
	}
	switch base {
	case "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml":
		return true
	default:
		return strings.HasPrefix(base, "docker-compose.") && strings.HasSuffix(base, ".yml")
	}
}

func isKubernetesYAML(path string) bool {
	if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
		return false
	}
	lower := strings.ToLower(path)
	return strings.Contains(lower, "kubernetes") || strings.Contains(lower, "k8s") ||
		strings.Contains(lower, "/deploy/") || strings.Contains(lower, "/manifests/")
}

func discoverDockerfile(file FileInput, repoID int64, seen map[string]struct{}) []ImageReference {
	var out []ImageReference
	lines := strings.Split(file.Content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		m := dockerfileFrom.FindStringSubmatch(trimmed)
		if len(m) < 2 {
			continue
		}
		img := strings.TrimSpace(m[1])
		if img == "" || strings.EqualFold(img, "scratch") {
			continue
		}
		ref := makeReference(img, TargetRegistryImage, file.Path, i+1, "", repoID)
		if _, ok := seen[ref.Image]; ok {
			continue
		}
		seen[ref.Image] = struct{}{}
		out = append(out, ref)
	}
	return out
}

func discoverCompose(file FileInput, repoID int64, seen map[string]struct{}) []ImageReference {
	var out []ImageReference
	lines := strings.Split(file.Content, "\n")
	service := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasSuffix(trimmed, ":") && !strings.Contains(trimmed, " ") && !strings.HasPrefix(trimmed, "-") {
			service = strings.TrimSuffix(trimmed, ":")
			continue
		}
		m := composeImage.FindStringSubmatch(line)
		if len(m) < 2 {
			continue
		}
		ref := makeReference(strings.TrimSpace(m[1]), TargetComposeFile, file.Path, i+1, service, repoID)
		if _, ok := seen[ref.Image]; ok {
			continue
		}
		seen[ref.Image] = struct{}{}
		out = append(out, ref)
	}
	return out
}

func discoverKubernetes(file FileInput, repoID int64, seen map[string]struct{}) []ImageReference {
	var out []ImageReference
	lines := strings.Split(file.Content, "\n")
	for i, line := range lines {
		m := k8sImage.FindStringSubmatch(line)
		if len(m) < 2 {
			continue
		}
		img := strings.TrimSpace(m[1])
		if strings.Contains(img, "{{") {
			continue
		}
		ref := makeReference(img, TargetKubernetesManifest, file.Path, i+1, "", repoID)
		if _, ok := seen[ref.Image]; ok {
			continue
		}
		seen[ref.Image] = struct{}{}
		out = append(out, ref)
	}
	return out
}

func discoverWorkflowImages(file FileInput, repoID int64, seen map[string]struct{}) []ImageReference {
	return discoverCompose(file, repoID, seen) // image: lines in workflow YAML
}

func makeReference(raw string, target TargetType, path string, line int, service string, repoID int64) ImageReference {
	raw = strings.TrimSpace(raw)
	tag := ""
	digest := ""
	mutable := false
	private := false

	if at := strings.Index(raw, "@"); at > 0 {
		digest = raw[at+1:]
		raw = raw[:at]
	}
	if colon := strings.LastIndex(raw, ":"); colon > strings.LastIndex(raw, "/") {
		tag = raw[colon+1:]
		raw = raw[:colon]
	}
	full := raw
	if tag != "" {
		full = raw + ":" + tag
	}
	if digest != "" {
		full = full + "@" + digest
	} else if tag == "" {
		mutable = true
		full = full + ":latest"
		tag = "latest"
	}
	if tag == "latest" || tag == "" {
		mutable = true
	}
	lower := strings.ToLower(full)
	if strings.Contains(lower, "localhost") || strings.Contains(lower, "127.0.0.1") ||
		(!strings.Contains(full, "docker.io") && !strings.Contains(full, "/") && !strings.Contains(full, ".")) {
		private = true
	}

	return ImageReference{
		Image:           full,
		Tag:             tag,
		Digest:          digest,
		TargetType:      target,
		FilePath:        path,
		Line:            line,
		ServiceName:     service,
		MutableTag:      mutable,
		PrivateRegistry: private,
		RepoID:          repoID,
	}
}

// DiscoveryFindings converts references into policy findings for persistence.
func DiscoveryFindings(refs []ImageReference) []ScanFinding {
	var out []ScanFinding
	for _, ref := range refs {
		out = append(out, ScanFinding{
			RuleID: "CONTAINER-IMAGE-REFERENCE", Severity: "info", Confidence: 0.9,
			Title:       "Container image reference discovered",
			Description: "Image " + ref.Image + " referenced in " + ref.FilePath,
		})
		if ref.MutableTag {
			out = append(out, ScanFinding{
				RuleID: "CONTAINER-MUTABLE-TAG", Severity: "low", Confidence: 0.85,
				Title:       "Mutable container image tag",
				Description: "Image uses mutable tag; prefer digest pinning for production.",
			})
		}
		if ref.Digest == "" {
			out = append(out, ScanFinding{
				RuleID: "CONTAINER-NO-DIGEST", Severity: "info", Confidence: 0.8,
				Title:       "Container image not digest-pinned",
				Description: "No digest pin detected in manifest reference.",
			})
		}
		out = append(out, ScanFinding{
			RuleID: "CONTAINER-UNSCANNED-IMAGE", Severity: "info", Confidence: 0.75,
			Title:       "Container image not yet scanned",
			Description: "Run a container image scan via runner to assess vulnerabilities.",
		})
	}
	return out
}
