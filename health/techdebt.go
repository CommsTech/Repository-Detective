package health

import (
	"regexp"
	"strings"
)

var (
	techDebtMarker = regexp.MustCompile(`(?i)\b(TODO|FIXME|HACK|XXX)\b`)
	techDebtPhrase = regexp.MustCompile(`(?i)\b(temporary|quick fix|workaround)\b`)
	deprecatedMark = regexp.MustCompile(`(?i)\bdeprecated\b`)
)

func runTechDebtChecks(files []FileInput) []Finding {
	var findings []Finding
	for _, file := range files {
		if strings.HasSuffix(file.Path, "_test.go") || strings.Contains(file.Path, "/testdata/") {
			continue
		}
		lines := strings.Split(file.Content, "\n")
		markerCount := 0
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			if isCommentOnly(trimmed) {
				if isTechDebtSourceFile(file.Path) && commentedOutCodeBlockSize(lines, i) >= 12 {
					findings = append(findings, makeFinding(
						"tech_debt", "tech_debt", "HEALTH-COMMENT-BLOCK", "low", 0.88,
						"Large commented-out code block",
						"Large block of commented code may indicate abandoned logic; review for removal.",
						file.Path, i+1, sampleLine(trimmed),
					))
				}
				findings = append(findings, techDebtLineFindings(file.Path, i+1, trimmed, line, &markerCount)...)
				continue
			}
			findings = append(findings, techDebtLineFindings(file.Path, i+1, trimmed, line, &markerCount)...)
		}
	}
	return findings
}

func techDebtLineFindings(path string, lineNum int, trimmed, line string, markerCount *int) []Finding {
	var out []Finding
	if skipTechDebtFalsePositive(path, trimmed, line) {
		return out
	}
	if techDebtMarker.MatchString(line) {
		*markerCount++
		sev := "low"
		if *markerCount >= 3 {
			sev = "medium"
		}
		out = append(out, makeFinding(
			"tech_debt", "tech_debt", "HEALTH-TECH-MARKER", sev, 0.92,
			"Technical debt marker found in code",
			"TODO/FIXME/HACK/XXX marker detected; track or resolve before release.",
			path, lineNum, sampleLine(trimmed),
		))
	}
	if techDebtPhrase.MatchString(line) {
		out = append(out, makeFinding(
			"tech_debt", "tech_debt", "HEALTH-TECH-PHRASE", "low", 0.85,
			"Technical debt marker found in code",
			"Comment suggests a temporary or workaround change; review recommended.",
			path, lineNum, sampleLine(trimmed),
		))
	}
	if deprecatedMark.MatchString(line) {
		out = append(out, makeFinding(
			"tech_debt", "tech_debt", "HEALTH-DEPRECATED", "low", 0.86,
			"Deprecated API or pattern referenced",
			"Code references deprecated behavior; plan migration.",
			path, lineNum, sampleLine(trimmed),
		))
	}
	return out
}

func isTechDebtSourceFile(path string) bool {
	lower := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	switch {
	case strings.HasSuffix(lower, ".go"), strings.HasSuffix(lower, ".py"),
		strings.HasSuffix(lower, ".js"), strings.HasSuffix(lower, ".ts"),
		strings.HasSuffix(lower, ".java"), strings.HasSuffix(lower, ".rb"),
		strings.HasSuffix(lower, ".php"), strings.HasSuffix(lower, ".cs"):
		return true
	default:
		return false
	}
}

// commentedOutCodeBlockSize counts consecutive comments that look like disabled
// source, not godoc or configuration notes.
func commentedOutCodeBlockSize(lines []string, start int) int {
	count := 0
	for i := start; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if t == "" {
			if count > 0 {
				break
			}
			continue
		}
		if !isCommentOnly(t) {
			break
		}
		if looksLikeCommentedOutCode(t) {
			count++
		}
	}
	return count
}

func looksLikeCommentedOutCode(line string) bool {
	body := strings.TrimSpace(line)
	for _, prefix := range []string{"//", "#", "/*", "*"} {
		if strings.HasPrefix(body, prefix) {
			body = strings.TrimSpace(strings.TrimPrefix(body, prefix))
			break
		}
	}
	if body == "" {
		return false
	}
	hints := []string{
		"func ", "if ", "for ", "return ", ":=", "package ", "import ",
		"var ", "const ", "class ", "def ", "{", "}",
	}
	for _, hint := range hints {
		if strings.Contains(body, hint) {
			return true
		}
	}
	return false
}

func isCommentOnly(line string) bool {
	return strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") ||
		strings.HasPrefix(line, "/*") || strings.HasPrefix(line, "*")
}

func sampleLine(line string) string {
	if len(line) > 200 {
		return line[:200] + "..."
	}
	return line
}

func makeFinding(category, source, ruleID, severity string, confidence float64, title, desc, file string, line int, evidence string) Finding {
	return Finding{
		Category: category, Source: source, RuleID: ruleID, Severity: severity,
		Confidence: confidence, Title: title, Description: desc,
		File: file, Line: line, Evidence: evidence,
	}
}

func skipTechDebtFalsePositive(path, trimmed, line string) bool {
	lower := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	lowerLine := strings.ToLower(line)
	if strings.Contains(lowerLine, "deprecated") {
		if strings.Contains(lowerLine, "legacyconfig") || strings.Contains(lowerLine, "prefer repository_detective") ||
			strings.Contains(lowerLine, "query string api key") || strings.Contains(lowerLine, "backward compatibility") ||
			strings.Contains(lowerLine, "supported but deprecated") {
			return true
		}
	}
	if techDebtPhrase.MatchString(line) {
		if strings.Contains(lowerLine, "temporary git") || strings.Contains(lowerLine, "temporary directory") ||
			strings.Contains(lowerLine, "temporary git clone") || strings.Contains(lowerLine, "mkdirtemp") {
			return true
		}
	}
	if strings.HasPrefix(lower, "ai/config.go") && strings.Contains(lowerLine, "deprecated") {
		return true
	}
	if strings.Contains(lowerLine, "deprecated") && (strings.Contains(lowerLine, "logger.") || strings.Contains(lowerLine, "log.")) {
		return true
	}
	if strings.Contains(lowerLine, "query string api key") && strings.Contains(lowerLine, "deprecated") {
		return true
	}
	return false
}
