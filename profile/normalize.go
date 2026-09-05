package profile

import (
	"strings"

	"git.commsnet.org/commstech/repository-detective/ai"
)

// NormalizeInput carries context for finding normalization.
type NormalizeInput struct {
	Repository   string
	CommitSHA    string
	ScanID       string
	Profile      RepoProfile
	Reporting    ReportingConfig
	FalsePositive FalsePositiveReductionConfig
	KnownPaths   map[string]struct{}
	ScannerAgreement map[string]int // fingerprint -> scanner count
}

// NormalizeIssues converts raw scanner/AI output into normalized findings with routing metadata.
func NormalizeIssues(raw []ai.CodeIssue, in NormalizeInput) []ai.CodeIssue {
	if len(raw) == 0 {
		return raw
	}
	in.Reporting = ApplyReportingMode(in.Reporting)

	out := make([]ai.CodeIssue, 0, len(raw))
	seen := map[string]int{}

	for _, issue := range raw {
		n := normalizeOne(issue, in)

		// Track multi-scanner agreement
		if in.FalsePositive.Enabled && in.FalsePositive.RaiseConfidenceWhenMultipleScannersAgree {
			if count, ok := in.ScannerAgreement[n.Fingerprint]; ok && count > 1 {
				n.Confidence = raiseConfidence(n.Confidence, 0.1)
				n.FalsePositiveRisk = downgradeRisk(n.FalsePositiveRisk)
			}
		}

		if prev, dup := seen[n.Fingerprint]; dup {
			existing := &out[prev]
			existing.LifecycleState = LifecycleStillPresent
			if existing.File != n.File {
				existing.LifecycleState = LifecycleMovedFile
			} else if existing.LineNumber != n.LineNumber && n.LineNumber > 0 {
				existing.LifecycleState = LifecycleChangedLine
			}
			continue
		}

		seen[n.Fingerprint] = len(out)
		if n.LifecycleState == "" {
			n.LifecycleState = LifecycleFirstSeen
		}
		out = append(out, n)
	}
	return out
}

func normalizeOne(issue ai.CodeIssue, in NormalizeInput) ai.CodeIssue {
	n := issue
	n.NormalizedPath = NormalizePath(issue.File)
	if n.SourceType == "" {
		n.SourceType = ClassifySourceType(issue.File)
	}

	n.RepoProfileSummary = in.Profile.Layout + "/" + in.Profile.PrimaryEcosystem
	n.CommitSHA = in.CommitSHA
	if n.ScanID == "" {
		n.ScanID = in.ScanID
	}

	baseConf := n.Confidence
	if baseConf <= 0 {
		baseConf = defaultConfidenceForSeverity(n.Severity)
	}
	n.Confidence = AdjustConfidence(baseConf, n, in)

	n.FalsePositiveRisk = assessFalsePositiveRisk(n, in)

	action, suppressReason := DecideAction(n.Severity, n.Category, n.SourceType, n.RuleID, n.Confidence, in.Reporting, in.FalsePositive)
	n.ReportingAction = action
	n.SuppressionReason = suppressReason

	if action == ActionSuppressedWithReason {
		n.LifecycleState = LifecycleSuppressed
	}

	if in.FalsePositive.Enabled && in.FalsePositive.RequireFileExists && n.File != "" {
		if in.KnownPaths != nil {
			if _, ok := in.KnownPaths[n.NormalizedPath]; !ok {
				n.Confidence = lowerConfidence(n.Confidence, 0.15)
				n.FalsePositiveRisk = "file path not verified in checkout"
				if n.ReportingAction == ActionAutoIssue {
					n.ReportingAction = ActionManualReview
				}
			}
		}
	}

	if n.Evidence == "" && n.CodeSnippet != "" {
		n.Evidence = n.CodeSnippet
	}
	if n.Remediation == "" {
		n.Remediation = defaultRemediation(n)
	}

	return n
}

func defaultConfidenceForSeverity(severity string) float64 {
	switch normalizeKey(severity) {
	case "critical":
		return 0.9
	case "high":
		return 0.85
	case "medium", "warning":
		return 0.7
	case "low":
		return 0.55
	default:
		return 0.6
	}
}

func assessFalsePositiveRisk(issue ai.CodeIssue, in NormalizeInput) string {
	var risks []string
	switch issue.SourceType {
	case SourceTypeTest:
		risks = append(risks, "test/fixture path")
	case SourceTypeDocs:
		risks = append(risks, "documentation path")
	case SourceTypeExample:
		risks = append(risks, "example/sample path")
	case SourceTypeGenerated:
		risks = append(risks, "generated code")
	case SourceTypeVendor:
		risks = append(risks, "vendor/third-party code")
	}
	if issue.LineNumber <= 0 {
		risks = append(risks, "no reliable line number")
	}
	if issue.RuleID == "" {
		risks = append(risks, "missing rule ID")
	}
	if issue.Confidence < 0.5 {
		risks = append(risks, "low confidence score")
	}
	if len(risks) == 0 {
		return "low"
	}
	return strings.Join(risks, "; ")
}

func defaultRemediation(issue ai.CodeIssue) string {
	cat := normalizeKey(issue.Category)
	switch {
	case strings.Contains(cat, "secret"):
		return "Remove secret, rotate credential, inject via secret manager"
	case strings.Contains(cat, "depend"):
		return "Upgrade dependency and verify lockfiles"
	case strings.Contains(cat, "quality"), strings.Contains(cat, "maintain"):
		return "Apply lint-guided fix with regression test"
	default:
		return "Review affected code and apply minimal targeted fix"
	}
}

func raiseConfidence(c, delta float64) float64 {
	c += delta
	if c > 1 {
		return 1
	}
	return c
}

func lowerConfidence(c, delta float64) float64 {
	c -= delta
	if c < 0.05 {
		return 0.05
	}
	return c
}

func downgradeRisk(current string) string {
	if current == "" || current == "low" {
		return "low"
	}
	return current
}

// FilterForgeIssues returns only findings eligible for automatic Gitea issue creation.
func FilterForgeIssues(issues []ai.CodeIssue, cfg ReportingConfig) []ai.CodeIssue {
	out := make([]ai.CodeIssue, 0, len(issues))
	for _, issue := range issues {
		if IsForgeAction(issue.ReportingAction, cfg) {
			out = append(out, issue)
		}
	}
	return out
}

// BuildScannerAgreement counts findings by rule/file for multi-scanner agreement.
func BuildScannerAgreement(issues []ai.CodeIssue) map[string]int {
	counts := map[string]int{}
	for _, issue := range issues {
		key := normalizeKey(issue.Source) + "|" + NormalizePath(issue.File) + "|" + normalizeKey(issue.RuleID)
		counts[key]++
	}
	return counts
}
