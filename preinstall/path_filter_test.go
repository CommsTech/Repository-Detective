package preinstall

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestSkipPreinstallPathBenchmarkFixture(t *testing.T) {
	paths := []string{
		"benchmark/fixture/secret_hardcoded.go.src",
		"vendor/minified.js",
		"pkg/main.go",
	}
	want := []bool{true, true, false}
	for i, p := range paths {
		if skipPreinstallPath(p) != want[i] {
			t.Fatalf("skipPreinstallPath(%q)=%v want %v", p, !want[i], want[i])
		}
	}
}

func TestFilterPreinstallFindingsDropsBenchmark(t *testing.T) {
	in := []store.AuditFinding{
		{FilePath: "benchmark/fixture/dup_pattern_a.go.src", Severity: "critical", Source: "static"},
		{FilePath: "src/main.go", Severity: "high", Source: "semgrep", Confidence: 0.95},
	}
	out := filterPreinstallFindings(in)
	if len(out) != 1 || out[0].FilePath != "src/main.go" {
		t.Fatalf("expected benchmark dropped, got %+v", out)
	}
}

func TestIsEnvFallbackPattern(t *testing.T) {
	if !isEnvFallbackPattern("token", `token := os.Getenv("API_KEY")`) {
		t.Fatal("expected env fallback detection")
	}
}
