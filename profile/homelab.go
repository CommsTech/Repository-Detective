package profile

import (
	"path/filepath"
	"strings"
)

// IsHomelabInfra reports whether a repository matches homelab/infra operational patterns.
func IsHomelabInfra(p RepoProfile) bool {
	if p.Layout == LayoutInfrastructure || p.Layout == LayoutDocumentation {
		return true
	}
	if p.FileCount > 0 && p.FileCount <= 150 {
		if hasHomelabManifest(p.Manifests) {
			return true
		}
		if p.PrimaryEcosystem == EcosystemShell && p.FileCount <= 80 {
			return true
		}
	}
	if p.PrimaryEcosystem == EcosystemPython && p.FileCount <= 100 && hasHomelabManifest(p.Manifests) {
		return true
	}
	return false
}

func hasHomelabManifest(manifests []string) bool {
	for _, m := range manifests {
		base := strings.ToLower(filepath.Base(m))
		switch base {
		case "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml",
			"dockerfile", "makefile", "readme.md", "requirements.txt", "pyproject.toml", "setup.py":
			return true
		default:
			if strings.HasPrefix(base, "docker-compose.") || strings.HasPrefix(base, "dockerfile.") {
				return true
			}
		}
	}
	return false
}

// HomelabInfraCredentialPattern matches lines where internal refs may indicate secret exposure.
var homelabInfraCredentialPattern = []string{
	"password", "token", "secret", "api_key", "apikey", "credential", "auth=",
}

// ShouldDowngradeInternalInfraRef reports whether REL-INTERNAL-INFRA-REF should be informational.
func ShouldDowngradeInternalInfraRef(path, line string, p RepoProfile) bool {
	if !IsHomelabInfra(p) {
		return false
	}
	lower := strings.ToLower(line)
	for _, hint := range homelabInfraCredentialPattern {
		if strings.Contains(lower, hint) {
			return false
		}
	}
	norm := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	if strings.HasSuffix(norm, ".md") || strings.HasSuffix(norm, ".rst") {
		return true
	}
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
		return true
	}
	if strings.Contains(lower, "example") || strings.Contains(lower, "sample") || strings.Contains(lower, `write("#`) {
		return true
	}
	return true
}

// HomelabInfraSeverity returns adjusted severity/confidence for homelab repos.
func HomelabInfraSeverity(ruleID, defaultSeverity string, defaultConfidence float64, path, line string, p RepoProfile) (string, float64) {
	if ruleID == "REL-INTERNAL-INFRA-REF" && ShouldDowngradeInternalInfraRef(path, line, p) {
		return "info", 0.55
	}
	if ruleID == "QUAL-DEBUG" && IsHomelabInfra(p) {
		if strings.Contains(strings.ToLower(path), "test") || strings.Contains(strings.ToLower(path), "debug") {
			return "info", 0.5
		}
	}
	return defaultSeverity, defaultConfidence
}
