package benchmark

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/findinglearn"
)

func TestBenchmarkFixtureExpectations(t *testing.T) {
	root := filepath.Join("fixture")
	read := func(name string) string {
		var path string
		if strings.HasSuffix(name, ".txt") || strings.Contains(name, "/") {
			path = filepath.Join(root, name)
		} else {
			path = filepath.Join(root, name+".src")
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		return string(b)
	}

	secret := read("secret_hardcoded.go")
	if !strings.Contains(secret, "sk-benchmark-fake-key") {
		t.Fatal("expected hardcoded secret inject")
	}
	if !regexp.MustCompile(`(?i)(api[_-]?key|secret)`).MatchString(secret) {
		t.Fatal("expected secret-like pattern")
	}

	sql := read("sql_concat.go")
	if !strings.Contains(sql, `"SELECT * FROM users WHERE id = " + userInput`) {
		t.Fatal("expected SQL concat inject")
	}

	mock := read("mock_secret_test.go")
	if !strings.Contains(mock, "sk-test-mock-not-real") {
		t.Fatal("expected mock secret in test file")
	}
	if !strings.Contains(mock, "package fixture_test") {
		t.Fatal("expected test fixture semantics")
	}

	dupA := read("dup_pattern_a.go")
	dupB := read("dup_pattern_b.go")
	hashA := findinglearn.StructuralHash("EVAL", "security", dupA)
	hashB := findinglearn.StructuralHash("EVAL", "security", dupB)
	if hashA != hashB {
		t.Fatalf("expected structural duplicate grouping, got %s vs %s", hashA, hashB)
	}

	reqs := read("requirements.txt")
	if !strings.Contains(reqs, "requests==2.32.3") {
		t.Fatal("expected pinned requests dependency fixture")
	}
	if !strings.Contains(reqs, "urllib3>=2.6.0") {
		t.Fatal("expected urllib3 floor in dependency fixture")
	}

	vendor := read("vendor/minified.js")
	if !strings.Contains(vendor, "function") {
		t.Fatal("expected minified vendor fixture")
	}

	orphan := read("orphan_module.go")
	if !strings.Contains(orphan, "UnusedHelper") {
		t.Fatal("expected orphan module fixture")
	}
}

func TestReachabilityHeuristicsOnFixture(t *testing.T) {
	in := findinglearn.ClassifyPath("benchmark/fixture/mock_secret_test.go")
	if !in.TestOnlyPath {
		t.Fatal("expected test-only classification")
	}
	sev, conf, note := findinglearn.ActionabilityAdjust("medium", 0.8, in)
	if sev != "info" || note == "" {
		t.Fatalf("unexpected adjust: %s %v %q", sev, conf, note)
	}
}
