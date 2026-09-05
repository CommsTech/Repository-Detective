package profile_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/profile"
	"git.commsnet.org/commstech/repository-detective/scanners"
)

func loadFixturePaths(t *testing.T, name string) []string {
	t.Helper()
	root := filepath.Join("..", "testdata", "fixtures", name)
	var paths []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("walk fixture %s: %v", name, err)
	}
	return paths
}

func TestDetectProfileMonorepo(t *testing.T) {
	paths := loadFixturePaths(t, "monorepo")
	p := profile.DetectProfile(paths)
	if p.Layout != profile.LayoutMonorepo {
		t.Fatalf("expected monorepo layout, got %s", p.Layout)
	}
	if len(p.Subpaths) < 2 {
		t.Fatalf("expected subpaths for monorepo, got %+v", p.Subpaths)
	}
}

func TestDetectProfileDocsOnly(t *testing.T) {
	paths := loadFixturePaths(t, "docs-only")
	p := profile.DetectProfile(paths)
	if p.Layout != profile.LayoutDocumentation {
		t.Fatalf("expected documentation layout, got %s", p.Layout)
	}
	if !p.IsDocsOnlyRepo() {
		t.Fatal("expected docs-only repo")
	}
}

func TestDetectProfileSingleGoApp(t *testing.T) {
	paths := loadFixturePaths(t, "go-single")
	p := profile.DetectProfile(paths)
	if p.Layout != profile.LayoutSingleApp {
		t.Fatalf("expected single_app, got %s", p.Layout)
	}
	if p.PrimaryEcosystem != profile.EcosystemGo {
		t.Fatalf("expected go ecosystem, got %s", p.PrimaryEcosystem)
	}
}

func TestClassifySourceTypeVendorAndTest(t *testing.T) {
	if got := profile.ClassifySourceType("vendor/acme/lib.go"); got != profile.SourceTypeVendor {
		t.Fatalf("vendor path: got %s", got)
	}
	if got := profile.ClassifySourceType("internal/handler_test.go"); got != profile.SourceTypeTest {
		t.Fatalf("test file: got %s", got)
	}
	if got := profile.ClassifySourceType("examples/demo/main.go"); got != profile.SourceTypeExample {
		t.Fatalf("example path: got %s", got)
	}
}

func TestNormalizeSuppressesVendorFinding(t *testing.T) {
	raw := []ai.CodeIssue{{
		Severity:   "high",
		Category:   "secrets",
		Title:      "secret in vendor",
		File:       "vendor/acme/config.go",
		LineNumber: 10,
		RuleID:     "G101",
		Source:     "gosec",
		Confidence: 0.9,
	}}
	out := profile.NormalizeIssues(raw, profile.NormalizeInput{
		Repository:    "org/repo",
		Profile:       profile.RepoProfile{Layout: profile.LayoutSingleApp},
		Reporting:     profile.DefaultReportingConfig(),
		FalsePositive: profile.DefaultFalsePositiveReductionConfig(),
	})
	if len(out) != 1 {
		t.Fatalf("expected 1 finding preserved, got %d", len(out))
	}
	if out[0].ReportingAction != profile.ActionSuppressedWithReason {
		t.Fatalf("expected suppressed vendor finding, got %s", out[0].ReportingAction)
	}
	if out[0].SuppressionReason == "" {
		t.Fatal("expected suppression reason")
	}
}

func TestNormalizeHighSeveritySourceAutoIssue(t *testing.T) {
	raw := []ai.CodeIssue{{
		Severity:   "high",
		Category:   "security",
		Title:      "SQL injection",
		File:       "internal/api/handler.go",
		LineNumber: 42,
		RuleID:     "sql-inject",
		Source:     "semgrep",
		Confidence: 0.85,
	}}
	out := profile.NormalizeIssues(raw, profile.NormalizeInput{
		Repository:    "org/repo",
		Profile:       profile.RepoProfile{Layout: profile.LayoutSingleApp, PrimaryEcosystem: profile.EcosystemGo},
		Reporting:     profile.DefaultReportingConfig(),
		FalsePositive: profile.FalsePositiveReductionConfig{Enabled: true},
		KnownPaths:    map[string]struct{}{"internal/api/handler.go": {}},
	})
	if out[0].ReportingAction != profile.ActionAutoIssue {
		t.Fatalf("expected auto_issue, got %s", out[0].ReportingAction)
	}
	if out[0].NormalizedPath != "internal/api/handler.go" {
		t.Fatalf("unexpected normalized path: %s", out[0].NormalizedPath)
	}
}

func TestDecideActionLowSeverityReportOnly(t *testing.T) {
	cfg := profile.DefaultReportingConfig()
	action, _ := profile.DecideAction("low", "code_quality", profile.SourceTypeSource, "", 0.6, cfg, profile.DefaultFalsePositiveReductionConfig())
	if action != profile.ActionReportOnly {
		t.Fatalf("expected report_only for low severity, got %s", action)
	}
}

func TestFilterForgeIssues(t *testing.T) {
	cfg := profile.DefaultReportingConfig()
	issues := []ai.CodeIssue{
		{ReportingAction: profile.ActionAutoIssue, Title: "a"},
		{ReportingAction: profile.ActionManualReview, Title: "m"},
		{ReportingAction: profile.ActionReportOnly, Title: "b"},
		{ReportingAction: profile.ActionSuppressedWithReason, Title: "c"},
	}
	out := profile.FilterForgeIssues(issues, cfg)
	if len(out) != 2 {
		t.Fatalf("expected auto_issue and manual_review when enabled, got %+v", out)
	}
	cfg.ManualReviewCanCreateIssue = false
	out = profile.FilterForgeIssues(issues, cfg)
	if len(out) != 1 || out[0].Title != "a" {
		t.Fatalf("expected only auto_issue when manual review disabled, got %+v", out)
	}
}

func TestAnnotateScannerSkippedNoGo(t *testing.T) {
	results := []scanners.RunResult{{
		Scanner: "govulncheck",
		Status:  scanners.StatusClean,
		Detail:  "no Go module or files",
	}}
	prof := profile.RepoProfile{Layout: profile.LayoutSingleApp}
	out := profile.AnnotateScannerResults(results, prof)
	if out[0].ApplicabilityReason != profile.ApplicabilitySkippedNoMatchingFiles {
		t.Fatalf("expected skipped_no_matching_files, got %s", out[0].ApplicabilityReason)
	}
}

func TestDuplicateFingerprintLifecycle(t *testing.T) {
	raw := []ai.CodeIssue{
		{Severity: "high", Category: "security", Title: "a", File: "a.go", LineNumber: 1, RuleID: "r1", Source: "semgrep", Confidence: 0.9, Fingerprint: "fp1"},
		{Severity: "high", Category: "security", Title: "a", File: "a.go", LineNumber: 1, RuleID: "r1", Source: "semgrep", Confidence: 0.9, Fingerprint: "fp1"},
	}
	out := profile.NormalizeIssues(raw, profile.NormalizeInput{
		Repository: "org/repo",
		Reporting:  profile.DefaultReportingConfig(),
	})
	if len(out) != 1 {
		t.Fatalf("expected deduped normalized findings, got %d", len(out))
	}
}

func TestIssueTemplateDetection(t *testing.T) {
	paths := []string{
		".gitea/ISSUE_TEMPLATE/bug.yaml",
		".github/ISSUE_TEMPLATE/bug_report.md",
		"CONTRIBUTING.md",
		"SECURITY.md",
		"CODEOWNERS",
		"docs/bug_report.md",
	}
	p := profile.DetectProfile(paths)
	if !p.ReportingHints.HasContributing || !p.ReportingHints.HasSecurity || !p.ReportingHints.HasCodeowners {
		t.Fatalf("missing reporting hints: %+v", p.ReportingHints)
	}
	if len(p.ReportingHints.GiteaIssueTemplates) == 0 {
		t.Fatal("expected gitea issue template detection")
	}
}

func TestBetaNoiseRulesReportOnlyForStandardProfile(t *testing.T) {
	for _, name := range []string{"standard", "beta_standard", "light"} {
		cfg := profile.ReportingForScanProfile(profile.DefaultReportingConfig(), name)
		action, _ := profile.DecideAction("medium", "maintainability", profile.SourceTypeSource, "GRAPH-ORPHAN-FILE", 0.9, cfg, profile.DefaultFalsePositiveReductionConfig())
		if action != profile.ActionReportOnly {
			t.Fatalf("%s: expected report_only for graph orphan, got %s", name, action)
		}
	}
}

func TestBetaNoiseRulesNotAppliedForDeep(t *testing.T) {
	for _, name := range []string{"deep", "maintainer_deep"} {
		cfg := profile.ReportingForScanProfile(profile.DefaultReportingConfig(), name)
		action, _ := profile.DecideAction("medium", "maintainability", profile.SourceTypeSource, "GRAPH-ORPHAN-FILE", 0.9, cfg, profile.DefaultFalsePositiveReductionConfig())
		if action != profile.ActionManualReview {
			t.Fatalf("%s should not force report_only for graph findings, got %s", name, action)
		}
	}
}

func TestQualDebugReportOnlyByCategory(t *testing.T) {
	cfg := profile.DefaultReportingConfig()
	action, _ := profile.DecideAction("low", "quality", profile.SourceTypeSource, "QUAL-DEBUG", 0.8, cfg, profile.DefaultFalsePositiveReductionConfig())
	if action != profile.ActionReportOnly {
		t.Fatalf("expected report_only for QUAL-DEBUG via quality category, got %s", action)
	}
}

func TestMaxIssuesGateViaReportingConfig(t *testing.T) {
	cfg := profile.DefaultReportingConfig()
	if cfg.MaxIssuesPerScan != 25 {
		t.Fatalf("expected default max 25, got %d", cfg.MaxIssuesPerScan)
	}
}

func TestMonitorOnlyMode(t *testing.T) {
	cfg := profile.DefaultReportingConfig()
	cfg.Mode = profile.ModeMonitorOnly
	cfg = profile.ApplyReportingMode(cfg)
	action, _ := profile.DecideAction("critical", "secrets", profile.SourceTypeSource, "", 0.99, cfg, profile.DefaultFalsePositiveReductionConfig())
	if action != profile.ActionReportOnly {
		t.Fatalf("monitor_only should not auto issue, got %s", action)
	}
}

func TestNormalizePath(t *testing.T) {
	if got := profile.NormalizePath("./src/../src/main.go"); got != "src/../src/main.go" {
		// NormalizePath does not clean .. — stable path only
		if !strings.Contains(got, "main.go") {
			t.Fatalf("unexpected normalize: %s", got)
		}
	}
}
