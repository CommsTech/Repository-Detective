package security

import (
	"path/filepath"
	"regexp"
	"strings"
)

var safeFilenamePart = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// SafeAttachmentFilename returns a basename safe for Content-Disposition headers.
func SafeAttachmentFilename(parts ...string) string {
	var cleaned []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		part = filepath.Base(part)
		part = strings.ReplaceAll(part, "..", "")
		part = safeFilenamePart.ReplaceAllString(part, "-")
		part = strings.Trim(part, "-.")
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	if len(cleaned) == 0 {
		return "download"
	}
	return strings.Join(cleaned, "-")
}
