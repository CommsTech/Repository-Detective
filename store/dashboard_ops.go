package store

import (
	"strings"
	"time"

	"git.commsnet.org/commstech/bugbot/operator"
)

// FindingBacklogSummary separates deduplicated backlog from raw detector noise.
type FindingBacklogSummary struct {
	OpenUnique           int
	NewLast7Days         int
	RegressionsLast7Days int
	ResolvedVerified     int
	ClosedOther          int
	CriticalOpen         int
	HighOpen             int
	MediumOpen           int
	LowOpen              int
	RawDetectorHits7d    int
	RawInstances7d       int
}

// ScannerPlatformRollup summarizes one scanner's runtime readiness (not a finding).
type ScannerPlatformRollup struct {
	Name            string
	Configured      bool
	Available       bool
	Optional        bool
	Required        bool
	InstallState    string
	Version         string
	VersionDisplay  string
	CoverageImpact  string
	StatusLabel     string
	AffectedScans   int
	AffectedRepos   int
	FailureScans    int
	RecommendedFix  string
}

// ScannerPlatformSummary groups platform warnings separately from repo findings.
type ScannerPlatformSummary struct {
	UniqueMissingTools          int
	UniqueFailedScanners        int
	ConfiguredMissingRuntime    int
	DegradedCoverage            bool
	RawMissingEvents            int
	RawFailureEvents            int
	Rollups                     []ScannerPlatformRollup
}

// ScanFailureBucket groups failed repository scans by coarse reason.
type ScanFailureBucket struct {
	Bucket string
	Label  string
	Count  int
}

// FailedScanBrief is a failed scan row for operator action lists.
type FailedScanBrief struct {
	ScanID       string
	RepoFullName string
	RepositoryID int64
	Error        string
	Bucket       string
	StartedAt    time.Time
}

// RepoAttentionBrief identifies repos needing operator review.
type RepoAttentionBrief struct {
	RepositoryID int64
	FullName     string
	Reason       string
	LastScanAt   *time.Time
}

// ScanHealthSummary powers the scan-health panel.
type ScanHealthSummary struct {
	CompletedScans        int
	FailedScans           int
	ActiveScans           int
	FailureBuckets        []ScanFailureBucket
	RecentFailedScans     []FailedScanBrief
	ReposNeedingAttention []RepoAttentionBrief
}

// RemediationInsight explains remediation pipeline counts.
type RemediationInsight struct {
	Candidates   int
	Summary      string
	Reasons      []string
	SettingsHint string
}

// DashboardAction is an operator-facing next step.
type DashboardAction struct {
	Priority string
	Title    string
	Detail   string
	LinkPath string
	Command  string
}

// ClassifyScanFailure buckets scan.error text for dashboard grouping.
func ClassifyScanFailure(errMsg string) string {
	msg := strings.ToLower(strings.TrimSpace(errMsg))
	switch {
	case msg == "":
		return "unknown"
	case strings.Contains(msg, "clone"), strings.Contains(msg, "auth"), strings.Contains(msg, "401"), strings.Contains(msg, "403"),
		strings.Contains(msg, "repository not found"), strings.Contains(msg, "ref not found"), strings.Contains(msg, "permission"):
		return "clone_auth"
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "deadline"), strings.Contains(msg, "context canceled"), strings.Contains(msg, "timed out"):
		return "timeout"
	case strings.Contains(msg, "prepare"), strings.Contains(msg, "workspace"), strings.Contains(msg, "content not found"):
		return "prepare"
	case strings.Contains(msg, "scanner"), strings.Contains(msg, "binary_missing"), strings.Contains(msg, "parse_failed"):
		return "scanner"
	case strings.Contains(msg, "config"), strings.Contains(msg, "invalid"), strings.Contains(msg, "validation"):
		return "config"
	default:
		return "other"
	}
}

func scanFailureBucketLabel(bucket string) string {
	switch bucket {
	case "clone_auth":
		return "Clone / auth"
	case "timeout":
		return "Timeout"
	case "prepare":
		return "Prepare / workspace"
	case "scanner":
		return "Scanner failure"
	case "config":
		return "Config error"
	case "unknown":
		return "Unknown"
	default:
		return "Other"
	}
}

// BuildRemediationInsight explains why remediation candidates may be zero.
func BuildRemediationInsight(openFindings, candidates int, plannerEnabled, prEnabled bool, remediationPolicy string) RemediationInsight {
	insight := RemediationInsight{Candidates: candidates}
	reasons := []string{}
	if !plannerEnabled {
		reasons = append(reasons, "Remediation planner is disabled globally (remediation_planner_enabled=false).")
	}
	if openFindings == 0 {
		reasons = append(reasons, "No open findings are tracked in the database.")
	}
	if candidates == 0 && plannerEnabled && openFindings > 0 {
		reasons = append(reasons, "No proposed plans marked safe_for_auto_pr — generate plans from finding detail pages.")
		reasons = append(reasons, "Findings may not meet confidence/severity gates or may not be auto-patchable.")
	}
	if remediationPolicy != "" && remediationPolicy != "auto_pr" && remediationPolicy != "plan_only" {
		reasons = append(reasons, "Effective remediation_policy may restrict automatic planning.")
	}
	if !prEnabled {
		reasons = append(reasons, "Remediation PR automation is disabled (remediation_pr_enabled=false).")
	}
	insight.Reasons = reasons
	switch {
	case !plannerEnabled:
		insight.Summary = "Auto-remediation is off — triage and fix findings manually or enable the planner."
		insight.SettingsHint = "Set remediation_planner_enabled=true and restart the service."
	case candidates == 0 && openFindings > 0:
		insight.Summary = "Findings exist but no auto-remediation candidates yet — plans are created per finding."
		insight.SettingsHint = "Open a finding → Generate plan → Approve → Attempt PR (when enabled)."
	case candidates > 0:
		insight.Summary = "Remediation candidates are ready for operator review."
	default:
		insight.Summary = "No remediation work queued."
	}
	return insight
}

// MergeScannerRollups combines DB scan-result rollups with runtime tool probes.
func MergeScannerRollups(dbRollups map[string]scannerDBRollup, tools []operator.ToolStatus) ScannerPlatformSummary {
	seenMissing := map[string]struct{}{}
	seenFailed := map[string]struct{}{}
	out := make([]ScannerPlatformRollup, 0, len(tools))
	var summary ScannerPlatformSummary

	for _, tool := range tools {
		db := dbRollups[tool.Name]
		bypassed := operator.TrivyBypassedByGrype(tool, tools)
		r := ScannerPlatformRollup{
			Name:           tool.Name,
			Configured:     tool.Configured,
			Available:      tool.Available,
			Optional:       tool.IsOptional() || bypassed,
			Required:       tool.IsRequiredInProfile() && !bypassed,
			InstallState:   tool.InstallState(),
			Version:        tool.Version,
			VersionDisplay: tool.VersionDisplay(),
			CoverageImpact: tool.CoverageImpact(),
			AffectedScans:  db.MissingScans + db.FailureScans,
			FailureScans:   db.FailureScans,
		}
		if bypassed {
			r.CoverageImpact = "inactive"
			r.StatusLabel = "bypassed (grype active)"
			r.RecommendedFix = "Dependency scanning uses grype; install trivy only if you need its misconfig/secret scanners."
		}
		if tool.Configured && !tool.Available && !bypassed {
			summary.ConfiguredMissingRuntime++
		}
		if db.MissingScans > 0 {
			r.AffectedRepos = db.MissingRepos
			seenMissing[tool.Name] = struct{}{}
		} else if db.FailureScans > 0 {
			r.AffectedRepos = db.FailureRepos
			seenFailed[tool.Name] = struct{}{}
		}
		summary.RawMissingEvents += db.MissingScans
		summary.RawFailureEvents += db.FailureScans
		if !bypassed {
			r.StatusLabel, r.RecommendedFix = scannerStatusLabel(r, tool)
		}
		out = append(out, r)
	}
	summary.Rollups = out
	summary.UniqueMissingTools = len(seenMissing)
	summary.UniqueFailedScanners = len(seenFailed)
	summary.DegradedCoverage = summary.ConfiguredMissingRuntime > 0
	return summary
}

type scannerDBRollup struct {
	MissingScans  int
	MissingRepos  int
	FailureScans  int
	FailureRepos  int
}

func scannerStatusLabel(r ScannerPlatformRollup, tool operator.ToolStatus) (status, fix string) {
	fix = tool.RemediationHint()
	if !r.Configured {
		return "optional, inactive", fix
	}
	if r.Available {
		if r.FailureScans > 0 {
			return "installed, recent run failures", "Review scan logs for parse/timeouts; re-run scan after fixing scanner output."
		}
		if strings.TrimSpace(r.Version) == "" {
			return "installed, version unknown", fix
		}
		return "installed", fix
	}
	if r.AffectedRepos > 0 {
		return "configured, missing (degraded coverage)", fix
	}
	return "configured, missing (degraded coverage)", fix
}

// BuildDashboardActions assembles prioritized operator next steps.
func BuildDashboardActions(
	criticalOpen, highOpen, failedScans int,
	recentFailed []FailedScanBrief,
	missingConfigured []ScannerPlatformRollup,
	reposAttention []RepoAttentionBrief,
) []DashboardAction {
	var actions []DashboardAction
	if criticalOpen > 0 {
		actions = append(actions, DashboardAction{
			Priority: "critical",
			Title:    "Triage critical findings",
			Detail:   strings.TrimSpace(strings.Join([]string{itoa(criticalOpen) + " critical open findings need immediate review."}, " ")),
			LinkPath: "/findings?severity=critical&status=open",
		})
	}
	if highOpen > 0 {
		actions = append(actions, DashboardAction{
			Priority: "high",
			Title:    "Review high severity backlog",
			Detail:   itoa(highOpen) + " high severity findings are still open.",
			LinkPath: "/findings?severity=high&status=open",
		})
	}
	if failedScans > 0 && len(recentFailed) > 0 {
		f := recentFailed[0]
		actions = append(actions, DashboardAction{
			Priority: "high",
			Title:    "Investigate failed scans",
			Detail:   f.RepoFullName + ": " + truncate(f.Error, 120),
			LinkPath: "/scans/" + f.ScanID,
		})
	}
	for _, s := range missingConfigured {
		if s.Configured && !s.Available {
			actions = append(actions, DashboardAction{
				Priority: "medium",
				Title:    "Restore scanner: " + s.Name,
				Detail:   s.StatusLabel + " — affected " + itoa(s.AffectedRepos) + " repos in recent scans.",
				Command:  "./deploy.sh",
				LinkPath: "/health",
			})
			break
		}
	}
	if len(reposAttention) > 0 {
		r := reposAttention[0]
		actions = append(actions, DashboardAction{
			Priority: "medium",
			Title:    "Repository needs attention: " + r.FullName,
			Detail:   r.Reason,
			LinkPath: "/repos/" + itoa64(r.RepositoryID),
		})
	}
	return actions
}

// ApplyPlatformReadiness merges runtime tool probes into platform summary.
func ApplyPlatformReadiness(summary *DashboardSummary, tools []operator.ToolStatus) {
	if summary.platformRollups == nil {
		summary.platformRollups = map[string]scannerDBRollup{}
	}
	summary.Platform = MergeScannerRollups(summary.platformRollups, tools)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func itoa64(n int64) string {
	return itoa(int(n))
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
