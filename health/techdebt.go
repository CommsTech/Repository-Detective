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
				if block := commentedBlockSize(lines, i); block >= 12 {
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

func commentedBlockSize(lines []string, start int) int {
	count := 0
	for i := start; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if t == "" {
			if count > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(t, "//") || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "/*") || strings.HasPrefix(t, "*") {
			count++
			continue
		}
		break
	}
	return count
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
