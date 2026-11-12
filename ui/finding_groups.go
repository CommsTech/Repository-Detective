package ui

import (
	"fmt"
	"sort"
	"strings"

	"git.commsnet.org/commstech/repository-detective/store"
)

// FindingGroup summarizes repeated low-value informational findings.
type FindingGroup struct {
	RuleID      string
	Source      string
	Severity    string
	Count       int
	Title       string
	SamplePath  string
	Description string
}

// ScanFindingsBreakdown separates actionable findings from grouped informational noise.
type ScanFindingsBreakdown struct {
	TotalRaw        int
	ActionableCount int
	GroupedCount    int
	Grouped         []FindingGroup
	SeverityCounts  map[string]int
	ActionableBySev map[string]int
}

func isGroupableInformationalFinding(f store.Finding) bool {
	sev := strings.ToLower(f.Severity)
	if sev != "info" && sev != "low" {
		return false
	}
	rule := strings.ToUpper(strings.TrimSpace(f.RuleID))
	switch {
	case strings.HasPrefix(rule, "GRAPH-"):
		return true
	case rule == "QUAL-DEBUG":
		return true
	case strings.HasPrefix(rule, "HEALTH-TECH-"):
		return true
	case rule == "HEALTH-COMMENT-BLOCK":
		return true
	}
	return false
}

func groupKey(f store.Finding) string {
	return strings.ToLower(f.Source) + "|" + strings.ToUpper(f.RuleID) + "|" + strings.ToLower(f.Severity)
}

// BuildScanFindingsBreakdown groups repeated graph/quality informational findings for scan summary UI.
func BuildScanFindingsBreakdown(findings map[int64]store.Finding) ScanFindingsBreakdown {
	out := ScanFindingsBreakdown{
		TotalRaw:        len(findings),
		SeverityCounts:  make(map[string]int),
		ActionableBySev: make(map[string]int),
	}
	type acc struct {
		count  int
		sample store.Finding
	}
	buckets := make(map[string]*acc)
	for _, f := range findings {
		sev := strings.ToLower(f.Severity)
		out.SeverityCounts[sev]++
		if isGroupableInformationalFinding(f) {
			k := groupKey(f)
			if buckets[k] == nil {
				buckets[k] = &acc{sample: f}
			}
			buckets[k].count++
			out.GroupedCount++
			continue
		}
		out.ActionableCount++
		out.ActionableBySev[sev]++
	}
	for _, b := range buckets {
		if b.count < 3 {
			out.ActionableCount += b.count
			out.ActionableBySev[strings.ToLower(b.sample.Severity)] += b.count
			out.GroupedCount -= b.count
			continue
		}
		out.Grouped = append(out.Grouped, FindingGroup{
			RuleID:      b.sample.RuleID,
			Source:      b.sample.Source,
			Severity:    b.sample.Severity,
			Count:       b.count,
			Title:       b.sample.Title,
			SamplePath:  b.sample.FilePath,
			Description: groupedFindingDescription(b.sample.RuleID, b.count),
		})
	}
	sort.Slice(out.Grouped, func(i, j int) bool {
		if out.Grouped[i].Count != out.Grouped[j].Count {
			return out.Grouped[i].Count > out.Grouped[j].Count
		}
		return out.Grouped[i].RuleID < out.Grouped[j].RuleID
	})
	return out
}

func groupedFindingDescription(ruleID string, count int) string {
	rule := strings.ToUpper(ruleID)
	switch {
	case strings.HasPrefix(rule, "GRAPH-"):
		return fmt.Sprintf("%d similar graph/map heuristics — informational structure signals, not security defects. Expand findings list for per-file detail.", count)
	case rule == "QUAL-DEBUG":
		return fmt.Sprintf("%d debug print/log statements — review before production hardening.", count)
	default:
		return fmt.Sprintf("%d similar informational findings — drill down in the findings list for raw detail.", count)
	}
}
