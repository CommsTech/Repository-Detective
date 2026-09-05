package remediation

import (
	"path"
	"strings"
)

// RepoHints carries lightweight repo signals for test suggestions.
type RepoHints struct {
	HasGoMod       bool
	HasPackageJSON bool
	HasPytest      bool
	HasDockerfile  bool
	HasTerraform   bool
	TestScript     string
}

// SuggestTests returns deterministic validation commands (not executed).
func SuggestTests(ctx FindingContext, hints RepoHints) (tests []string, commands []string) {
	source := strings.ToLower(ctx.Source)
	category := strings.ToLower(ctx.Category)

	if hints.HasGoMod || strings.HasSuffix(strings.ToLower(ctx.FilePath), ".go") {
		if source != "staticcheck" && source != "golangci-lint" {
			tests = append(tests, "Run Go unit tests for affected packages")
			commands = append(commands, "go test ./...")
		}
	}

	if hints.HasPackageJSON {
		tests = append(tests, "Run project JavaScript/TypeScript test suite")
		if hints.TestScript != "" {
			commands = append(commands, hints.TestScript)
		} else {
			commands = append(commands, "npm test")
		}
		commands = append(commands, "npm run typecheck")
	}

	if hints.HasPytest {
		tests = append(tests, "Run Python test suite")
		commands = append(commands, "pytest")
	} else if strings.HasSuffix(strings.ToLower(ctx.FilePath), ".py") {
		tests = append(tests, "Add or run tests covering the affected module")
	}

	switch source {
	case "hadolint", "dockerfile":
		if ctx.FilePath != "" {
			commands = append(commands, "hadolint "+ctx.FilePath)
		} else if hints.HasDockerfile {
			commands = append(commands, "hadolint Dockerfile")
		}
		tests = append(tests, "Re-run hadolint after Dockerfile changes")
	case "checkov":
		commands = append(commands, "checkov -d .")
		tests = append(tests, "Re-run Checkov after IaC changes")
	case "govulncheck", "grype", "trivy":
		tests = append(tests, "Verify dependency lockfiles and re-run vulnerability scan")
		if hints.HasGoMod {
			commands = append(commands, "go test ./...")
		}
	case "staticcheck", "golangci-lint":
		pkg := goPackagePattern(ctx.FilePath)
		tests = append(tests, "Run Go tests and staticcheck for affected package")
		commands = append(commands, "go test "+pkg)
		commands = append(commands, "staticcheck "+pkg)
	case "gosec":
		tests = append(tests, "Add regression test proving unsafe pattern is removed")
		if hints.HasGoMod {
			commands = append(commands, "go test ./...")
		}
	}

	switch category {
	case "test_gap":
		tests = append(tests, "Add tests covering the referenced code path before refactoring")
	case "secret":
		tests = append(tests, "Verify secret is absent from repository and scanning passes")
		commands = append(commands, "Re-run secret scanner on affected paths")
	}

	return uniqueStrings(tests), uniqueStrings(commands)
}

func goPackagePattern(filePath string) string {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return "./..."
	}
	dir := path.Dir(strings.ReplaceAll(filePath, `\`, `/`))
	if dir == "." || dir == "" {
		return "./..."
	}
	return "./" + dir + "/..."
}

// InferRepoHints builds best-effort repo hints from file paths (no filesystem access).
func InferRepoHints(filePaths ...string) RepoHints {
	var hints RepoHints
	for _, p := range filePaths {
		lower := strings.ToLower(p)
		switch {
		case strings.HasSuffix(lower, "go.mod") || strings.HasSuffix(lower, ".go"):
			hints.HasGoMod = true
		case strings.HasSuffix(lower, "package.json"):
			hints.HasPackageJSON = true
		case strings.Contains(lower, "pytest") || strings.HasSuffix(lower, "conftest.py"):
			hints.HasPytest = true
		case strings.HasSuffix(lower, "dockerfile") || strings.Contains(lower, "dockerfile"):
			hints.HasDockerfile = true
		case strings.HasSuffix(lower, ".tf"):
			hints.HasTerraform = true
		}
	}
	return hints
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
