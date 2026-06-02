package health

import (
	"path/filepath"
	"regexp"
	"strings"
)

var (
	ignoredError   = regexp.MustCompile(`_\s*=\s*(\w+\([^)]*\)|[\w.]+\([^)]*\))`)
	panicCall      = regexp.MustCompile(`\bpanic\s*\(`)
	logFatal       = regexp.MustCompile(`\b(log\.Fatal|log\.Fatalf|os\.Exit)\s*\(`)
	httpNoTimeout  = regexp.MustCompile(`\bhttp\.(Get|Post|Head|Do)\s*\(`)
	defaultClient  = regexp.MustCompile(`\bhttp\.DefaultClient\b`)
	emptyCatchJS   = regexp.MustCompile(`catch\s*\([^)]*\)\s*\{\s*\}`)
	emptyCatchPy   = regexp.MustCompile(`(?i)except\s*:\s*pass`)
	retryNoSleep   = regexp.MustCompile(`(?i)for\s+.*retry|while\s+.*retry`)
)

func runReliabilityChecks(files []FileInput) []Finding {
	var findings []Finding
	for _, file := range files {
		lang := strings.ToLower(file.Language)
		if lang == "" {
			lang = detectLang(file.Path)
		}
		isTest := strings.HasSuffix(file.Path, "_test.go") || strings.Contains(file.Path, ".test.") || strings.Contains(file.Path, "/tests/")
		isMain := strings.HasSuffix(file.Path, "main.go") || strings.HasSuffix(strings.ToLower(file.Path), "/main.py")

		lines := strings.Split(file.Content, "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || skipReliabilityLine(trimmed) {
				continue
			}
			if lang == "go" {
				if m := ignoredError.FindStringSubmatch(line); m != nil {
					call := m[1]
					if isAllowedIgnoredError(call) {
						continue
					}
					findings = append(findings, makeFinding(
						"reliability", "reliability", "HEALTH-IGNORED-ERROR", "medium", 0.87,
						"Potential reliability issue: ignored error return",
						"Error return value is discarded; failures may go unnoticed.",
						file.Path, i+1, sampleLine(trimmed),
					))
				}
				if !isTest && panicCall.MatchString(line) && !isMain {
					findings = append(findings, makeFinding(
						"reliability", "reliability", "HEALTH-PANIC", "medium", 0.9,
						"Potential reliability issue: panic in library code",
						"panic() outside tests/main can crash the process; prefer returning errors.",
						file.Path, i+1, sampleLine(trimmed),
					))
				}
				if !isMain && logFatal.MatchString(line) && !isTest {
					findings = append(findings, makeFinding(
						"reliability", "reliability", "HEALTH-FATAL-EXIT", "medium", 0.88,
						"Potential reliability issue: fatal exit in library code",
						"log.Fatal or os.Exit in non-main code prevents graceful error handling.",
						file.Path, i+1, sampleLine(trimmed),
					))
				}
				if !isTest && (httpNoTimeout.MatchString(line) || defaultClient.MatchString(line)) {
					findings = append(findings, makeFinding(
						"reliability", "reliability", "HEALTH-HTTP-NO-TIMEOUT", "medium", 0.86,
						"Potential reliability issue: network call without explicit timeout",
						"Use http.Client with context/deadline instead of DefaultClient or bare http.Get.",
						file.Path, i+1, sampleLine(trimmed),
					))
				}
			}
			if lang == "javascript" || lang == "typescript" || lang == "python" {
				if emptyCatchJS.MatchString(line) || emptyCatchPy.MatchString(line) {
					findings = append(findings, makeFinding(
						"reliability", "reliability", "HEALTH-EMPTY-CATCH", "medium", 0.84,
						"Potential reliability issue: broad exception swallowing",
						"Empty catch/except blocks hide failures; log or handle errors explicitly.",
						file.Path, i+1, sampleLine(trimmed),
					))
				}
			}
			if retryNoSleep.MatchString(line) && !strings.Contains(strings.ToLower(line), "sleep") && !strings.Contains(strings.ToLower(line), "backoff") {
				findings = append(findings, makeFinding(
					"reliability", "reliability", "HEALTH-RETRY-NO-BACKOFF", "low", 0.75,
					"Potential reliability issue: retry loop without obvious backoff",
					"Retry loops without delay/backoff can amplify load; review retry policy.",
					file.Path, i+1, sampleLine(trimmed),
				))
			}
		}
	}
	return findings
}

func skipReliabilityLine(trimmed string) bool {
	if strings.HasPrefix(trimmed, "//") {
		return true
	}
	// Heuristic rule text and finding descriptions must not self-match.
	if strings.Contains(trimmed, "makeFinding(") ||
		strings.Contains(trimmed, `"Potential reliability`) ||
		strings.Contains(trimmed, `"panic()`) ||
		strings.Contains(trimmed, `"log.Fatal`) {
		return true
	}
	return false
}

func isAllowedIgnoredError(call string) bool {
	lower := strings.ToLower(call)
	for _, allowed := range []string{
		"close(", "remove(", "removeall(", "sync.", "unlock(", "waitgroup",
		"commentissue", "addlifecyclelabels", "updatefindingstatus", "addlifecycleevent",
		"emit(", "commentandlabel",
	} {
		if strings.Contains(lower, allowed) {
			return true
		}
	}
	return false
}

func detectLang(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js", ".jsx", ".ts", ".tsx":
		return "javascript"
	default:
		return ""
	}
}
