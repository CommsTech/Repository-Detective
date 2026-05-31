package health

import (
	"regexp"
	"strings"
)

var goFuncDecl = regexp.MustCompile(`^func\s+(?:\([^)]*\)\s+)?(\w+)\s*\(([^)]*)\)`)

func runMaintainabilityChecks(files []FileInput, cfg Config) []Finding {
	var findings []Finding
	for _, file := range files {
		lines := strings.Split(file.Content, "\n")
		if len(lines) > cfg.LargeFileLines {
			findings = append(findings, makeFinding(
				"maintainability", "maintainability", "HEALTH-LARGE-FILE", "medium", 0.9,
				"Very large source file",
				"File exceeds configured line threshold; consider splitting modules.",
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
		if paramCount > cfg.MaxFunctionParams {
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
		if funcLines > cfg.LargeFunctionLines {
			findings = append(findings, makeFinding(
				"maintainability", "maintainability", "HEALTH-LARGE-FUNC", "medium", 0.88,
				"Very large function",
				"Function exceeds configured line threshold; consider decomposition.",
				path, i+1, sampleLine(trimmed),
			))
		}
		depth := maxBraceDepth(lines[bodyStart:bodyEnd])
		if depth > cfg.MaxNestingDepth {
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
