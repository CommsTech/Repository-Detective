package ui

import (
	"path"
	"regexp"
	"strings"
)

var (
	legacyScanScratchRE = regexp.MustCompile(`(?i)(?:^|/)(?:app/)?data/tmp/(?:bugbot|repository-detective)-scan-\d+(?:/(?:data/tmp/(?:bugbot|repository-detective)-scan-\d+))*/`)
	legacyScratchNameRE = regexp.MustCompile(`(?i)\b(?:bugbot|repository-detective)-scan-\d+\b`)
)

// scrubLegacyBrand rewrites operator-visible legacy product names and forge paths
// so the UI never surfaces "Bugbot" branding from historical scan/error data.
func scrubLegacyBrand(value string) string {
	if value == "" {
		return value
	}
	out := value
	replacements := []struct{ old, new string }{
		{"git.commsnet.org/api/v1/repos/commstech/Bugbot", "git.commsnet.org/api/v1/repos/commstech/Repository-Detective"},
		{"git.commsnet.org/commstech/Bugbot", "git.commsnet.org/commstech/Repository-Detective"},
		{"commstech/Bugbot", "commstech/Repository-Detective"},
		{"X-Bugbot-API-Key", "X-Repository-Detective-API-Key"},
		{"BUGBOT_", "REPOSITORY_DETECTIVE_"},
		{"bugbot-scan-", "repository-detective-scan-"},
		{"Bugbot", "Repository Detective"},
		{"bugbot", "repository-detective"},
	}
	for _, r := range replacements {
		out = strings.ReplaceAll(out, r.old, r.new)
	}
	return out
}

// displayPath cleans scanner workspace prefixes and legacy brand tokens from file paths.
func displayPath(filePath string) string {
	if strings.TrimSpace(filePath) == "" {
		return filePath
	}
	out := strings.ReplaceAll(filePath, "\\", "/")
	for {
		next := legacyScanScratchRE.ReplaceAllString(out, "/")
		if next == out {
			break
		}
		out = next
	}
	out = legacyScratchNameRE.ReplaceAllString(out, "")
	out = strings.ReplaceAll(out, "//", "/")
	out = strings.TrimPrefix(out, "/")
	out = scrubLegacyBrand(out)
	if out == "" {
		return scrubLegacyBrand(path.Base(filePath))
	}
	return out
}

// displayBrandText is the template helper for any free-form operator-facing string.
func displayBrandText(value string) string {
	return scrubLegacyBrand(value)
}
