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
		Pattern:     regexp.MustCompile(`(?i)(password|api[_-]?key|secret|token|auth)\s*(:=|[=:])\s*["'][^"']{8,}["']`),
	},
	{
		ID: "SEC-EVAL", Category: "code_injection", Severity: "critical",
		Title:       "Dynamic code execution",
		Description: "Use of eval or equivalent dynamic execution can allow arbitrary code injection.",
		Pattern:     regexp.MustCompile(`(?i)\beval\s*\(|new\s+Function\s*\(`),
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

// FileContent holds fetched source for analysis.
type FileContent struct {
	Path    string
	Content string
	Language string
}

// RunStaticAnalysis performs deterministic pattern checks without LLM calls.
func RunStaticAnalysis(files []FileContent, enableSecurity, enableQuality bool) []models.CandidateFinding {
	var findings []models.CandidateFinding

	for _, file := range files {
		if file.Content == "" {
			continue
		}
		lines := strings.Split(file.Content, "\n")
		for lineNum, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
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
					File:        file.Path,
					Line:        lineNum + 1,
				})
			}
		}
	}

	return findings
}
