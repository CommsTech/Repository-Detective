package health

import (
	"regexp"
	"strings"
)

var (
	genericComment = regexp.MustCompile(`(?i)^\s*(//|#|/\*)\s*(this function|this method|initialize the|returns the|handles the)\b`)
	hallucTODO     = regexp.MustCompile(`(?i)implement proper error handling`)
	passThroughGo  = regexp.MustCompile(`^func\s+(\w+)\s*\([^)]*\)\s+\w+\s*\{\s*return\s+\w+\(`)
)

func runAIRiskChecks(files []FileInput) []Finding {
	type fileSignals struct {
		generic int
		halluc  int
		pass    int
	}
	byFile := map[string]*fileSignals{}
	repoGeneric := 0

	for _, file := range files {
		sig := &fileSignals{}
		lines := strings.Split(file.Content, "\n")
		for _, line := range lines {
			if genericComment.MatchString(line) {
				sig.generic++
				repoGeneric++
			}
			if hallucTODO.MatchString(line) {
				sig.halluc++
			}
			trimmed := strings.TrimSpace(line)
			if passThroughGo.MatchString(trimmed) {
				sig.pass++
			}
		}
		byFile[file.Path] = sig
	}

	var findings []Finding
	for path, sig := range byFile {
		signalCount := 0
		if sig.generic >= 2 {
			signalCount++
		}
		if sig.halluc >= 1 {
			signalCount++
		}
		if sig.pass >= 2 {
			signalCount++
		}
		if signalCount == 0 {
			continue
		}
		sev := "low"
		conf := 0.65
		if signalCount >= 2 {
			sev = "medium"
			conf = 0.72
		}
		desc := "Possible low-context or AI-generated code risk: review recommended. "
		var parts []string
		if sig.generic >= 2 {
			parts = append(parts, "repeated generic comments")
		}
		if sig.halluc >= 1 {
			parts = append(parts, "generic error-handling TODO markers")
		}
		if sig.pass >= 2 {
			parts = append(parts, "pass-through wrapper functions")
		}
		desc += strings.Join(parts, ", ") + "."
		findings = append(findings, makeFinding(
			"ai_generated_risk", "ai_generated_risk", "HEALTH-AI-RISK-SIGNALS", sev, conf,
			"Possible low-context or AI-generated code risk — review recommended",
			desc,
			path, 1, "",
		))
	}
	if repoGeneric >= 8 && len(findings) == 0 {
		findings = append(findings, makeFinding(
			"ai_generated_risk", "ai_generated_risk", "HEALTH-AI-RISK-GENERIC", "low", 0.62,
			"Possible low-context or AI-generated code risk — review recommended",
			"Repository contains many generic restatement comments; manual review recommended.",
			"(repository-wide)", 0, "",
		))
	}
	return findings
}
