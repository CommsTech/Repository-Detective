package health

import (
	"regexp"
	"strings"
)

var (
	regexInLoop   = regexp.MustCompile(`(?i)for\s+.*\{[\s\S]{0,400}regexp\.MustCompile`)
	longSleep     = regexp.MustCompile(`time\.Sleep\s*\(\s*[5-9]\d{8,}|\d{3,}\s*\*\s*time\.(Minute|Hour)`)
	readAllFile   = regexp.MustCompile(`(?i)(ioutil\.ReadAll|io\.ReadAll|os\.ReadFile)\s*\(`)
)

func runPerformanceChecks(files []FileInput) []Finding {
	var findings []Finding
	for _, file := range files {
		if strings.HasSuffix(file.Path, "_test.go") || strings.Contains(file.Path, ".test.") {
			continue
		}
		path := strings.ReplaceAll(file.Path, "\\", "/")
		if strings.HasPrefix(path, "analyzers/static") {
			continue
		}
		content := file.Content
		if detectLang(file.Path) == "go" && regexInLoop.MatchString(content) {
			idx := strings.Index(content, "regexp.MustCompile")
			line := lineNumberAt(content, idx)
			findings = append(findings, makeFinding(
				"performance", "performance", "HEALTH-REGEX-IN-LOOP", "low", 0.8,
				"Performance footgun: regex compilation inside loop",
				"Compile regular expressions once outside hot loops.",
				file.Path, line, "regexp.MustCompile inside loop",
			))
		}
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if longSleep.MatchString(line) {
				findings = append(findings, makeFinding(
					"performance", "performance", "HEALTH-LONG-SLEEP", "low", 0.78,
					"Performance footgun: long sleep in production path",
					"Long time.Sleep may block workers; review whether delay is intentional.",
					file.Path, i+1, sampleLine(strings.TrimSpace(line)),
				))
			}
			if readAllFile.MatchString(line) && !strings.Contains(strings.ToLower(line), "// health:ok") && !isExpectedReadAll(path, line) {
				findings = append(findings, makeFinding(
					"performance", "performance", "HEALTH-READ-ALL", "low", 0.76,
					"Performance footgun: read entire file/stream into memory",
					"Reading full files into memory can be expensive for large inputs; stream when possible.",
					file.Path, i+1, sampleLine(strings.TrimSpace(line)),
				))
			}
		}
	}
	return findings
}

func isExpectedReadAll(path, line string) bool {
	lower := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	// HTTP/API transports read bounded response bodies; not bulk file ingestion.
	switch {
	case strings.HasPrefix(lower, "ai/"), strings.HasPrefix(lower, "gitea/"), strings.HasPrefix(lower, "api/runner_handler.go"):
		return strings.Contains(line, "ReadAll") || strings.Contains(line, "ReadFile")
	case strings.HasPrefix(lower, "runner/client.go"):
		return strings.Contains(line, "ReadAll")
	}
	return false
}

func lineNumberAt(content string, idx int) int {
	if idx <= 0 {
		return 1
	}
	return strings.Count(content[:idx], "\n") + 1
}
