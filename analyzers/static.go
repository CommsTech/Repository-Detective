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
	regexp.MustCompile(`\$\{|mapstructure:|` + "`json:" + `|data-api[-_]key|{{\s*\.`),
}

var safeSQLConcatSuffix = regexp.MustCompile(`(?i)\+\s*` + "`" + `[^` + "`" + `]*\?`)

// FileContent holds fetched source for analysis.
type FileContent struct {
	Path     string
	Content  string
	Language string
}

func skipStaticAnalysisPath(path string) bool {
	path = strings.ReplaceAll(path, "\\", "/")
	lower := strings.ToLower(path)
	switch {
	case strings.Contains(path, "/vendor/"), strings.HasPrefix(path, "vendor/"):
		return true
	case strings.HasSuffix(path, "_test.go"), strings.HasSuffix(path, "_test.js"), strings.HasSuffix(path, "_test.py"):
		return true
	case strings.Contains(path, "/testdata/"), strings.Contains(path, "/fixtures/"):
		return true
	case strings.HasPrefix(path, "web/static/"), strings.HasPrefix(path, "docs/"):
		return true
	case strings.HasPrefix(path, "ui/templates/"), strings.HasPrefix(path, "ui/static/"):
		return true
	case strings.HasSuffix(lower, ".sh"), strings.HasSuffix(lower, ".bash"):
		return true
	case strings.HasPrefix(lower, "scripts/"):
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

func isStaticFalsePositive(rule staticRule, path, line string) bool {
	switch rule.ID {
	case "SEC-HARDCODED-SECRET":
		return isFalsePositiveHardcodedSecret(path, line)
	case "SEC-SQL-CONCAT":
		return isFalsePositiveSQLConcat(line)
	case "QUAL-DEBUG":
		return isFalsePositiveDebugLine(path, line)
	default:
		return false
	}
}

func isFalsePositiveHardcodedSecret(path, line string) bool {
	lower := strings.ToLower(line)
	// Shell/Python/Go reading secrets from environment, not embedding literals.
	if strings.Contains(line, "${") || strings.Contains(line, ":-") || strings.Contains(line, "os.Getenv") ||
		strings.Contains(line, "viper.") || strings.Contains(line, "process.env") {
		return true
	}
	// HTML/JS data attributes and template query params (e.g. data-api-key="{{.APIKey}}").
	if strings.Contains(lower, "data-api") || strings.Contains(lower, "api_key=") && strings.Contains(line, "{{") {
		return true
	}
	// Struct/config field names without string literal secrets.
	if strings.Contains(line, "mapstructure:") || strings.Contains(line, "`json:") {
		return true
	}
	// Variable names that mention api_key but assign from another variable.
	if regexp.MustCompile(`(?i)\b(local\s+)?[a-z_]*api[_-]?key\s*=\s*["']?\$\{`).MatchString(line) {
		return true
	}
	if regexp.MustCompile(`(?i)api[_-]?key\s*=\s*["']\$\{`).MatchString(line) {
		return true
	}
	return false
}

func isFalsePositiveSQLConcat(line string) bool {
	trimmed := strings.TrimSpace(line)
	// Safe pattern: append a constant SQL fragment that only adds placeholders (?, $1).
	if safeSQLConcatSuffix.MatchString(trimmed) {
		return true
	}
	if strings.Contains(trimmed, "+") && strings.Contains(trimmed, "?") &&
		!strings.Contains(trimmed, "+ \"") && !strings.Contains(trimmed, "+'") &&
		!strings.Contains(trimmed, "fmt.Sprintf") && !strings.Contains(trimmed, " + ") {
		// e.g. query := base + ` WHERE id = ?`
		if strings.Contains(trimmed, "`") {
			return true
		}
	}
	// Go/sql comment or test scaffolding.
	if strings.Contains(trimmed, "sqlmock") || strings.Contains(trimmed, "SELECT 1") {
		return true
	}
	return false
}

func isFalsePositiveDebugLine(path, line string) bool {
	lowerPath := strings.ToLower(path)
	if strings.Contains(lowerPath, "/cmd/") || strings.HasSuffix(lowerPath, "main.go") {
		return strings.Contains(line, "fmt.Println") && strings.Contains(line, "usage")
	}
	return false
}

func staticRuleConfidence(rule staticRule) float64 {
	switch rule.ID {
	case "SEC-EVAL":
		return 0.95
	case "SEC-HARDCODED-SECRET", "SEC-SQL-CONCAT", "SEC-CMD-EXEC":
		return 0.82
	default:
		return 0.75
	}
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
				if isStaticFalsePositive(rule, file.Path, line) {
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
					Confidence:  staticRuleConfidence(rule),
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
