package analyzers

import (
	"regexp"
	"strings"

	"git.commsnet.org/commstech/bugbot/models"
)

type staticRule struct {
	ID          string
	Category    string
	Severity    string
	Title       string
	Description string
	Pattern     *regexp.Regexp
}

var staticRules = []staticRule{
	{
		ID: "SEC-SQL-CONCAT", Category: "sql_injection", Severity: "high",
		Title:       "Possible SQL injection via string concatenation",
		Description: "User-controlled data may be concatenated into a SQL query instead of using parameterized queries.",
		Pattern:     regexp.MustCompile(`(?i)(query|sql)\s*:?=.*(\+|\|\||fmt\.Sprintf).*`),
	},
	{
		ID: "SEC-HARDCODED-SECRET", Category: "hardcoded_secret", Severity: "high",
		Title:       "Possible hardcoded secret",
		Description: "A literal that looks like a password, API key, or token is embedded in source code.",
		Pattern:     regexp.MustCompile(`(?i)(^|[^A-Z0-9_])(password|api[_-]?key|secret|token|auth)\s*(:=|[=:])\s*["'][^"']{8,}["']`),
	},
	{
		ID: "SEC-EVAL", Category: "code_injection", Severity: "critical",
		Title:       "Dynamic code execution",
		Description: "Use of eval or equivalent dynamic execution can allow arbitrary code injection.",
		Pattern:     regexp.MustCompile(`(?i)(^|[^a-zA-Z0-9_])eval\s*\(|new\s+Function\s*\(`),
	},
	{
		ID: "SEC-XSS-INNERHTML", Category: "xss", Severity: "medium",
		Title:       "Possible DOM XSS via innerHTML",
		Description: "Assigning untrusted content to innerHTML can lead to cross-site scripting.",
		Pattern:     regexp.MustCompile(`(?i)\.innerHTML\s*=|dangerouslySetInnerHTML`),
	},
	{
		ID: "SEC-CMD-EXEC", Category: "command_injection", Severity: "high",
		Title:       "Possible command execution with dynamic input",
		Description: "Shell or process execution combined with string formatting may allow command injection.",
		Pattern:     regexp.MustCompile(`(?i)(exec\.Command|os/system|subprocess\.(call|Popen)|Runtime\.getRuntime\(\)\.exec).*(\+|\|\||fmt\.Sprintf|%s)`),
	},
	{
		ID: "QUAL-DEBUG", Category: "quality", Severity: "low",
		Title:       "Debug logging left in code",
		Description: "Debug print statements should be removed or gated before production.",
		Pattern:     regexp.MustCompile(`(?i)\b(console\.log|fmt\.Println|print\(|debugger)\b`),
	},
}

var staticLineSkipPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)REDACTED|EXAMPLE|YOUR[-_ ]?API[-_ ]?KEY|your-api-key|changeme|user_input|userInput|AKIA[0-9A-Z]{16}`),
}

// FileContent holds fetched source for analysis.
type FileContent struct {
	Path     string
	Content  string
	Language string
}

func skipStaticAnalysisPath(path string) bool {
	path = strings.ReplaceAll(path, "\\", "/")
	switch {
	case strings.Contains(path, "/vendor/"), strings.HasPrefix(path, "vendor/"):
		return true
	case strings.HasSuffix(path, "_test.go"), strings.HasSuffix(path, "_test.js"), strings.HasSuffix(path, "_test.py"):
		return true
	case strings.Contains(path, "/testdata/"), strings.Contains(path, "/fixtures/"):
		return true
	case strings.HasPrefix(path, "web/static/"), strings.HasPrefix(path, "docs/"):
		return true
	case path == "ai/client.go":
		return true
	}
	return false
}

func skipStaticAnalysisLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
		return true
	}
	for _, pattern := range staticLineSkipPatterns {
		if pattern.MatchString(line) {
			return true
		}
	}
	return false
}

// RunStaticAnalysis performs deterministic pattern checks without LLM calls.
func RunStaticAnalysis(files []FileContent, enableSecurity, enableQuality bool) []models.CandidateFinding {
	var findings []models.CandidateFinding

	for _, file := range files {
		if file.Content == "" || skipStaticAnalysisPath(file.Path) {
			continue
		}
		lines := strings.Split(file.Content, "\n")
		for lineNum, line := range lines {
			if skipStaticAnalysisLine(line) {
				continue
			}
			for _, rule := range staticRules {
				if !enableSecurity && rule.Category != "quality" {
					continue
				}
				if !enableQuality && rule.Category == "quality" {
					continue
				}
				if !rule.Pattern.MatchString(line) {
					continue
				}
				findings = append(findings, models.CandidateFinding{
					ID:         rule.ID,
					Hypothesis: rule.Title,
					Evidence: models.Evidence{
						Code:      strings.TrimSpace(line),
						CallChain: []string{file.Path},
					},
					Severity:    rule.Severity,
					Confidence:  0.92,
					AuditorType: "static",
					Category:    rule.Category,
					File:        file.Path,
					Line:        lineNum + 1,
				})
			}
		}
	}

	return findings
}
