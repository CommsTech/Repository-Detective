package analyzers

import (
	"regexp"
	"strings"

	"git.commsnet.org/commstech/repository-detective/models"
	"git.commsnet.org/commstech/repository-detective/profile"
)

type staticRule struct {
	ID          string
	Category    string
	Severity    string
	Title       string
	Description string
	Pattern     *regexp.Regexp
	Advisory    bool
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
	{
		ID: "OPT-NESTED-LOOP", Category: "optimization", Severity: "low",
		Title:       "Possible inefficient nested loop",
		Description: "Nested loops can become an O(n^2) hotspot on large inputs. Treat as advisory and verify with profiling before optimizing.",
		Pattern:     regexp.MustCompile(`(?i)\b(for|foreach|while)\b.*\b(for|foreach|while)\b`),
		Advisory:    true,
	},
	{
		ID: "OPT-HTTP-CLIENT-PER-CALL", Category: "optimization", Severity: "low",
		Title:       "HTTP client may be created per call",
		Description: "Creating HTTP clients in hot paths prevents connection reuse. Prefer a shared client or pool when this path is performance sensitive.",
		Pattern:     regexp.MustCompile(`(?i)new\s+HttpClient\s*\(|http\.Client\s*\{`),
		Advisory:    true,
	},
	{
		ID: "GOV-ACTION-FLOATING-REF", Category: "pipeline_governance", Severity: "medium",
		Title:       "Workflow action uses a floating reference",
		Description: "Third-party workflow actions should be pinned to an immutable commit SHA so upstream tag changes cannot alter pipeline behavior.",
		Pattern:     regexp.MustCompile(`(?i)uses:\s*[\w./-]+@(?:main|master|v?\d+(?:\.\d+){0,2})\b`),
		Advisory:    true,
	},
	{
		ID: "GOV-PIPELINE-SECRET-ECHO", Category: "pipeline_governance", Severity: "high",
		Title:       "Pipeline may print secret material",
		Description: "Pipeline steps must not echo tokens, passwords, or secrets into build logs.",
		Pattern:     regexp.MustCompile(`(?i)\b(echo|printf)\b.*\$\{?\{?(?:.*secret|.*token|.*password|.*api[_-]?key)`),
	},
	{
		ID: "REL-INTERNAL-INFRA-REF", Category: "public_release", Severity: "medium",
		Title:       "Possible internal infrastructure reference",
		Description: "Public-release review should remove internal hostnames, private IPs, and environment-specific endpoints from code, tests, and docs.",
		Pattern:     regexp.MustCompile(`(?i)\b(10\.\d{1,3}\.\d{1,3}\.\d{1,3}|172\.(1[6-9]|2\d|3[01])\.\d{1,3}\.\d{1,3}|192\.168\.\d{1,3}\.\d{1,3}|localhost|\.local|\.internal)\b`),
		Advisory:    true,
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
	case strings.Contains(path, "/benchmark/fixture/"), strings.HasPrefix(path, "benchmark/fixture/"):
		return true
	case strings.HasSuffix(lower, ".go.src"):
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
	case strings.HasPrefix(path, "analyzers/static"), path == "static.go":
		// Rule definitions must not self-match (see Gitea #38).
		return true
	case strings.HasSuffix(lower, ".example"):
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
	if strings.Contains(line, "regexp.MustCompile") || strings.Contains(line, "Pattern:") {
		return true
	}
	return false
}

func isStaticFalsePositive(rule staticRule, path, line string) bool {
	switch rule.ID {
	case "SEC-HARDCODED-SECRET":
		return isFalsePositiveHardcodedSecret(path, line)
	case "SEC-SQL-CONCAT":
		return isFalsePositiveSQLConcat(path, line)
	case "SEC-CMD-EXEC":
		return isFalsePositiveCmdExec(path, line)
	case "SEC-EVAL":
		return isFalsePositiveEval(path, line)
	case "QUAL-DEBUG":
		return isFalsePositiveDebugLine(path, line)
	case "REL-INTERNAL-INFRA-REF":
		return isFalsePositiveInternalInfraRef(path, line)
	case "OPT-HTTP-CLIENT-PER-CALL":
		return isFalsePositiveHTTPClientPerCall(path, line)
	case "OPT-NESTED-LOOP":
		return isFalsePositiveNestedLoop(path)
	default:
		return false
	}
}

func isFalsePositiveNestedLoop(path string) bool {
	lower := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	return strings.HasPrefix(lower, "operator/")
}

func isFalsePositiveEval(path, line string) bool {
	lowerPath := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	if strings.HasSuffix(lowerPath, "pdf.js") || strings.Contains(lowerPath, "/vendor/") ||
		strings.Contains(lowerPath, "node_modules/") || strings.HasSuffix(lowerPath, ".min.js") {
		return true
	}
	trimmed := strings.TrimSpace(line)
	// PyTorch/TensorFlow/JAX: model.eval() is inference mode, not dynamic code execution.
	if regexp.MustCompile(`\.eval\s*\(\s*\)`).MatchString(trimmed) {
		return true
	}
	// Comments/docstrings mentioning eval (including negation) are not code execution.
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") ||
		strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") ||
		strings.HasPrefix(trimmed, `"""`) || strings.HasPrefix(trimmed, "'''") ||
		(strings.Contains(lower, "without") && strings.Contains(lower, "eval")) ||
		strings.Contains(lower, "dynamic code execution") {
		return true
	}
	// Common library false positives (sourcemaps / safe wrappers).
	if strings.Contains(lower, "/*#__pure__*/") {
		return true
	}
	// ast.literal_eval is not arbitrary code execution.
	if strings.Contains(trimmed, "ast.literal_eval") || strings.Contains(trimmed, "literal_eval(") {
		return true
	}
	return false
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
	if assessHardcodedSecret(path, line).Skip {
		return true
	}
	return false
}

func isFalsePositiveSQLConcat(path, line string) bool {
	trimmed := strings.TrimSpace(line)
	// Safe pattern: append a constant SQL fragment that only adds placeholders (?, $1).
	if safeSQLConcatSuffix.MatchString(trimmed) {
		return true
	}
	// Store layer: parameterized IN (?) lists built from placeholder slices only.
	lowerPath := strings.ToLower(path)
	// Store batch loader builds IN (?) lists from placeholder slices only.
	if strings.HasSuffix(lowerPath, "findings_batch_sqlite.go") {
		return true
	}
	if strings.Contains(lowerPath, "store/") || strings.Contains(lowerPath, "/store/") {
		if strings.Contains(trimmed, "fmt.Sprintf") {
			return true
		}
		if strings.Contains(trimmed, "strings.Join(placeholders") && strings.Contains(trimmed, "IN (") {
			return true
		}
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
	// Field access such as db.Query == nil is not SQL string building.
	if strings.Contains(trimmed, ".Query") && strings.Contains(trimmed, "==") {
		return true
	}
	return false
}

func isFalsePositiveCmdExec(path, line string) bool {
	lower := strings.ToLower(path)
	if strings.Contains(lower, "analyzers/static.go") || strings.Contains(lower, "analyzers/static_test.go") {
		return true
	}
	// Heuristic rules describing exec patterns, not executing commands.
	if strings.Contains(line, "Pattern:") || strings.Contains(line, "regexp.MustCompile") {
		return true
	}
	// Safe wrappers: exec.Command with static args only (no dynamic fmt.Sprintf on same line).
	if strings.Contains(line, "exec.CommandContext") && !strings.Contains(line, "fmt.Sprintf") && !strings.Contains(line, "+") {
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

var privateCIDRExamplePattern = regexp.MustCompile(`\b(?:10|172\.(?:1[6-9]|2\d|3[01])|192\.168)\.\d{1,3}\.\d{1,3}(?:\.\d{1,3})?/\d{1,2}\b`)

func isInfraReferenceExampleContext(path, line string) bool {
	trimmed := strings.TrimSpace(line)
	lower := strings.ToLower(line)
	if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
		return true
	}
	if strings.Contains(lower, `write("#`) || strings.Contains(lower, `write(' #`) || strings.Contains(lower, "example") {
		return true
	}
	if privateCIDRExamplePattern.MatchString(line) && (strings.Contains(lower, "example") || strings.Contains(lower, "sample") || strings.HasPrefix(trimmed, "#")) {
		return true
	}
	norm := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	if strings.HasSuffix(norm, ".md") || strings.HasSuffix(norm, ".rst") {
		return true
	}
	if !strings.Contains(norm, "/") && strings.HasSuffix(norm, ".md") {
		return true
	}
	return false
}

func isFalsePositiveInternalInfraRef(path, line string) bool {
	if isInfraReferenceExampleContext(path, line) {
		return true
	}
	norm := strings.ReplaceAll(path, "\\", "/")
	lower := strings.ToLower(norm)
	switch {
	case lower == "preinstall/url.go":
		// Blocked-host catalog for SSRF prevention, not embedded infra endpoints.
		return strings.Contains(line, "blockedHost") || strings.Contains(line, "localhost") ||
			strings.Contains(line, "loopback")
	case lower == "readme.md", lower == "quick_setup.md", lower == "deployment.md":
		// Product setup docs reference homelab endpoints by design.
		return true
	case lower == "preinstall/audit_failure.go":
		// Failure-stage classifier references blocked hosts (localhost, private), not embedded endpoints.
		return strings.Contains(line, "localhost") || strings.Contains(line, "private") ||
			strings.Contains(line, "ClassifyFailureStage")
	case strings.HasPrefix(lower, "containers/"):
		// Registry/image reference classification detects localhost and private registries — not embedded infra.
		return strings.Contains(line, "localhost") || strings.Contains(line, "127.0.0.1") ||
			strings.Contains(line, "PrivateRegistry") || strings.Contains(line, "private")
	case strings.HasSuffix(lower, ".example"):
		return true
	}
	return false
}

func isFalsePositiveHTTPClientPerCall(path, line string) bool {
	norm := strings.ReplaceAll(path, "\\", "/")
	lower := strings.ToLower(norm)
	if strings.HasPrefix(lower, "health/") {
		return true
	}
	// Shared client factories configure timeouts/transport once; not hot-path per-call clients.
	if strings.HasSuffix(lower, "client.go") || strings.HasSuffix(lower, "/httpclient.go") || strings.HasSuffix(lower, "notify/http.go") {
		if strings.Contains(line, "http.Client{") || strings.Contains(line, "func New") {
			return true
		}
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
	return RunStaticAnalysisWithProfile(files, enableSecurity, enableQuality, profile.RepoProfile{})
}

// RunStaticAnalysisWithProfile applies homelab/infra calibration when repo profile is provided.
func RunStaticAnalysisWithProfile(files []FileContent, enableSecurity, enableQuality bool, repoProfile profile.RepoProfile) []models.CandidateFinding {
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
				severity := rule.Severity
				confidence := staticRuleConfidence(rule)
				hypothesis := rule.Title
				evidenceNote := rule.Description
				if rule.ID == "SEC-HARDCODED-SECRET" {
					assessment := assessHardcodedSecret(file.Path, line)
					if assessment.Skip {
						continue
					}
					severity = assessment.Severity
					confidence = assessment.Confidence
					if assessment.Evidence != "" {
						evidenceNote = assessment.Evidence
						hypothesis = rule.Title + " — " + assessment.Evidence
					}
				}
				severity, confidence = profile.HomelabInfraSeverity(rule.ID, severity, confidence, file.Path, line, repoProfile)
				findings = append(findings, models.CandidateFinding{
					ID:         rule.ID,
					Hypothesis: hypothesis,
					Evidence: models.Evidence{
						Code:      strings.TrimSpace(line),
						CallChain: []string{file.Path},
						ASTNode:   evidenceNote,
					},
					Severity:    severity,
					Confidence:  confidence,
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
