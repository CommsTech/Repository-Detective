package ui

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

// dashboardChartPayload is embedded in the dashboard for Chart.js initialization.
type dashboardChartPayload struct {
	SeverityLabels  []string `json:"severityLabels"`
	SeverityValues  []int    `json:"severityValues"`
	CategoryLabels  []string `json:"categoryLabels"`
	CategoryValues  []int    `json:"categoryValues"`
	ScanTrendLabels           []string `json:"scanTrendLabels"`
	ScanTrendValues           []int    `json:"scanTrendValues"`
	RemediationTrendValues    []int    `json:"remediationTrendValues"`
	PlanTrendValues           []int    `json:"planTrendValues"`
	RepoMapLabels             []string `json:"repoMapLabels"`
	RepoMapValues      []int    `json:"repoMapValues"`
	RepoMapFailed      []bool   `json:"repoMapFailed"`
	RepoMapStackLabels []string `json:"repoMapStackLabels"`
	RepoMapStacks      [][]int  `json:"repoMapStacks"`
	BacklogOpen     int      `json:"backlogOpen"`
	BacklogCritical int      `json:"backlogCritical"`
	BacklogHigh     int      `json:"backlogHigh"`
}

func buildDashboardChartJSON(summary store.DashboardSummary, repos []store.RepositorySummary) string {
	return buildDashboardChartJSONWithStore(context.Background(), nil, summary, repos)
}

func buildDashboardChartJSONWithStore(ctx context.Context, qs store.QueryStore, summary store.DashboardSummary, repos []store.RepositorySummary) string {
	payload := dashboardChartPayload{
		BacklogOpen:     summary.Backlog.OpenUnique,
		BacklogCritical: summary.Backlog.CriticalOpen,
		BacklogHigh:     summary.Backlog.HighOpen,
	}

	order := []string{"critical", "high", "medium", "low", "info"}
	for _, sev := range order {
		if n := summary.OpenFindingsBySeverity[sev]; n > 0 {
			payload.SeverityLabels = append(payload.SeverityLabels, titleCase(sev))
			payload.SeverityValues = append(payload.SeverityValues, n)
		}
	}
	for sev, n := range summary.OpenFindingsBySeverity {
		if n > 0 && !containsString(order, sev) {
			payload.SeverityLabels = append(payload.SeverityLabels, sev)
			payload.SeverityValues = append(payload.SeverityValues, n)
		}
	}

	type catPair struct {
		name  string
		count int
	}
	var cats []catPair
	for cat, n := range summary.OpenFindingsByCategory {
		cats = append(cats, catPair{cat, n})
	}
	sort.Slice(cats, func(i, j int) bool { return cats[i].count > cats[j].count })
	if len(cats) > 10 {
		cats = cats[:10]
	}
	for _, c := range cats {
		payload.CategoryLabels = append(payload.CategoryLabels, humanCategory(c.name))
		payload.CategoryValues = append(payload.CategoryValues, c.count)
	}

	trend := buildScanActivityTrend(ctx, qs, summary.RecentScans, 14)
	for _, t := range trend {
		payload.ScanTrendLabels = append(payload.ScanTrendLabels, t.label)
		payload.ScanTrendValues = append(payload.ScanTrendValues, t.value)
	}
	payload.RemediationTrendValues = dayCountsAligned(trend, buildActivityDayCounts(ctx, qs, 14, countAutoRemediatedByDay))
	payload.PlanTrendValues = dayCountsAligned(trend, buildActivityDayCounts(ctx, qs, 14, countRemediationPlansByDay))

	sort.Slice(repos, func(i, j int) bool {
		return repos[i].OpenFindingsCount > repos[j].OpenFindingsCount
	})
	limit := 12
	if len(repos) > limit {
		repos = repos[:limit]
	}
	repoIDs := make([]int64, 0, len(repos))
	for _, r := range repos {
		repoIDs = append(repoIDs, r.ID)
	}
	var categoryByRepo map[int64]map[string]int
	if qs != nil && ctx != nil && len(repoIDs) > 0 {
		if byRepo, err := qs.OpenFindingsByCategoryForRepositories(ctx, repoIDs); err == nil {
			categoryByRepo = byRepo
		}
	}
	for _, r := range repos {
		short := r.FullName
		if idx := strings.LastIndex(short, "/"); idx >= 0 {
			short = short[idx+1:]
		}
		payload.RepoMapLabels = append(payload.RepoMapLabels, short)
		payload.RepoMapValues = append(payload.RepoMapValues, r.OpenFindingsCount)
		payload.RepoMapFailed = append(payload.RepoMapFailed, strings.EqualFold(r.LastScanStatus, "failed"))
		if cats := categoryByRepo[r.ID]; cats != nil {
			stack := make([]int, len(riskStackOrder))
			for cat, n := range cats {
				stack[riskStackIndex(cat)] += n
			}
			payload.RepoMapStacks = append(payload.RepoMapStacks, stack)
		}
	}
	if len(payload.RepoMapStackLabels) == 0 {
		payload.RepoMapStackLabels = riskStackOrder
	}

	raw, _ := json.Marshal(payload)
	return string(raw)
}

var riskStackOrder = []string{"Security", "Dependency", "Reliability", "Graph", "Quality", "Compliance", "Operations"}

func riskStackIndex(category string) int {
	c := strings.ToLower(strings.TrimSpace(category))
	switch {
	case strings.Contains(c, "secret"), c == "security", strings.Contains(c, "vulner"):
		return 0
	case strings.Contains(c, "depend"):
		return 1
	case strings.Contains(c, "reliab"), strings.Contains(c, "public"):
		return 2
	case strings.Contains(c, "arch"), strings.Contains(c, "maintain"), strings.Contains(c, "graph"):
		return 3
	case strings.Contains(c, "quality"), strings.Contains(c, "test"), strings.Contains(c, "tech"):
		return 4
	case strings.Contains(c, "license"), strings.Contains(c, "pipeline"), strings.Contains(c, "compliance"):
		return 5
	default:
		return 6
	}
}

type trendPoint struct {
	label string
	value int
}

const scanActivityWindowDays = 14

// buildScanActivityTrend prefers SQLite aggregation across the full window.
// Falling back to RecentScans alone caused a regression: only ~10 recent rows
// were counted and (on older builds) issues_found was summed — spiking one day.
func buildScanActivityTrend(ctx context.Context, qs store.QueryStore, recent []store.ScanWithRepo, days int) []trendPoint {
	if days <= 0 {
		days = scanActivityWindowDays
	}
	if qs != nil && ctx != nil {
		since := time.Now().UTC().AddDate(0, 0, -(days - 1))
		if byDay, err := qs.CountCompletedScansByDay(ctx, since); err == nil {
			return scanTrendFromDayCounts(byDay, days)
		}
	}
	return scanTrendFromRecent(recent, days)
}

type dayCountFn func(ctx context.Context, qs store.QueryStore, since time.Time) (map[string]int, error)

func countAutoRemediatedByDay(ctx context.Context, qs store.QueryStore, since time.Time) (map[string]int, error) {
	return qs.CountAutoRemediatedFindingsByDay(ctx, since)
}

func countRemediationPlansByDay(ctx context.Context, qs store.QueryStore, since time.Time) (map[string]int, error) {
	return qs.CountRemediationPlansByDay(ctx, since)
}

func buildActivityDayCounts(ctx context.Context, qs store.QueryStore, days int, fn dayCountFn) map[string]int {
	if qs == nil || ctx == nil || fn == nil {
		return map[string]int{}
	}
	if days <= 0 {
		days = scanActivityWindowDays
	}
	since := time.Now().UTC().AddDate(0, 0, -(days - 1))
	byDay, err := fn(ctx, qs, since)
	if err != nil || byDay == nil {
		return map[string]int{}
	}
	return byDay
}

// dayCountsAligned maps YYYY-MM-DD counts onto the same label order as the scan trend.
func dayCountsAligned(trend []trendPoint, byDay map[string]int) []int {
	now := time.Now().UTC()
	out := make([]int, len(trend))
	start := len(trend) - 1
	for i := range trend {
		key := now.AddDate(0, 0, -(start - i)).Format("2006-01-02")
		out[i] = byDay[key]
	}
	return out
}

func scanTrendFromRecent(scans []store.ScanWithRepo, days int) []trendPoint {
	if days <= 0 {
		days = 14
	}
	byDay := map[string]int{}
	now := time.Now().UTC()
	for i := days - 1; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format("2006-01-02")
		byDay[d] = 0
	}
	for _, s := range scans {
		if !strings.EqualFold(s.Status, "completed") {
			continue
		}
		day := s.StartedAt.UTC().Format("2006-01-02")
		if _, ok := byDay[day]; !ok {
			continue
		}
		byDay[day]++
	}
	return scanTrendFromDayCounts(byDay, days)
}

func scanTrendFromDayCounts(byDay map[string]int, days int) []trendPoint {
	if days <= 0 {
		days = 14
	}
	now := time.Now().UTC()
	out := make([]trendPoint, 0, days)
	for i := days - 1; i >= 0; i-- {
		d := now.AddDate(0, 0, -i)
		key := d.Format("2006-01-02")
		out = append(out, trendPoint{
			label: d.Format("Jan 2"),
			value: byDay[key],
		})
	}
	return out
}

func humanCategory(cat string) string {
	cat = strings.ReplaceAll(cat, "_", " ")
	if cat == "" {
		return "Unknown"
	}
	return strings.ToUpper(cat[:1]) + cat[1:]
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func containsString(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}
