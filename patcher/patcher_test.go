package patcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/remediation"
)

func enabledPRConfig() Config {
	return Config{
		Enabled:                          true,
		BranchPrefix:                     "repository-detective/fix",
		RequireApproval:                  true,
		MaxFilesChanged:                  3,
		MaxDiffLines:                     100,
		ValidationTimeoutSec:             30,
		RequireTests:                     true,
		UseRunnerVerification:            true,
		BlockHighCriticalWithoutOverride: true,
		AllowedSeverities:                []string{"low", "medium"},
	}
}

func connectedRepo() RepoContext {
	return RepoContext{
		FullName: "o/r", CloneURL: "https://git.example.com/o/r.git",
		DefaultBranch: "main", ConnectedRepo: true,
	}
}

func TestHighSeverityBlockedByDefault(t *testing.T) {
	plan := EligiblePlan()
	plan.Severity = "high"
	elig := CheckPREligibility(plan, connectedRepo(), enabledPRConfig())
	if elig.Eligible {
		t.Fatal("high severity should be blocked when block_high_critical is true")
	}
}

func TestEligibleApprovedPlan(t *testing.T) {
	plan := EligiblePlan()
	elig := CheckPREligibility(plan, connectedRepo(), enabledPRConfig())
	if !elig.Eligible {
		t.Fatalf("expected eligible, blocked: %v", elig.BlockedReasons)
	}
}

func TestUnapprovedPlanBlocked(t *testing.T) {
	plan := EligiblePlan()
	plan.Status = remediation.StatusProposed
	elig := CheckPREligibility(plan, connectedRepo(), enabledPRConfig())
	if elig.Eligible {
		t.Fatal("unapproved plan should be blocked")
	}
}

func TestSafeForAutoPRFalseBlocked(t *testing.T) {
	plan := EligiblePlan()
	plan.SafeForAutoPR = false
	elig := CheckPREligibility(plan, connectedRepo(), enabledPRConfig())
	if elig.Eligible {
		t.Fatal("safe_for_auto_pr=false should block")
	}
}

func TestRequiresHumanReviewBlocked(t *testing.T) {
	plan := EligiblePlan()
	plan.RequiresHumanReview = true
	elig := CheckPREligibility(plan, connectedRepo(), enabledPRConfig())
	if elig.Eligible {
		t.Fatal("requires_human_review=true should block")
	}
}

func TestHighRegressionRiskBlocked(t *testing.T) {
	plan := EligiblePlan()
	plan.RegressionRisk = remediation.RiskHigh
	elig := CheckPREligibility(plan, connectedRepo(), enabledPRConfig())
	if elig.Eligible {
		t.Fatal("high regression risk should block")
	}
}

func TestLargeComplexityBlocked(t *testing.T) {
	plan := EligiblePlan()
	plan.FixComplexity = remediation.ComplexityLarge
	elig := CheckPREligibility(plan, connectedRepo(), enabledPRConfig())
	if elig.Eligible {
		t.Fatal("large complexity should block")
	}
}

func TestSecretFindingBlocked(t *testing.T) {
	plan := EligiblePlan()
	plan.Category = "secret"
	elig := CheckPREligibility(plan, connectedRepo(), enabledPRConfig())
	if elig.Eligible {
		t.Fatal("secret category should block")
	}
}

func TestDependencyMajorBlocked(t *testing.T) {
	plan := EligiblePlan()
	plan.Category = "dependency"
	plan.Summary = "major upgrade required"
	elig := CheckPREligibility(plan, connectedRepo(), enabledPRConfig())
	if elig.Eligible {
		t.Fatal("dependency major upgrade should block")
	}
}

func TestGraphFindingBlocked(t *testing.T) {
	plan := EligiblePlan()
	plan.Source = "graph"
	elig := CheckPREligibility(plan, connectedRepo(), enabledPRConfig())
	if elig.Eligible {
		t.Fatal("graph source should block")
	}
}

func TestUnsupportedRuleBlocked(t *testing.T) {
	plan := EligiblePlan()
	plan.RuleID = "SA9999"
	elig := CheckPREligibility(plan, connectedRepo(), enabledPRConfig())
	if elig.Eligible {
		t.Fatal("unsupported patcher should block")
	}
}

func TestHadolintDL3018ApkPinPatch(t *testing.T) {
	dir := t.TempDir()
	df := filepath.Join(dir, "Dockerfile")
	content := "RUN apk add --no-cache ca-certificates tzdata wget su-exec git && \\\n"
	if err := os.WriteFile(df, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := EligiblePlan()
	plan.Source = "hadolint"
	plan.RuleID = "DL3018"
	plan.AffectedFiles = []string{"Dockerfile"}
	result, err := ApplyPatch(plan, dir, 3, 100)
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary == "" {
		t.Fatal("expected summary")
	}
	updated, _ := os.ReadFile(df)
	if !strings.Contains(string(updated), "ca-certificates=*") || !strings.Contains(string(updated), "git=*") {
		t.Fatalf("expected pinned packages, got %q", updated)
	}
	if !strings.HasPrefix(string(updated), "RUN ") {
		t.Fatalf("expected RUN prefix preserved, got %q", updated)
	}
}

func TestHadolintDL3018ApkPinPatchIfBlock(t *testing.T) {
	dir := t.TempDir()
	df := filepath.Join(dir, "Dockerfile")
	content := "RUN if [ \"$INSTALL\" = \"true\" ]; then \\\n      apk add --no-cache git ca-certificates; \\\n    fi\n"
	if err := os.WriteFile(df, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := EligiblePlan()
	plan.Source = "hadolint"
	plan.RuleID = "DL3018"
	plan.AffectedFiles = []string{"Dockerfile"}
	if _, err := ApplyPatch(plan, dir, 3, 100); err != nil {
		t.Fatal(err)
	}
	updated, _ := os.ReadFile(df)
	if !strings.Contains(string(updated), "git=*") || !strings.Contains(string(updated), "ca-certificates=*") {
		t.Fatalf("expected pinned packages, got %q", updated)
	}
	if strings.Contains(string(updated), "ca-certificates;=*") {
		t.Fatalf("semicolon must stay outside pinned package token, got %q", updated)
	}
}

func TestHadolintDL3018ApkPinPatchTargetLineOnly(t *testing.T) {
	dir := t.TempDir()
	df := filepath.Join(dir, "Dockerfile")
	content := strings.Join([]string{
		"FROM alpine",
		"RUN apk add --no-cache git ca-certificates tzdata",
		"RUN apk add --no-cache wget su-exec && \\",
	}, "\n")
	if err := os.WriteFile(df, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := EligiblePlan()
	plan.Source = "hadolint"
	plan.RuleID = "DL3018"
	plan.AffectedFiles = []string{"Dockerfile"}
	plan.TargetLine = 3
	result, err := ApplyPatch(plan, dir, 3, 100)
	if err != nil {
		t.Fatal(err)
	}
	if result.DiffLines != 2 {
		t.Fatalf("expected all apk add lines patched for hadolint validation, got %d", result.DiffLines)
	}
	updated, _ := os.ReadFile(df)
	lines := strings.Split(string(updated), "\n")
	if !strings.Contains(lines[1], "git=*") || !strings.Contains(lines[2], "wget=*") {
		t.Fatalf("expected all apk add lines pinned, got %q", updated)
	}
}

func TestStaticcheckPatchGenerated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	content := `package main
import "fmt"
func main() { _ = fmt.Sprintf("hello") }
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := EligiblePlan()
	plan.AffectedFiles = []string{"main.go"}
	result, err := ApplyPatch(plan, dir, 3, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.FilesChanged) != 1 {
		t.Fatalf("expected 1 file, got %d", len(result.FilesChanged))
	}
	updated, _ := os.ReadFile(path)
	if strings.Contains(string(updated), `fmt.Sprintf("hello")`) {
		t.Fatal("expected fmt.Sprintf removed")
	}
	if strings.Contains(string(updated), `"fmt"`) {
		t.Fatal("expected unused fmt import removed")
	}
}

func TestStaticcheckS1039RemovesGroupedFmtImport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "marker.go")
	content := `package dogfood

import (
	"fmt"
)

func StaticcheckE2EMarker() string {
	return fmt.Sprintf("repository-detective-staticcheck-e2e")
}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := EligiblePlan()
	plan.AffectedFiles = []string{"marker.go"}
	if _, err := ApplyPatch(plan, dir, 3, 100); err != nil {
		t.Fatal(err)
	}
	updated, _ := os.ReadFile(path)
	text := string(updated)
	if strings.Contains(text, "fmt.") || strings.Contains(text, `"fmt"`) {
		t.Fatalf("expected fmt usage and import removed, got %q", text)
	}
	if !strings.Contains(text, `"repository-detective-staticcheck-e2e"`) {
		t.Fatalf("expected string literal, got %q", text)
	}
}

func TestStaticcheckS1039RemovesFmtImportWhenCommentMentionsFmt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "marker.go")
	content := `package dogfood

import "fmt"

// Replace fmt.Sprintf with a plain string literal.
func StaticcheckE2EMarker() string {
	return fmt.Sprintf("repository-detective-staticcheck-e2e")
}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := EligiblePlan()
	plan.AffectedFiles = []string{"marker.go"}
	if _, err := ApplyPatch(plan, dir, 3, 100); err != nil {
		t.Fatal(err)
	}
	updated, _ := os.ReadFile(path)
	text := string(updated)
	if strings.Contains(text, `"fmt"`) {
		t.Fatalf("expected fmt import removed despite comment mentioning fmt.Sprintf, got %q", text)
	}
}

func TestPatchBoundedByMaxDiffLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	var b strings.Builder
	b.WriteString("package main\nimport \"fmt\"\nfunc main() {\n")
	for i := 0; i < 50; i++ {
		b.WriteString(`  _ = fmt.Sprintf("x")
`)
	}
	b.WriteString("}\n")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := EligiblePlan()
	plan.AffectedFiles = []string{"main.go"}
	_, err := ApplyPatch(plan, dir, 3, 5)
	if err == nil {
		t.Fatal("expected max diff lines error")
	}
}

func TestAllowedPackageScopedGoTest(t *testing.T) {
	argv, err := ParseAllowedCommand("go test ./internal/dogfood/...")
	if err != nil || len(argv) != 3 {
		t.Fatalf("unexpected: %v %v", argv, err)
	}
}

func TestAllowedCommandAccepted(t *testing.T) {
	argv, err := ParseAllowedCommand("go test ./...")
	if err != nil || len(argv) != 3 {
		t.Fatalf("unexpected: %v %v", argv, err)
	}
}

func TestShellCommandRejected(t *testing.T) {
	if _, err := ParseAllowedCommand("bash -c 'go test'"); err == nil {
		t.Fatal("shell command should be rejected")
	}
}

func TestInstallCommandRejected(t *testing.T) {
	if _, err := ParseAllowedCommand("npm install"); err == nil {
		t.Fatal("install command should be rejected")
	}
}

func TestNoTokenLeakageInGitSanitizer(t *testing.T) {
	out := redactTokenFromGitOutput("fatal: https://oauth2:secret-token@git.example.com/o/r.git")
	if strings.Contains(out, "secret-token") {
		t.Fatal("token leaked in sanitized output")
	}
}

func TestDefaultBranchPushRejected(t *testing.T) {
	if err := EnsureDefaultBranchNotTarget("main", "main"); err == nil {
		t.Fatal("push to default branch should be rejected")
	}
}

func TestCreatePRPayload(t *testing.T) {
	body := RenderPRBody("summary", "diff", "- `go vet ./...`: **passed**")
	if !strings.Contains(body, "Repository Detective") {
		t.Fatal("expected PR body marker")
	}
}

func TestIssueCommentSafe(t *testing.T) {
	body := RenderIssuePRComment("repository-detective/fix/abc", "https://git.example.com/pr/1", "ok")
	if !strings.Contains(body, "will not merge") {
		t.Fatal("expected no-merge warning")
	}
}

func TestValidationTimeoutRecorded(t *testing.T) {
	orig := execFixedRunner
	defer func() { execFixedRunner = orig }()
	execFixedRunner = func(argv []string, dir string, timeout time.Duration) ([]byte, error) {
		if timeout != 2*time.Second {
			t.Fatalf("unexpected timeout %v", timeout)
		}
		return []byte("ok"), nil
	}
	result := RunAllowedCommand("go vet ./...", t.TempDir(), 2*time.Second)
	if result.Status != "passed" {
		t.Fatalf("expected passed, got %s", result.Status)
	}
}

func TestFailedValidationPreventsEligibleWhenNoPassingCommand(t *testing.T) {
	plan := EligiblePlan()
	plan.ValidationCommands = []string{"make test"}
	elig := CheckPREligibility(plan, connectedRepo(), enabledPRConfig())
	if elig.Eligible {
		t.Fatal("make test should fail validation eligibility")
	}
}

func TestSafeFixRolloutPolicyText(t *testing.T) {
	if !strings.Contains(SafeFixRolloutPolicy, "never auto-merges") {
		t.Fatalf("policy text missing no-merge guarantee: %q", SafeFixRolloutPolicy)
	}
}
