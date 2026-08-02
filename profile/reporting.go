package profile

import "strings"

// ReportingConfig controls how findings are routed to Gitea vs dashboard.
type ReportingConfig struct {
	Mode                         string            `mapstructure:"mode"`
	DefaultIssueMinSeverity      string            `mapstructure:"default_issue_min_severity"`
	DefaultIssueMinConfidence    string            `mapstructure:"default_issue_min_confidence"`
	CreateIssuesForLow           bool              `mapstructure:"create_issues_for_low"`
	CreateIssuesForTests         bool              `mapstructure:"create_issues_for_tests"`
	CreateIssuesForDocs          bool              `mapstructure:"create_issues_for_docs"`
	CreateIssuesForExamples      bool              `mapstructure:"create_issues_for_examples"`
	CreateIssuesForGenerated     bool              `mapstructure:"create_issues_for_generated"`
	CreateIssuesForVendor        bool              `mapstructure:"create_issues_for_vendor"`
	IncludeRawScannerOutput      bool              `mapstructure:"include_raw_scanner_output_in_issue"`
	MaxIssuesPerScan             int               `mapstructure:"max_issues_per_scan"`
	GroupSimilarFindings         bool              `mapstructure:"group_similar_findings"`
	AllowAllCategories           bool              `mapstructure:"allow_all_categories"`
	PreserveAllFindings          bool              `mapstructure:"preserve_all_findings"`
	DefaultActionBySeverity      map[string]string `mapstructure:"default_action_by_severity"`
	CategoryOverrides            map[string]string `mapstructure:"category_overrides"`
	RuleOverrides                map[string]string `mapstructure:"rule_overrides"`
	SourceTypeOverrides          map[string]string `mapstructure:"source_type_overrides"`
	ManualReviewCanCreateIssue   bool              `mapstructure:"manual_review_can_create_issue"`
	SuppressedFindingsAuditable  bool              `mapstructure:"suppressed_findings_are_auditable"`
}

// FalsePositiveReductionConfig tunes confidence and suppression.
type FalsePositiveReductionConfig struct {
	Enabled                              bool `mapstructure:"enabled"`
	SuppressGenerated                    bool `mapstructure:"suppress_generated"`
	SuppressVendor                       bool `mapstructure:"suppress_vendor"`
	SuppressMinified                     bool `mapstructure:"suppress_minified"`
	SuppressTestFixtures                 bool `mapstructure:"suppress_test_fixtures"`
	SuppressDocsExamples                 bool `mapstructure:"suppress_docs_examples"`
	RequireFileExists                    bool `mapstructure:"require_file_exists"`
	RequireLineMatch                     bool `mapstructure:"require_line_match"`
	LowerConfidenceForDevDependencies    bool `mapstructure:"lower_confidence_for_dev_dependencies"`
	LowerConfidenceForUnreachableCode      bool `mapstructure:"lower_confidence_for_unreachable_code"`
	RaiseConfidenceWhenMultipleScannersAgree bool `mapstructure:"raise_confidence_when_multiple_scanners_agree"`
}

// DefaultReportingConfig returns high-signal defaults from the spec.
func DefaultReportingConfig() ReportingConfig {
	return ReportingConfig{
		Mode:                      ModeHighSignal,
		DefaultIssueMinSeverity:   "high",
		DefaultIssueMinConfidence: "medium",
		CreateIssuesForLow:        false,
		CreateIssuesForTests:      false,
		CreateIssuesForDocs:       false,
		CreateIssuesForExamples:   false,
		CreateIssuesForGenerated:  false,
		CreateIssuesForVendor:     false,
		IncludeRawScannerOutput:   false,
		MaxIssuesPerScan:          25,
		GroupSimilarFindings:      true,
		AllowAllCategories:        true,
		PreserveAllFindings:       true,
		ManualReviewCanCreateIssue: true,
		SuppressedFindingsAuditable: true,
		DefaultActionBySeverity: map[string]string{
			"critical": ActionAutoIssue,
			"high":     ActionAutoIssue,
			"medium":   ActionManualReview,
			"low":      ActionReportOnly,
			"info":     ActionReportOnly,
		},
		CategoryOverrides: map[string]string{
			"secrets":                  ActionAutoIssue,
			"secret":                   ActionAutoIssue,
			"dependency_vulnerability": ActionAutoIssue,
			"dependency":               ActionAutoIssue,
			"license":                  ActionManualReview,
			"code_quality":             ActionReportOnly,
			"quality":                  ActionReportOnly,
			"documentation":            ActionReportOnly,
			"test_gap":                 ActionReportOnly,
			"ai_risk":                  ActionManualReview,
		},
		SourceTypeOverrides: map[string]string{
			SourceTypeSource:     ActionAutoIssue,
			SourceTypeConfig:     ActionManualReview,
			SourceTypeDependency: ActionAutoIssue,
			SourceTypeTest:       ActionReportOnly,
			SourceTypeDocs:       ActionReportOnly,
			SourceTypeExample:    ActionReportOnly,
			SourceTypeGenerated:  ActionSuppressedWithReason,
			SourceTypeVendor:     ActionSuppressedWithReason,
		},
	}
}

// DefaultFalsePositiveReductionConfig returns conservative FP reduction defaults.
func DefaultFalsePositiveReductionConfig() FalsePositiveReductionConfig {
	return FalsePositiveReductionConfig{
		Enabled:                              true,
		SuppressGenerated:                    true,
		SuppressVendor:                       true,
		SuppressMinified:                     true,
		SuppressTestFixtures:                 true,
		SuppressDocsExamples:                 true,
		RequireFileExists:                    true,
		RequireLineMatch:                     false,
		LowerConfidenceForDevDependencies:    true,
		LowerConfidenceForUnreachableCode:    true,
		RaiseConfidenceWhenMultipleScannersAgree: true,
	}
}

// BetaNoiseRuleIDs are graph/debug rules kept on the dashboard but not opened as issues by default.
var BetaNoiseRuleIDs = []string{
	"GRAPH-ORPHAN-FILE",
	"GRAPH-ORPHAN-FUNCTION",
	"GRAPH-DISCONNECTED-PACKAGE",
	"GRAPH-SUSPICIOUS-ISLAND",
	"QUAL-DEBUG",
}

// BetaNoiseRuleOverrides returns report-only actions for beta calibration noise rules.
func BetaNoiseRuleOverrides() map[string]string {
	out := make(map[string]string, len(BetaNoiseRuleIDs))
	for _, id := range BetaNoiseRuleIDs {
		out[normalizeKey(id)] = ActionReportOnly
	}
	return out
}

// ProfileAllowsBetaNoiseSuppression reports whether graph/debug rules should stay report-only.
// Deep (and legacy maintainer_deep) keep graph findings actionable; Light/Standard calm noise.
func ProfileAllowsBetaNoiseSuppression(scanProfile string) bool {
	switch normalizeKey(scanProfile) {
	case "deep", "maintainer_deep", "custom", "":
		return false
	default:
		return true
	}
}

// ReportingForScanProfile layers profile-specific rule overrides on global reporting config.
func ReportingForScanProfile(base ReportingConfig, scanProfile string) ReportingConfig {
	if !ProfileAllowsBetaNoiseSuppression(scanProfile) {
		return base
	}
	cfg := base
	if cfg.RuleOverrides == nil {
		cfg.RuleOverrides = map[string]string{}
	} else {
		merged := make(map[string]string, len(cfg.RuleOverrides)+len(BetaNoiseRuleIDs))
		for k, v := range cfg.RuleOverrides {
			merged[k] = v
		}
		cfg.RuleOverrides = merged
	}
	for k, v := range BetaNoiseRuleOverrides() {
		cfg.RuleOverrides[k] = v
	}
	return cfg
}

// ApplyReportingMode adjusts config for named reporting modes.
func ApplyReportingMode(cfg ReportingConfig) ReportingConfig {
	switch cfg.Mode {
	case ModeMonitorOnly:
		for k := range cfg.DefaultActionBySeverity {
			cfg.DefaultActionBySeverity[k] = ActionReportOnly
		}
	case ModeStrict:
		cfg.DefaultIssueMinSeverity = "medium"
		cfg.DefaultActionBySeverity["medium"] = ActionAutoIssue
	case ModeCompliance:
		cfg.CreateIssuesForLow = true
		cfg.DefaultIssueMinSeverity = "low"
		for k := range cfg.DefaultActionBySeverity {
			if k != "info" {
				cfg.DefaultActionBySeverity[k] = ActionAutoIssue
			}
		}
		cfg.SourceTypeOverrides[SourceTypeTest] = ActionManualReview
		cfg.SourceTypeOverrides[SourceTypeDocs] = ActionManualReview
		cfg.SourceTypeOverrides[SourceTypeExample] = ActionManualReview
	}
	return cfg
}

// DecideAction picks the reporting outcome for a normalized finding.
func DecideAction(severity, category, sourceType, ruleID string, confidence float64, cfg ReportingConfig, fp FalsePositiveReductionConfig) (action string, suppressionReason string) {
	cfg = ApplyReportingMode(cfg)

	if cfg.Mode == ModeMonitorOnly {
		return ActionReportOnly, "monitor_only reporting mode"
	}

	if ruleID != "" {
		if ruleAction, ok := cfg.RuleOverrides[normalizeKey(ruleID)]; ok && ruleAction != "" {
			if ruleAction == ActionSuppressedWithReason {
				return ruleAction, "rule policy: " + ruleID
			}
			return ruleAction, ""
		}
	}

	// Source type overrides first (unless mode allows explicit create flags)
	if action, reason := actionFromSourceType(sourceType, cfg, fp); action != "" {
		if action == ActionSuppressedWithReason {
			return action, reason
		}
		if action != ActionAutoIssue || passesSeverityConfidenceGate(severity, confidence, cfg) {
			return action, reason
		}
	}

	if catAction, ok := cfg.CategoryOverrides[normalizeKey(category)]; ok && catAction != "" {
		if catAction == ActionSuppressedWithReason {
			return catAction, "category policy: " + category
		}
		return catAction, ""
	}

	action = cfg.DefaultActionBySeverity[normalizeKey(severity)]
	if action == "" {
		action = ActionReportOnly
	}

	if action == ActionAutoIssue && !passesSeverityConfidenceGate(severity, confidence, cfg) {
		if confidenceRank(confidence) >= confidenceRank(parseConfidence(cfg.DefaultIssueMinConfidence)) {
			return ActionManualReview, "below severity gate but sufficient confidence"
		}
		return ActionReportOnly, "below severity/confidence gate"
	}

	return action, ""
}

func actionFromSourceType(sourceType string, cfg ReportingConfig, fp FalsePositiveReductionConfig) (string, string) {
	if !fp.Enabled {
		if a, ok := cfg.SourceTypeOverrides[normalizeKey(sourceType)]; ok {
			return a, ""
		}
		return "", ""
	}

	switch sourceType {
	case SourceTypeGenerated:
		if fp.SuppressGenerated && !cfg.CreateIssuesForGenerated {
			return ActionSuppressedWithReason, "generated code path"
		}
	case SourceTypeVendor:
		if fp.SuppressVendor && !cfg.CreateIssuesForVendor {
			return ActionSuppressedWithReason, "vendor/third-party path"
		}
	case SourceTypeTest:
		if fp.SuppressTestFixtures && !cfg.CreateIssuesForTests {
			return ActionReportOnly, "test/fixture path"
		}
	case SourceTypeDocs:
		if fp.SuppressDocsExamples && !cfg.CreateIssuesForDocs {
			return ActionReportOnly, "documentation path"
		}
	case SourceTypeExample:
		if fp.SuppressDocsExamples && !cfg.CreateIssuesForExamples {
			return ActionReportOnly, "example/sample path"
		}
	}

	if a, ok := cfg.SourceTypeOverrides[normalizeKey(sourceType)]; ok {
		return a, ""
	}
	return "", ""
}

func passesSeverityConfidenceGate(severity string, confidence float64, cfg ReportingConfig) bool {
	if severityRank(severity) < severityRank(cfg.DefaultIssueMinSeverity) {
		if normalizeKey(severity) == "low" && cfg.CreateIssuesForLow {
			return confidenceRank(confidence) >= confidenceRank(parseConfidence(cfg.DefaultIssueMinConfidence))
		}
		return false
	}
	return confidenceRank(confidence) >= confidenceRank(parseConfidence(cfg.DefaultIssueMinConfidence))
}

func severityRank(severity string) int {
	switch normalizeKey(severity) {
	case "critical", "crit":
		return 5
	case "high", "error":
		return 4
	case "medium", "warning", "warn":
		return 3
	case "low":
		return 2
	case "info", "informational", "note":
		return 1
	default:
		return 0
	}
}

func confidenceRank(conf float64) int {
	switch {
	case conf >= 0.85:
		return 4
	case conf >= 0.7:
		return 3
	case conf >= 0.5:
		return 2
	case conf > 0:
		return 1
	default:
		return 0
	}
}

func parseConfidence(label string) float64 {
	switch normalizeKey(label) {
	case "high":
		return 0.85
	case "medium":
		return 0.7
	case "low":
		return 0.5
	default:
		return 0.7
	}
}

func normalizeKey(s string) string {
	return stringsToLowerTrim(s)
}

func stringsToLowerTrim(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

// IsForgeAction reports whether the action should create/update Gitea issues.
func IsForgeAction(action string, cfg ReportingConfig) bool {
	switch action {
	case ActionAutoIssue:
		return true
	case ActionManualReview:
		return cfg.ManualReviewCanCreateIssue
	default:
		return false
	}
}

// ShouldShowInDashboard reports whether a finding remains visible in dashboard views.
func ShouldShowInDashboard(action string, cfg ReportingConfig) bool {
	if cfg.PreserveAllFindings {
		return true
	}
	return action != ActionSuppressedWithReason && action != ActionDisabledByPolicy
}
