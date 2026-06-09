package health

import (
	"path/filepath"
	"strings"
)

func runTestGapChecks(allPaths []string, files []FileInput) []Finding {
	var findings []Finding
	goDirs := map[string]bool{}
	goTests := map[string]bool{}
	pyDirs := map[string]bool{}
	pyTests := map[string]bool{}
	hasPackageJSON := false
	hasTestScript := false
	hasTestsDir := false
	for _, f := range files {
		if strings.EqualFold(filepath.Base(f.Path), "package.json") {
			hasPackageJSON = true
			if ScanPackageJSONForTestScript(f.Content) {
				hasTestScript = true
			}
		}
	}

	for _, p := range allPaths {
		lower := strings.ToLower(p)
		if strings.Contains(p, "/testdata/") || strings.Contains(p, "/fixtures/") {
			continue
		}
		if strings.HasSuffix(lower, "package.json") {
			hasPackageJSON = true
			continue
		}
		if strings.Contains(lower, "/tests/") || strings.HasPrefix(lower, "tests/") {
			hasTestsDir = true
		}
		ext := strings.ToLower(filepath.Ext(p))
		dir := filepath.ToSlash(filepath.Dir(p))
		switch ext {
		case ".go":
			if strings.HasSuffix(p, "_test.go") {
				goTests[dir] = true
			} else if !strings.HasSuffix(p, ".pb.go") {
				goDirs[dir] = true
			}
		case ".py":
			base := filepath.Base(p)
			if strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py") {
				pyTests[dir] = true
			} else if !strings.HasPrefix(base, "__") {
				pyDirs[dir] = true
			}
		}
	}

	for dir := range goDirs {
		if dir == "." || strings.HasSuffix(dir, "/vendor") {
			continue
		}
		if !goTests[dir] && !goTests[parentDir(dir)] {
			findings = append(findings, makeFinding(
				"test_gap", "test_gap", "HEALTH-GO-NO-TEST", "medium", 0.82,
				"Potential test coverage gap for package/module",
				"Go package directory has source files but no `_test.go` nearby; add tests for critical paths.",
				filepath.ToSlash(filepath.Join(dir, "…")), 0, dir,
			))
		}
	}
	for dir := range pyDirs {
		if !pyTests[dir] && !pyTests[parentDir(dir)] {
			findings = append(findings, makeFinding(
				"test_gap", "test_gap", "HEALTH-PY-NO-TEST", "medium", 0.8,
				"Potential test coverage gap for package/module",
				"Python module directory lacks nearby test files; consider adding tests.",
				filepath.ToSlash(filepath.Join(dir, "…")), 0, dir,
			))
		}
	}
	if hasPackageJSON && !hasTestScript && !hasTestsDir {
		findings = append(findings, makeFinding(
			"test_gap", "test_gap", "HEALTH-JS-NO-TEST", "low", 0.78,
			"Potential test coverage gap for package/module",
			"package.json present without tests directory or obvious test script; verify test coverage.",
			"package.json", 0, "",
		))
	}
	return findings
}

func parentDir(dir string) string {
	if dir == "." || dir == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Dir(dir))
}

// ScanPackageJSONForTestScript checks package.json content for a test script.
func ScanPackageJSONForTestScript(content string) bool {
	lower := strings.ToLower(content)
	return strings.Contains(lower, `"test"`) || strings.Contains(lower, `"vitest"`) || strings.Contains(lower, `"jest"`)
}
