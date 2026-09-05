package health

import (
	"regexp"
	"strings"
)

var goFuncDecl = regexp.MustCompile(`^func\s+(?:\([^)]*\)\s+)?(\w+)\s*\(([^)]*)\)`)

func runMaintainabilityChecks(files []FileInput, cfg Config) []Finding {
	var findings []Finding
	for _, file := range files {
		if strings.HasSuffix(file.Path, "_test.go") || strings.Contains(file.Path, "/testdata/") {
			continue
		}
		if skipMaintainabilityPath(file.Path) {
			continue
		}
		lines := strings.Split(file.Content, "\n")
		if len(lines) > largeFileThreshold(file.Path, cfg) && !skipLargeFileFinding(file.Path) {
			sev, conf, detail := largeFileAssessment(file.Path, len(lines))
			findings = append(findings, makeFinding(
				"maintainability", "maintainability", "HEALTH-LARGE-FILE", sev, conf,
				"Very large source file",
				detail,
				file.Path, 1, "",
			))
		}
		lang := detectLang(file.Path)
		if lang == "go" {
			findings = append(findings, analyzeGoFunctions(file.Path, lines, cfg)...)
		}
		if lang == "javascript" || lang == "typescript" {
			findings = append(findings, analyzeJSFunctions(file.Path, lines, cfg)...)
		}
	}
	return findings
}

func largeFileThreshold(path string, cfg Config) int {
	lower := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	switch lower {
	case "main.go", "ui/handler.go", "analyzers/engine.go":
		return cfg.LargeFileLines + 400
	}
	return cfg.LargeFileLines
}

func maxFunctionParams(path string, cfg Config) int {
	lower := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	extra := 0
	if strings.HasPrefix(lower, "store/") || strings.HasPrefix(lower, "main") {
		extra += 4
	}
	for _, prefix := range []string{"gitea/", "issuelink/", "preinstall/", "remediation/", "runner/", "ui/"} {
		if strings.HasPrefix(lower, prefix) {
			extra += 6
			break
		}
	}
	return cfg.MaxFunctionParams + extra
}

func maxNestingDepth(path string, cfg Config) int {
	lower := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	if strings.HasPrefix(lower, "main") || strings.HasPrefix(lower, "analyzers/static") {
		return cfg.MaxNestingDepth + 2
	}
	return cfg.MaxNestingDepth
}

func skipMaintainabilityPath(path string) bool {
	path = strings.ReplaceAll(path, "\\", "/")
	return strings.HasPrefix(path, "analyzers/static")
}

func largeFunctionThreshold(path string, cfg Config) int {
	lower := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	if strings.HasPrefix(lower, "store/profiles.go") {
		return cfg.LargeFunctionLines + 100
	}
	return cfg.LargeFunctionLines
}

func skipLargeFileFinding(path string) bool {
	lower := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	switch lower {
	case "main.go", "ui/handler.go", "analyzers/engine.go":
		return true
	}
	if strings.Contains(lower, "/vendor/") || strings.Contains(lower, "/dist/") ||
		strings.Contains(lower, "/build/") || strings.HasSuffix(lower, ".min.js") ||
		strings.HasSuffix(lower, ".min.css") {
		return true
	}
	return false
}

func largeFileAssessment(path string, lineCount int) (severity string, confidence float64, detail string) {
	severity = "medium"
	confidence = 0.9
	detail = "File exceeds configured line threshold; consider splitting modules."
	lower := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	switch {
	case strings.HasSuffix(lower, ".ps1"), strings.HasSuffix(lower, ".py"), strings.HasSuffix(lower, ".sh"),
		strings.HasSuffix(lower, ".bash"), strings.HasSuffix(lower, ".pl"), strings.HasSuffix(lower, ".rb"):
		severity = "low"
		confidence = 0.55
		detail = "Large operational script — common for tooling/collectors. Split into modules or add tests if maintainability becomes a problem."
	case strings.HasSuffix(lower, ".go"), strings.HasSuffix(lower, ".java"), strings.HasSuffix(lower, ".ts"):
		if lineCount > 2500 {
			detail = "Very large source file with high maintainability risk — prioritize decomposition."
		}
	}
	return severity, confidence, detail
}

func analyzeGoFunctions(path string, lines []string, cfg Config) []Finding {
	var findings []Finding
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "func ") {
			continue
		}
		if !goFuncDecl.MatchString(trimmed) {
			continue
		}
		m := goFuncDecl.FindStringSubmatch(trimmed)
		params := m[2]
		paramCount := countParams(params)
		if paramCount > maxFunctionParams(path, cfg) {
			findings = append(findings, makeFinding(
				"code_quality", "maintainability", "HEALTH-MANY-PARAMS", "low", 0.85,
				"Function has many parameters",
				"High parameter count reduces readability; consider a struct or options object.",
				path, i + 1, sampleLine(trimmed),
			))
		}
		bodyStart := i + 1
		bodyEnd := findGoFuncEnd(lines, bodyStart)
		funcLines := bodyEnd - bodyStart
		if funcLines > largeFunctionThreshold(path, cfg) {
			findings = append(findings, makeFinding(
				"maintainability", "maintainability", "HEALTH-LARGE-FUNC", "medium", 0.88,
				"Very large function",
				"Function exceeds configured line threshold; consider decomposition.",
				path, i+1, sampleLine(trimmed),
			))
		}
		depth := maxBraceDepth(lines[bodyStart:bodyEnd])
		if depth > maxNestingDepth(path, cfg) {
			findings = append(findings, makeFinding(
				"maintainability", "maintainability", "HEALTH-DEEP-NEST", "medium", 0.84,
				"Deeply nested control flow",
				"Deep nesting increases cognitive load; consider early returns or helpers.",
				path, i+1, sampleLine(trimmed),
			))
		}
	}
	return findings
}

func analyzeJSFunctions(path string, lines []string, cfg Config) []Finding {
	var findings []Finding
	funcStart := regexp.MustCompile(`(?m)^\s*(?:export\s+)?(?:async\s+)?function\s+\w+\s*\(|^\s*(?:const|let|var)\s+\w+\s*=\s*(?:async\s*)?\(`)
	for i, line := range lines {
		if !funcStart.MatchString(line) {
			continue
		}
		bodyEnd := findBraceBlockEnd(lines, i)
		if bodyEnd-i > cfg.LargeFunctionLines {
			findings = append(findings, makeFinding(
				"maintainability", "maintainability", "HEALTH-LARGE-FUNC", "medium", 0.86,
				"Very large function",
				"Function exceeds configured line threshold; consider decomposition.",
				path, i+1, sampleLine(strings.TrimSpace(line)),
			))
		}
	}
	return findings
}

func countParams(params string) int {
	params = strings.TrimSpace(params)
	if params == "" {
		return 0
	}
	return strings.Count(params, ",") + 1
}

func findGoFuncEnd(lines []string, start int) int {
	depth := 0
	started := false
	for i := start; i < len(lines); i++ {
		line := lines[i]
		for _, ch := range line {
			switch ch {
			case '{':
				depth++
				started = true
			case '}':
				depth--
				if started && depth == 0 {
					return i + 1
				}
			}
		}
	}
	return len(lines)
}

func findBraceBlockEnd(lines []string, start int) int {
	depth := 0
	started := false
	for i := start; i < len(lines); i++ {
		for _, ch := range lines[i] {
			switch ch {
			case '{':
				depth++
				started = true
			case '}':
				depth--
				if started && depth == 0 {
					return i + 1
				}
			}
		}
	}
	return len(lines)
}

func maxBraceDepth(lines []string) int {
	depth := 0
	max := 0
	for _, line := range lines {
		for _, ch := range line {
			switch ch {
			case '{':
				depth++
				if depth > max {
					max = depth
				}
			case '}':
				if depth > 0 {
					depth--
				}
			}
		}
	}
	return max
}
