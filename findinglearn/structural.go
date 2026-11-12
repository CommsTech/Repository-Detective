package findinglearn

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var (
	literalStripper = regexp.MustCompile(`"[^"]*"|'[^']*'|\b\d+\b`)
	identNormalizer = regexp.MustCompile(`\b[a-zA-Z_][a-zA-Z0-9_]*\b`)
)

// StructuralHash computes a deterministic shape hash for grouping repeated patterns.
func StructuralHash(ruleID, category, codeSnippet string) string {
	norm := normalizeShape(codeSnippet)
	payload := strings.ToLower(strings.TrimSpace(ruleID)) + "|" +
		strings.ToLower(strings.TrimSpace(category)) + "|" + norm
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:16])
}

func normalizeShape(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	code = literalStripper.ReplaceAllString(code, "LIT")
	idents := identNormalizer.FindAllString(code, -1)
	if len(idents) == 0 {
		parts := strings.Fields(code)
		if len(parts) == 0 {
			return ""
		}
		return parts[0]
	}
	for i := range idents {
		idents[i] = "ID"
	}
	return strings.Join(idents, " ")
}
