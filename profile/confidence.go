package profile

import (
	"strings"

	"git.commsnet.org/commstech/repository-detective/ai"
)

// AdjustConfidence applies false-positive reduction heuristics to confidence score.
func AdjustConfidence(base float64, issue ai.CodeIssue, in NormalizeInput) float64 {
	conf := base
	fp := in.FalsePositive

	if !fp.Enabled {
		return clampConfidence(conf)
	}

	switch issue.SourceType {
	case SourceTypeTest:
		if fp.SuppressTestFixtures {
			conf = lowerConfidence(conf, 0.2)
		}
	case SourceTypeDocs, SourceTypeExample:
		if fp.SuppressDocsExamples {
			conf = lowerConfidence(conf, 0.25)
		}
	case SourceTypeGenerated:
		if fp.SuppressGenerated {
			conf = lowerConfidence(conf, 0.35)
		}
	case SourceTypeVendor:
		if fp.SuppressVendor {
			conf = lowerConfidence(conf, 0.3)
		}
	}

	if issue.LineNumber <= 0 && fp.RequireLineMatch {
		conf = lowerConfidence(conf, 0.1)
	}

	cat := normalizeKey(issue.Category)
	if fp.LowerConfidenceForDevDependencies && (cat == "dependency" || issue.SourceType == SourceTypeDependency) {
		if isDevDependencyHint(issue) {
			conf = lowerConfidence(conf, 0.15)
		}
	}

	if fp.LowerConfidenceForUnreachableCode {
		if issue.RegressionRisk == "low" && issue.SourceType != SourceTypeSource {
			conf = lowerConfidence(conf, 0.05)
		}
	}

	// Boost production source with evidence
	if issue.SourceType == SourceTypeSource && issue.LineNumber > 0 && issue.RuleID != "" {
		conf = raiseConfidence(conf, 0.05)
	}

	rule := strings.ToUpper(strings.TrimSpace(issue.RuleID))
	if strings.HasPrefix(rule, "GRAPH-") {
		conf = lowerConfidence(conf, 0.12)
		if issue.SourceType == SourceTypeTest {
			conf = lowerConfidence(conf, 0.2)
		}
		switch rule {
		case "GRAPH-ORPHAN-FILE", "GRAPH-DISCONNECTED-PACKAGE", "GRAPH-ORPHAN-FUNCTION":
			conf = lowerConfidence(conf, 0.08)
		}
	}

	return clampConfidence(conf)
}

func isDevDependencyHint(issue ai.CodeIssue) bool {
	pkg := normalizeKey(issue.PackageName)
	if pkg == "" {
		return false
	}
	devHints := []string{"dev", "test", "mock", "eslint", "prettier", "jest", "vitest"}
	for _, h := range devHints {
		if strings.Contains(pkg, h) {
			return true
		}
	}
	return false
}

func clampConfidence(c float64) float64 {
	if c < 0.05 {
		return 0.05
	}
	if c > 1 {
		return 1
	}
	return c
}
