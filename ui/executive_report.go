package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

// ExecutiveRisk is a concise risk item for executive summaries.
type ExecutiveRisk struct {
	Title      string
	Severity   string
	Category   string
	Confidence float64
	Source     string
	ReviewNote string
}

// ExecutiveSummary is a business-ready report header.
type ExecutiveSummary struct {
	RiskPosture          string
	BusinessImpact       string
	TopRisks             []ExecutiveRisk
	Recommendation       string
	RecommendationLabel  string
	ConfidenceLevel      string
	ScopeScanned         string
	ScannerCoverage      string
	KnownLimitations     []string
	RemediationProgress  string
	ChangesSinceLastScan string
	RiskTrend            string
	ActionableCount      int
	ReviewCount          int
	EvidenceBacked       string
}

func buildFleetExecutiveSummary(summary store.DashboardSummary) ExecutiveSummary {
	crit := summary.OpenFindingsBySeverity["critical"]
	high := summary.OpenFindingsBySeverity["high"]
	open := summary.OpenFindingsCount

	out := ExecutiveSummary{
		RiskPosture:     fleetRiskPosture(crit, high, open, summary.UnhealthyReposCount),
		BusinessImpact:  fleetBusinessImpact(crit, high, open, summary.TotalRepositories, summary.UnhealthyReposCount),
		TopRisks:        topRisksFromDashboard(summary),
		ConfidenceLevel: fleetConfidenceLevel(summary),
		ScopeScanned: fmt.Sprintf("%d repositories monitored; %d open findings across fleet backlog",
			summary.TotalRepositories, open),
		ScannerCoverage: scannerCoverageFromSummary(summary),
		KnownLimitations: []string{
			"Executive view summarizes persisted findings; live exploitability requires targeted review.",
			"Low-confidence graph and template-path findings are routed to review, not treated as release blockers.",
			"Scanner binary gaps reduce coverage — see System Health for installed vs configured tools.",
		},
		RemediationProgress: fmt.Sprintf("%d verified resolved · %d remediation candidates · %d PRs opened",
			summary.Lifecycle.ResolvedVerified, summary.Lifecycle.RemediationCandidates, summary.Lifecycle.PROpened),
		EvidenceBacked: fmt.Sprintf("%d findings verified resolved through evidence-based closure",
			summary.Lifecycle.ResolvedVerified),
	}
	out.Recommendation, out.RecommendationLabel = fleetRecommendation(crit, high, summary.UnhealthyReposCount)
	if summary.TotalRepositories == 0 || len(summary.RecentScans) == 0 {
		if out.Recommendation != "blocked" {
			out.Recommendation, out.RecommendationLabel = "caution", "Insufficient scan evidence — investigate before release"
		}
		if crit == 0 {
			out.RiskPosture = "Indeterminate — no recent scans in the control-plane summary"
		}
		out.EvidenceBacked = "No recent scan evidence available; a zero finding count is not a security verdict"
		out.KnownLimitations = append(out.KnownLimitations, "No recent scan records appear in the fleet summary; confirm repository coverage before relying on the findings counts.")
	}
	out.ChangesSinceLastScan = fmt.Sprintf("%d raw issues detected in completed scans (7d window)",
		summary.IssuesDetectedInScans)
	if summary.UnhealthyReposCount > 0 {
		out.RiskTrend = fmt.Sprintf("Elevated — %d repositories have a failed latest scan", summary.UnhealthyReposCount)
	} else if summary.ActionableFailedScansCount > 0 {
		out.RiskTrend = fmt.Sprintf("Watch — %d failed scans in the last 14 days (repos recovered)", summary.ActionableFailedScansCount)
	} else if crit+high > 0 {
		out.RiskTrend = "Stable with open critical/high backlog requiring triage"
	} else {
		out.RiskTrend = "Improving or stable — no critical/high open findings in fleet summary"
	}
	return out
}

func buildRepoExecutiveSummary(
	repo store.Repository,
	severityCounts map[string]int,
	categoryCounts map[string]int,
	confidenceBands map[string]int,
	findings []store.FindingListItem,
	scans []store.Scan,
	latestScannerResults []store.ScannerResultRecord,
	effective store.EffectiveSettings,
	profileName string,
) ExecutiveSummary {
	crit := severityCounts["critical"]
	high := severityCounts["high"]
	totalOpen := 0
	for _, n := range severityCounts {
		totalOpen += n
	}

	out := ExecutiveSummary{
		RiskPosture:     repoRiskPosture(crit, high, totalOpen, scans),
		BusinessImpact:  repoBusinessImpact(repo.FullName, crit, high, totalOpen),
		TopRisks:        topRisksFromFindings(findings, 5),
		ConfidenceLevel: repoConfidenceLevel(confidenceBands, effective.ConfidenceGate),
		ScopeScanned: fmt.Sprintf("Repository %s · policy %s · profile %s · workspace %s",
			repo.FullName, effective.PolicyLevel, profileName, effective.WorkspaceMode),
		ScannerCoverage: repoScannerCoverage(effective),
		KnownLimitations: []string{
			"Report reflects persisted findings from Repository Detective scans, not manual audit sign-off.",
			"Technical findings table below includes filtered rows; executive risks prioritize high-confidence items.",
		},
		ActionableCount: confidenceBands["actionable"],
		ReviewCount:     confidenceBands["review"],
	}
	out.Recommendation, out.RecommendationLabel = repoRecommendation(crit, high, totalOpen, scans)
	if len(scans) == 0 {
		out.ConfidenceLevel = "Not established — no scan evidence recorded"
		out.BusinessImpact = fmt.Sprintf("%s has no recorded scans. Missing findings are not evidence of safe code.", repo.FullName)
	} else if !strings.EqualFold(scans[0].Status, "completed") {
		out.ConfidenceLevel = "Limited — latest scan did not complete successfully"
	}
	if len(scans) > 0 && len(latestScannerResults) == 0 {
		out.RiskPosture = "Indeterminate — latest scan has no recorded scanner outcomes"
		out.KnownLimitations = append(out.KnownLimitations, "Latest scan has no scanner result records. A completed job alone does not establish scan coverage.")
	}
	if len(latestScannerResults) > 0 {
		failedOrSkipped := 0
		for _, result := range latestScannerResults {
			status := strings.ToLower(result.Status)
			if (status != "success" && status != "completed") || result.Error != "" {
				failedOrSkipped++
			}
		}
		if failedOrSkipped > 0 {
			out.KnownLimitations = append(out.KnownLimitations, fmt.Sprintf("%d of %d recorded scanner outcomes are skipped, incomplete, or failed; inspect scan evidence.", failedOrSkipped, len(latestScannerResults)))
		}
	}
	out.Recommendation, out.RecommendationLabel = repoRecommendationWithEvidence(crit, high, totalOpen, scans, latestScannerResults)
	out.RemediationProgress = fmt.Sprintf("%d open findings · %d categories represented",
		totalOpen, len(categoryCounts))
	out.ChangesSinceLastScan = scanDeltaMessage(scans)
	out.RiskTrend = repoRiskTrend(scans, totalOpen)
	if len(scans) == 0 {
		out.EvidenceBacked = "Not verified — no scan records exist"
	} else if !strings.EqualFold(scans[0].Status, "completed") {
		out.EvidenceBacked = "Not verified — latest scan is not complete; inspect scan details"
	} else if len(latestScannerResults) == 0 {
		out.EvidenceBacked = "Not verified — latest completed scan has no recorded scanner results"
	} else {
		out.EvidenceBacked = fmt.Sprintf("%d scanner outcomes recorded for latest scan; inspect skips/errors and commit before approving a release", len(latestScannerResults))
	}
	return out
}

func fleetRiskPosture(crit, high, open, failed int) string {
	switch {
	case crit > 0:
		return "Critical — immediate executive attention required"
	case high > 5 || failed > 0:
		return "Elevated — significant security or reliability exposure"
	case high > 0:
		return "Moderate — high-severity items present but contained"
	case open > 0:
		return "Low–moderate — open findings without critical/high severity"
	default:
		return "Low — no open findings in monitored repositories"
	}
}

func repoRiskPosture(crit, high, open int, scans []store.Scan) string {
	if len(scans) == 0 {
		return "Indeterminate — repository has no recorded scans"
	}
	if !strings.EqualFold(scans[0].Status, "completed") {
		return "Indeterminate — latest scan is not complete; posture may be stale"
	}
	return fleetRiskPosture(crit, high, open, 0)
}

func fleetBusinessImpact(crit, high, open, repos, failed int) string {
	if crit > 0 {
		return fmt.Sprintf("Critical vulnerabilities in the fleet (%d critical open) may block release or increase breach risk across %d repositories.", crit, repos)
	}
	if high > 0 {
		return fmt.Sprintf("%d high-severity open findings may affect compliance posture, customer trust, or release velocity.", high)
	}
	if failed > 0 {
		return fmt.Sprintf("%d repository scans failed — risk visibility is incomplete until scans succeed.", failed)
	}
	if open > 0 {
		return fmt.Sprintf("%d open findings require scheduled remediation; none are critical/high in current summary.", open)
	}
	return "No material open findings detected in monitored repositories."
}

func repoBusinessImpact(fullName string, crit, high, open int) string {
	if crit > 0 || high > 0 {
		return fmt.Sprintf("%s has %d critical and %d high open findings that may affect release readiness or security commitments.", fullName, crit, high)
	}
	if open > 0 {
		return fmt.Sprintf("%s has %d open findings — review recommended before major releases.", fullName, open)
	}
	return fmt.Sprintf("%s shows no open findings in the latest persisted scan data.", fullName)
}

func fleetRecommendation(crit, high, failed int) (string, string) {
	switch {
	case crit > 0 || failed > 2:
		return "blocked", "Do not proceed — release blocked"
	case high > 0 || failed > 0:
		return "caution", "Proceed with caution"
	default:
		return "proceed", "Proceed"
	}
}

func repoRecommendation(crit, high, open int, scans []store.Scan) (string, string) {
	if crit > 0 {
		return "blocked", "Critical finding present — release blocked pending review"
	}
	if len(scans) == 0 {
		return "caution", "No scan evidence — do not approve release on this report"
	}
	latest := scans[0] // ListScansByRepository orders newest first.
	if !strings.EqualFold(latest.Status, "completed") {
		return "caution", "Latest scan incomplete — rescan and review coverage"
	}
	if time.Since(latest.StartedAt) > 14*24*time.Hour {
		return "caution", "Scan is older than 14 days — review freshness"
	}
	return fleetRecommendation(crit, high, 0)
}

// Scanner outcomes are evidence of execution, not proof of complete coverage.
// A verified release decision requires a separate policy/coverage evaluation.
func repoRecommendationWithEvidence(crit, high, open int, scans []store.Scan, results []store.ScannerResultRecord) (string, string) {
	recommendation, label := repoRecommendation(crit, high, open, scans)
	if recommendation == "blocked" || len(scans) == 0 || !strings.EqualFold(scans[0].Status, "completed") {
		return recommendation, label
	}
	if len(results) == 0 {
		return "caution", "Scanner evidence missing — review before release"
	}
	for _, result := range results {
		status := strings.ToLower(result.Status)
		if (status != "success" && status != "completed") || result.Error != "" {
			return "caution", "Scanner coverage incomplete — inspect skipped and failed checks"
		}
	}
	return recommendation, label
}

func fleetConfidenceLevel(summary store.DashboardSummary) string {
	if summary.OpenFindingsCount == 0 {
		return "Limited — zero open findings does not establish complete scanner coverage"
	}
	if summary.Backlog.CriticalOpen+summary.Backlog.HighOpen > 0 {
		return "Medium–high — critical/high counts based on scanner evidence with calibration layer"
	}
	return "Medium — findings present; prioritize high-confidence scanner results"
}

func repoConfidenceLevel(bands map[string]int, gate float64) string {
	actionable := bands["actionable"]
	review := bands["review"]
	if actionable == 0 && review == 0 {
		return "High — no open findings"
	}
	return fmt.Sprintf("Medium — %d actionable (≥%.0f%% confidence gate) · %d needs review",
		actionable, gate*100, review)
}

func topRisksFromDashboard(summary store.DashboardSummary) []ExecutiveRisk {
	var risks []ExecutiveRisk
	for _, sev := range []string{"critical", "high"} {
		if n := summary.OpenFindingsBySeverity[sev]; n > 0 {
			risks = append(risks, ExecutiveRisk{
				Title:      fmt.Sprintf("%d open %s severity findings fleet-wide", n, sev),
				Severity:   sev,
				Category:   "fleet",
				ReviewNote: "See repository risk ranking for ownership",
			})
		}
	}
	if summary.UnhealthyReposCount > 0 {
		risks = append(risks, ExecutiveRisk{
			Title:      fmt.Sprintf("%d repositories with failed latest scan", summary.UnhealthyReposCount),
			Severity:   "high",
			Category:   "reliability",
			ReviewNote: "Incomplete scan coverage reduces confidence",
		})
	} else if summary.ActionableFailedScansCount > 0 {
		risks = append(risks, ExecutiveRisk{
			Title:      fmt.Sprintf("%d failed scans in the last 14 days", summary.ActionableFailedScansCount),
			Severity:   "medium",
			Category:   "reliability",
			ReviewNote: "Repos may have recovered; review recent failure buckets",
		})
	}
	if len(risks) > 5 {
		risks = risks[:5]
	}
	return risks
}

func topRisksFromFindings(findings []store.FindingListItem, max int) []ExecutiveRisk {
	sorted := append([]store.FindingListItem(nil), findings...)
	sort.Slice(sorted, func(i, j int) bool {
		return severityRank(sorted[i].Severity) < severityRank(sorted[j].Severity)
	})
	var out []ExecutiveRisk
	for _, f := range sorted {
		if f.Confidence < 0.5 && severityRank(f.Severity) > 1 {
			continue
		}
		note := ""
		if f.Confidence < 0.7 {
			note = "Needs review — below high-confidence threshold"
		}
		out = append(out, ExecutiveRisk{
			Title:      f.Title,
			Severity:   f.Severity,
			Category:   f.Category,
			Confidence: f.Confidence,
			Source:     f.Source,
			ReviewNote: note,
		})
		if len(out) >= max {
			break
		}
	}
	return out
}

func severityRank(sev string) int {
	switch strings.ToLower(sev) {
	case "critical":
		return 0
	case "high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 4
	}
}

func scannerCoverageFromSummary(summary store.DashboardSummary) string {
	if summary.ScannerToolsMissingCount > 0 {
		return fmt.Sprintf("Partial coverage — %d configured scanner tools missing from runtime (see System Health)",
			summary.ScannerToolsMissingCount)
	}
	if summary.Platform.DegradedCoverage {
		return "Degraded — some scanners failed or are unavailable; see System Health"
	}
	return "Effective scan profile active; per-scanner binary status on System Health page"
}

func repoScannerCoverage(e store.EffectiveSettings) string {
	enabled := 0
	names := []struct {
		on   bool
		name string
	}{
		{e.EnableTrivy, "Trivy"},
		{e.EnableGrype, "Grype"},
		{e.EnableGitleaks, "Gitleaks"},
		{e.EnableSemgrep, "Semgrep"},
		{e.EnableGovulncheck, "govulncheck"},
		{e.EnableGosec, "gosec"},
		{e.EnableStaticcheck, "staticcheck"},
	}
	var list []string
	for _, s := range names {
		if s.on {
			enabled++
			list = append(list, s.name)
		}
	}
	if enabled == 0 {
		return "No scanners enabled in effective profile"
	}
	return fmt.Sprintf("%d scanners enabled: %s", enabled, strings.Join(list, ", "))
}

func scanDeltaMessage(scans []store.Scan) string {
	if len(scans) < 2 {
		return "Insufficient scan history for trend comparison"
	}
	prev := issuesFromScanSummary(scans[1].SummaryJSON)
	curr := issuesFromScanSummary(scans[0].SummaryJSON)
	delta := curr - prev
	switch {
	case delta > 0:
		return fmt.Sprintf("+%d findings vs previous scan (%s → %s)", delta, scans[1].StartedAt.Format("2006-01-02"), scans[0].StartedAt.Format("2006-01-02"))
	case delta < 0:
		return fmt.Sprintf("%d findings resolved since previous scan", -delta)
	default:
		return "Finding count unchanged since previous scan"
	}
}

func repoRiskTrend(scans []store.Scan, open int) string {
	if len(scans) == 0 {
		return "Unknown — no scans recorded"
	}
	last := scans[0]
	if strings.EqualFold(last.Status, "failed") {
		return "Stale — latest scan failed"
	}
	age := time.Since(last.StartedAt)
	if age > 14*24*time.Hour {
		return fmt.Sprintf("Stale — last successful scan %d days ago", int(age.Hours()/24))
	}
	if open == 0 {
		return "Improving — no open findings"
	}
	return scanDeltaMessage(scans)
}

func buildRepoReportChartJSON(severityCounts, categoryCounts map[string]int) string {
	payload := dashboardChartPayload{}
	order := []string{"critical", "high", "medium", "low", "info"}
	for _, sev := range order {
		if n := severityCounts[sev]; n > 0 {
			payload.SeverityLabels = append(payload.SeverityLabels, titleCase(sev))
			payload.SeverityValues = append(payload.SeverityValues, n)
		}
	}
	type catPair struct {
		name  string
		count int
	}
	var cats []catPair
	for cat, n := range categoryCounts {
		cats = append(cats, catPair{cat, n})
	}
	sort.Slice(cats, func(i, j int) bool { return cats[i].count > cats[j].count })
	if len(cats) > 8 {
		cats = cats[:8]
	}
	for _, c := range cats {
		payload.CategoryLabels = append(payload.CategoryLabels, humanCategory(c.name))
		payload.CategoryValues = append(payload.CategoryValues, c.count)
	}
	raw, _ := json.Marshal(payload)
	return string(raw)
}
