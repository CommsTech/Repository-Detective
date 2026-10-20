package analyzers

import (
	"math"
	"regexp"
	"strings"
	"unicode"
)

var hardcodedSecretLiteralPattern = regexp.MustCompile(`(?i)(?:password|api[_-]?key|secret|token|auth)\s*(:=|[=:])\s*["']([^"']+)["']`)

var placeholderSecretValues = []string{
	"decryption failed", "decryption_failed", "changeme", "change-me", "change_me",
	"placeholder", "example", "your-api-key", "your_api_key", "your-secret",
	"redacted", "dummy", "fake", "sample", "not set", "not_set", "undefined",
	"none", "null", "n/a", "na", "to" + "do", "fix" + "me", "x" + "xx", "yyy", "zzz",
	"insert", "replace", "enter", "password here", "secret here",
}

var strongSecretTokenPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^AKIA[0-9A-Z]{16}$`),
	regexp.MustCompile(`^ghp_[A-Za-z0-9]{20,}$`),
	regexp.MustCompile(`^github_pat_[A-Za-z0-9_]{20,}$`),
	regexp.MustCompile(`^glpat-[A-Za-z0-9_-]{20,}$`),
	regexp.MustCompile(`^xox[baprs]-[A-Za-z0-9-]{10,}$`),
	regexp.MustCompile(`^sk_(live|test)_[A-Za-z0-9]{20,}$`),
	regexp.MustCompile(`^eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`), // JWT shape
}

// hardcodedSecretAssessment captures severity/confidence for a matched literal.
type hardcodedSecretAssessment struct {
	Severity   string
	Confidence float64
	Evidence   string
	Skip       bool
}

func assessHardcodedSecret(path, line string) hardcodedSecretAssessment {
	m := hardcodedSecretLiteralPattern.FindStringSubmatch(line)
	if len(m) < 3 {
		return hardcodedSecretAssessment{Severity: "high", Confidence: 0.82}
	}
	literal := strings.TrimSpace(m[2])
	lower := strings.ToLower(literal)

	if isPlaceholderSecretValue(lower) {
		return hardcodedSecretAssessment{Skip: true}
	}

	if isExampleOrTestPath(path) {
		return hardcodedSecretAssessment{
			Severity:   "low",
			Confidence: 0.45,
			Evidence:   "Literal appears in example/test/docs path — verify before treating as a live credential.",
		}
	}

	if looksLikeStrongSecretToken(literal) {
		return hardcodedSecretAssessment{
			Severity:   "high",
			Confidence: 0.92,
			Evidence:   "Token shape matches known credential formats (API key / PAT / JWT-like). Rotate if this is a real secret.",
		}
	}

	entropy := shannonEntropy(literal)
	if entropy >= 4.0 && len(literal) >= 20 {
		return hardcodedSecretAssessment{
			Severity:   "high",
			Confidence: 0.88,
			Evidence:   "High-entropy literal assigned to a secret-like variable name — likely embedded credential.",
		}
	}

	if len(literal) < 12 || entropy < 2.5 || isMostlyWords(literal) {
		return hardcodedSecretAssessment{
			Severity:   "low",
			Confidence: 0.42,
			Evidence:   "Low-entropy or human-readable literal — often a status message or placeholder, not a live secret. Verify context.",
		}
	}

	return hardcodedSecretAssessment{
		Severity:   "medium",
		Confidence: 0.65,
		Evidence:   "Secret-like variable with embedded literal — review whether this should use environment or secret storage.",
	}
}

func isPlaceholderSecretValue(lower string) bool {
	if lower == "" {
		return true
	}
	for _, p := range placeholderSecretValues {
		if lower == p {
			return true
		}
	}
	// Phrase placeholders only when the entire literal is human-readable filler.
	for _, p := range []string{"decryption failed", "your-api-key", "your_api_key", "your-secret", "password here", "secret here"} {
		if strings.Contains(lower, p) {
			return true
		}
	}
	if strings.HasPrefix(lower, "your ") || strings.HasPrefix(lower, "insert ") || strings.HasPrefix(lower, "replace ") {
		return true
	}
	if lower == "example" || strings.HasSuffix(lower, "-example") || strings.HasSuffix(lower, "_example") {
		return true
	}
	return false
}

func isExampleOrTestPath(path string) bool {
	lower := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	switch {
	case strings.Contains(lower, "/test/"), strings.Contains(lower, "/tests/"),
		strings.Contains(lower, "/testdata/"), strings.Contains(lower, "/fixtures/"),
		strings.Contains(lower, "/examples/"), strings.Contains(lower, "/example/"),
		strings.HasSuffix(lower, "_test.go"), strings.HasSuffix(lower, "_test.py"),
		strings.HasSuffix(lower, ".example"), strings.HasSuffix(lower, ".sample"),
		strings.HasPrefix(lower, "docs/"), strings.HasPrefix(lower, "examples/"):
		return true
	}
	base := lower
	if i := strings.LastIndex(lower, "/"); i >= 0 {
		base = lower[i+1:]
	}
	switch base {
	case "readme.md", "legal.md", "contributing.md", "license.md":
		return true
	}
	return false
}

func looksLikeStrongSecretToken(literal string) bool {
	for _, p := range strongSecretTokenPatterns {
		if p.MatchString(literal) {
			return true
		}
	}
	return false
}

func isMostlyWords(s string) bool {
	if s == "" {
		return true
	}
	letters := 0
	spaces := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			letters++
		}
		if unicode.IsSpace(r) {
			spaces++
		}
	}
	return spaces >= 1 && float64(spaces)/float64(len(s)) > 0.08
}

func shannonEntropy(s string) float64 {
	if s == "" {
		return 0
	}
	freq := make(map[rune]int)
	for _, r := range s {
		freq[r]++
	}
	var entropy float64
	n := float64(len(s))
	for _, count := range freq {
		p := float64(count) / n
		entropy -= p * math.Log2(p)
	}
	return entropy
}
