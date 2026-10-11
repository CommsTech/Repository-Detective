package health_test

import (
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/health"
	"git.commsnet.org/commstech/repository-detective/issues"
	"git.commsnet.org/commstech/repository-detective/scanners"
)

func testCfg() health.Config {
	cfg := health.DefaultConfig()
	cfg.MaxFindings = 50
	return cfg
}

func TestTechDebtTODODetected(t *testing.T) {
	findings := health.Run(health.RunInput{Files: []health.FileInput{{
		Path: "main.go", Content: "// TODO: fix this later\nfunc main() {}\n", Language: "go",
	}}}, testCfg(), nil)
	if len(findings) == 0 {
		t.Fatal("expected TODO finding")
	}
	found := false
	for _, f := range findings {
		if f.Category == "tech_debt" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected tech_debt category, got %v", findings[0].Category)
	}
}

func TestTechDebtSkipsVendor(t *testing.T) {
	findings := health.Run(health.RunInput{Files: []health.FileInput{{
		Path: "vendor/lib/main.go", Content: "// TODO: ignored\n", Language: "go",
	}}}, testCfg(), nil)
	if len(findings) != 0 {
		t.Fatal("vendor TODO should be skipped")
	}
}

func TestReliabilityIgnoredError(t *testing.T) {
	findings := health.Run(health.RunInput{Files: []health.FileInput{{
		Path: "svc.go", Content: "func f() { _ = doWork() }\n", Language: "go",
	}}}, testCfg(), nil)
	found := false
	for _, f := range findings {
		if f.RuleID == "HEALTH-IGNORED-ERROR" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected ignored error finding")
	}
}

func TestReliabilitySkipsTestFiles(t *testing.T) {
	findings := health.Run(health.RunInput{Files: []health.FileInput{{
		Path: "svc_test.go", Content: "func TestX(t *testing.T) { _ = doWork() }\n", Language: "go",
	}}}, testCfg(), nil)
	for _, f := range findings {
		if f.RuleID == "HEALTH-IGNORED-ERROR" {
			t.Fatal("ignored errors in _test.go should be skipped")
		}
	}
}

func TestHealthSkipsTestdataFixtures(t *testing.T) {
	findings := health.Run(health.RunInput{
		AllPaths: []string{"testdata/fixtures/go-single/main.go"},
		Files:    []health.FileInput{{Path: "testdata/fixtures/go-single/main.go", Content: "package main\n", Language: "go"}},
	}, testCfg(), nil)
	for _, f := range findings {
		if f.RuleID == "HEALTH-GO-NO-TEST" {
			t.Fatal("testdata fixtures should be skipped for test gap checks")
		}
	}
}

func TestHealthSkipsDocsAndRuleDefinitions(t *testing.T) {
	findings := health.Run(health.RunInput{Files: []health.FileInput{
		{Path: "docs/TODO.md", Content: "// TODO: fix\n// temporary workaround\n", Language: "go"},
		{Path: "health/techdebt.go", Content: "// TODO: rule text\n// deprecated API\n", Language: "go"},
		{Path: "analyzers/static.go", Content: strings.Repeat("func big() {\n", 200) + "}\n", Language: "go"},
	}}, testCfg(), nil)
	if len(findings) != 0 {
		t.Fatalf("expected no findings in docs/health/analyzers rule files, got %d", len(findings))
	}
}

func TestReliabilityHTTPNoTimeout(t *testing.T) {
	findings := health.Run(health.RunInput{Files: []health.FileInput{{
		Path: "client.go", Content: "resp, err := http.Get(url)\n", Language: "go",
	}}}, testCfg(), nil)
	found := false
	for _, f := range findings {
		if f.RuleID == "HEALTH-HTTP-NO-TIMEOUT" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected http.Get finding")
	}
}

func TestLargeFileDetected(t *testing.T) {
	cfg := testCfg()
	cfg.LargeFileLines = 10
	content := strings.Repeat("line\n", 20)
	findings := health.Run(health.RunInput{Files: []health.FileInput{{
		Path: "big.go", Content: content, Language: "go",
	}}}, cfg, nil)
	found := false
	for _, f := range findings {
		if f.RuleID == "HEALTH-LARGE-FILE" {
			found = true
			if f.Severity != "medium" {
				t.Fatalf("expected medium for go file, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Fatal("expected large file finding")
	}
}

func TestLargePythonScriptLowSeverity(t *testing.T) {
	cfg := testCfg()
	cfg.LargeFileLines = 10
	content := strings.Repeat("print('x')\n", 20)
	findings := health.Run(health.RunInput{Files: []health.FileInput{{
		Path: "collector.py", Content: content, Language: "python",
	}}}, cfg, nil)
	for _, f := range findings {
		if f.RuleID == "HEALTH-LARGE-FILE" && f.Severity != "low" {
			t.Fatalf("expected low for large python script, got %s", f.Severity)
		}
	}
}

func TestGoTestGap(t *testing.T) {
	findings := health.Run(health.RunInput{
		AllPaths: []string{"pkg/foo.go"},
		Files:    []health.FileInput{{Path: "pkg/foo.go", Content: "package pkg\n", Language: "go"}},
	}, testCfg(), nil)
	found := false
	for _, f := range findings {
		if f.RuleID == "HEALTH-GO-NO-TEST" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected go test gap")
	}
}

func TestGoPackageWithTestsNotFlagged(t *testing.T) {
	findings := health.Run(health.RunInput{
		AllPaths: []string{"pkg/foo.go", "pkg/foo_test.go"},
		Files: []health.FileInput{
			{Path: "pkg/foo.go", Content: "package pkg\n", Language: "go"},
			{Path: "pkg/foo_test.go", Content: "package pkg\n", Language: "go"},
		},
	}, testCfg(), nil)
	for _, f := range findings {
		if f.RuleID == "HEALTH-GO-NO-TEST" {
			t.Fatal("should not flag package with _test.go")
		}
	}
}

func TestAIRiskDisabledByDefault(t *testing.T) {
	content := "// This function does something\n// This function does another\n// This function does more\n"
	findings := health.Run(health.RunInput{Files: []health.FileInput{{
		Path: "ai.go", Content: content, Language: "go",
	}}}, testCfg(), nil)
	for _, f := range findings {
		if f.Category == "ai_generated_risk" {
			t.Fatal("AI risk should be disabled by default")
		}
	}
}

func TestAIRiskWordingWhenEnabled(t *testing.T) {
	cfg := testCfg()
	cfg.EnableAIRisk = true
	content := "// This function initializes the module\n// This function initializes the handler\nfunc f() {}\n// implement proper error handling\n"
	findings := health.Run(health.RunInput{Files: []health.FileInput{{
		Path: "ai.go", Content: content, Language: "go",
	}}}, cfg, nil)
	found := false
	for _, f := range findings {
		if f.Category != "ai_generated_risk" {
			continue
		}
		found = true
		if strings.Contains(strings.ToLower(f.Title), "ai-written") || strings.Contains(strings.ToLower(f.Title), "generated by ai") {
			t.Fatal("must not claim AI authorship")
		}
		if !strings.Contains(strings.ToLower(f.Title), "possible") {
			t.Fatal("title should use cautious wording")
		}
	}
	if !found {
		t.Fatal("expected ai_generated_risk when enabled with signals")
	}
}

func TestToCandidateFindingsDeterministicSource(t *testing.T) {
	candidates := health.ToCandidateFindings([]health.Finding{{
		Source: "tech_debt", Category: "tech_debt", RuleID: "HEALTH-TECH-MARKER",
		Severity: "low", Confidence: 0.9, Title: "Technical debt marker found in code",
		File: "a.go", Line: 1,
	}})
	if len(candidates) != 1 {
		t.Fatal("expected candidate")
	}
	if !scanners.IsDeterministicSource("tech_debt") {
		t.Fatal("tech_debt should be deterministic")
	}
}

func TestMaxFindingsCap(t *testing.T) {
	cfg := testCfg()
	cfg.MaxFindings = 2
	var files []health.FileInput
	for i := 0; i < 10; i++ {
		files = append(files, health.FileInput{
			Path: "f.go", Content: "// TODO: item\n", Language: "go",
		})
	}
	findings := health.Run(health.RunInput{Files: files}, cfg, nil)
	if len(findings) > 2 {
		t.Fatalf("expected cap 2, got %d", len(findings))
	}
}

func TestNormalizeCategoryForHealth(t *testing.T) {
	if issues.NormalizeCategory("", "reliability") != issues.CategoryReliability {
		t.Fatal("reliability source mapping")
	}
}
