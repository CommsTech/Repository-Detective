package analyzers

import (
	"testing"
)

func TestRunStaticAnalysisSkipsTestFiles(t *testing.T) {
	findings := RunStaticAnalysis([]FileContent{{
		Path:    "handlers/webhook_test.go",
		Content: `secret := "super-secret-token-12345"`,
	}}, true, false)
	if len(findings) != 0 {
		t.Fatalf("expected test files to be skipped, got %d findings", len(findings))
	}
}

func TestRunStaticAnalysisFindsHardcodedSecret(t *testing.T) {
	findings := RunStaticAnalysis([]FileContent{{
		Path:    "config.go",
		Content: `api_key := "super-secret-token-12345"`,
	}}, true, false)

	if len(findings) == 0 {
		t.Fatal("expected static finding")
	}
	if findings[0].AuditorType != "static" {
		t.Fatalf("expected static source, got %s", findings[0].AuditorType)
	}
}

func TestRunStaticAnalysisSkipsEnvAndDeployShell(t *testing.T) {
	findings := RunStaticAnalysis([]FileContent{{
		Path: "deploy.sh",
		Content: `# shellcheck disable=SC1091
set -a && source .env && set +a
local api_key="${REPOSITORY_DETECTIVE_API_KEY:-}"
`,
	}}, true, false)
	if len(findings) != 0 {
		t.Fatalf("expected deploy.sh env reads to be skipped, got %d: %+v", len(findings), findings)
	}
}

func TestRunStaticAnalysisSkipsHTMLDataAPIKey(t *testing.T) {
	findings := RunStaticAnalysis([]FileContent{{
		Path:    "ui/templates/graph.html",
		Content: `<div data-api-key="{{.APIKey}}"></div>`,
	}}, true, false)
	if len(findings) != 0 {
		t.Fatalf("expected template data-api-key to be skipped, got %d", len(findings))
	}
}

func TestRunStaticAnalysisSkipsSafeSQLConcat(t *testing.T) {
	findings := RunStaticAnalysis([]FileContent{{
		Path:    "store/closure_sqlite.go",
		Content: `query := patchAttemptSelect + ` + "` WHERE status = ?`",
	}}, true, false)
	if len(findings) != 0 {
		t.Fatalf("expected safe SQL concat to be skipped, got %d: %+v", len(findings), findings)
	}
}

func TestRunStaticAnalysisQualityDisabled(t *testing.T) {
	findings := RunStaticAnalysis([]FileContent{{
		Path:    "app.js",
		Content: "console.log('debug');",
	}}, true, false)

	if len(findings) != 0 {
		t.Fatalf("expected no quality findings when disabled, got %d", len(findings))
	}
}

func TestIsFalsePositiveHardcodedSecret(t *testing.T) {
	line := `local gitea_token="${REPOSITORY_DETECTIVE_GITEA_TOKEN:-}"`
	if !isFalsePositiveHardcodedSecret("deploy.sh", line) {
		t.Fatal("expected bash env expansion to be false positive")
	}
}

func TestSkipStaticAnalysisPath(t *testing.T) {
	paths := []string{"ui/templates/x.html", "scripts/foo.sh", "vendor/x.go"}
	for _, p := range paths {
		if !skipStaticAnalysisPath(p) {
			t.Fatalf("expected skip %s", p)
		}
	}
	if skipStaticAnalysisPath("handlers/webhook.go") {
		t.Fatal("expected webhook.go to be analyzed")
	}
	if !skipStaticAnalysisPath("benchmark/fixture/secret_hardcoded.go.src") {
		t.Fatal("expected benchmark fixture to be skipped")
	}
}

func TestRunStaticAnalysisSkipsRuleDefinitionLines(t *testing.T) {
	findings := RunStaticAnalysis([]FileContent{{
		Path:    "analyzers/static.go",
		Content: `Pattern:     regexp.MustCompile(` + "`(?i)(exec\\.Command|fmt\\.Sprintf|%s)`" + `),`,
	}}, true, false)
	for _, f := range findings {
		if f.ID == "SEC-CMD-EXEC" {
			t.Fatalf("expected rule definition line to be skipped, got SEC-CMD-EXEC")
		}
	}
}

func TestRunStaticAnalysisSkipsModelEvalMethod(t *testing.T) {
	findings := RunStaticAnalysis([]FileContent{{
		Path:    "src/eagle/core/brain/ai_processor.py",
		Content: "model.eval()\nself.model.eval()",
	}}, true, false)
	for _, f := range findings {
		if f.ID == "SEC-EVAL" {
			t.Fatalf("expected PyTorch model.eval() to be skipped, got SEC-EVAL on %q", f.Evidence.Code)
		}
	}
}

func TestRunStaticAnalysisSkipsEvalMentionsInDocstrings(t *testing.T) {
	findings := RunStaticAnalysis([]FileContent{{
		Path: "helpers.py",
		Content: "def _safe():\n    \"\"\"Evaluate without using eval().\"\"\"\n    return True\n",
	}}, true, false)
	for _, f := range findings {
		if f.ID == "SEC-EVAL" {
			t.Fatalf("expected docstring eval mention to be skipped, got SEC-EVAL on %q", f.Evidence.Code)
		}
	}
}

func TestRunStaticAnalysisSkipsStoreINClauseSprintf(t *testing.T) {
	findings := RunStaticAnalysis([]FileContent{{
		Path: "store/findings_batch_sqlite.go",
		Content: "query := fmt.Sprintf(`SELECT id FROM findings WHERE id IN (%s)`, strings.Join(placeholders, \",\"))",
	}}, true, false)
	for _, f := range findings {
		if f.ID == "SEC-SQL-CONCAT" {
			t.Fatalf("expected store IN-clause fmt.Sprintf to be skipped, got SEC-SQL-CONCAT")
		}
	}
}

func TestRunStaticAnalysisSkipsStoreINClauseJoin(t *testing.T) {
	findings := RunStaticAnalysis([]FileContent{{
		Path: "store/findings_batch_sqlite.go",
		Content: "query := \"SELECT id FROM findings WHERE id IN (\" + strings.Join(placeholders, \",\") + \")\"",
	}}, true, false)
	for _, f := range findings {
		if f.ID == "SEC-SQL-CONCAT" {
			t.Fatalf("expected store IN-clause strings.Join to be skipped, got SEC-SQL-CONCAT")
		}
	}
}

func TestRunStaticAnalysisSkipsDBQueryNilCheck(t *testing.T) {
	findings := RunStaticAnalysis([]FileContent{{
		Path:    "issuelink/backfill.go",
		Content: "if db == nil || db.Query == nil || repositoryID <= 0 {",
	}}, true, false)
	for _, f := range findings {
		if f.ID == "SEC-SQL-CONCAT" {
			t.Fatalf("expected db.Query nil check to be skipped, got SEC-SQL-CONCAT")
		}
	}
}

func TestStaticRuleConfidenceOrdering(t *testing.T) {
	eval := staticRuleConfidence(staticRule{ID: "SEC-EVAL"})
	secret := staticRuleConfidence(staticRule{ID: "SEC-HARDCODED-SECRET"})
	if eval <= secret {
		t.Fatalf("eval confidence should exceed heuristic secret: eval=%v secret=%v", eval, secret)
	}
}

func TestRunStaticAnalysisFindsPipelineFloatingActionRef(t *testing.T) {
	findings := RunStaticAnalysis([]FileContent{{
		Path:    ".gitea/workflows/build.yml",
		Content: "steps:\n  - uses: actions/checkout@v4",
	}}, true, false)

	if len(findings) != 1 {
		t.Fatalf("expected pipeline governance finding, got %d", len(findings))
	}
	if findings[0].ID != "GOV-ACTION-FLOATING-REF" {
		t.Fatalf("expected GOV-ACTION-FLOATING-REF, got %s", findings[0].ID)
	}
}

func TestRunStaticAnalysisSkipsHTTPClientFactory(t *testing.T) {
	findings := RunStaticAnalysis([]FileContent{{
		Path:    "ai/httpclient.go",
		Content: "func NewHTTPClient() *http.Client {\n\treturn &http.Client{Timeout: 30 * time.Second}\n}\n",
	}}, false, true)
	for _, f := range findings {
		if f.ID == "OPT-HTTP-CLIENT-PER-CALL" {
			t.Fatal("client factory should not match OPT-HTTP-CLIENT-PER-CALL")
		}
	}
}

func TestRunStaticAnalysisSkipsHomelabDocsInfraRef(t *testing.T) {
	findings := RunStaticAnalysis([]FileContent{{
		Path:    "README.md",
		Content: "Open http://localhost:8081 for the UI\n",
	}}, true, false)
	for _, f := range findings {
		if f.ID == "REL-INTERNAL-INFRA-REF" {
			t.Fatal("README homelab endpoint refs are expected in product docs")
		}
	}
}

func TestRunStaticAnalysisSkipsContainerRegistryClassifier(t *testing.T) {
	findings := RunStaticAnalysis([]FileContent{{
		Path: "containers/discover.go",
		Content: `if strings.Contains(lower, "localhost") || strings.Contains(lower, "127.0.0.1") {
		private = true
	}`,
	}}, true, false)
	for _, f := range findings {
		if f.ID == "REL-INTERNAL-INFRA-REF" {
			t.Fatal("container registry classifier should not match REL-INTERNAL-INFRA-REF")
		}
	}
}
