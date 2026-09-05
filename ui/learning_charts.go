package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"git.commsnet.org/commstech/repository-detective/store"
)

type learningChartPayload struct {
	EventLabels        []string  `json:"eventLabels"`
	EventValues        []int     `json:"eventValues"`
	NoisyRuleLabels    []string  `json:"noisyRuleLabels"`
	NoisyRuleFPRates   []float64 `json:"noisyRuleFPRates"`
	NoisyRuleFindings  []int     `json:"noisyRuleFindings"`
	FPRatePct          float64   `json:"fpRatePct"`
	ScannerFailurePct  float64   `json:"scannerFailurePct"`
	EventsTotal        int       `json:"eventsTotal"`
	PendingRecs        int       `json:"pendingRecs"`
	ActiveRules        int       `json:"activeRules"`
	GroupedFindings    int       `json:"groupedFindings"`
}

func buildLearningChartJSON(
	health store.LearningHealthSummary,
	byType map[string]int,
	noisy []store.CalibrationRuleStat,
) string {
	payload := learningChartPayload{
		FPRatePct:         health.AvgFalsePositiveRate * 100,
		ScannerFailurePct: health.ScannerFailureRate * 100,
		EventsTotal:       health.EventsTotal,
		PendingRecs:       health.PendingRecommendations,
		ActiveRules:       health.ActiveRepoRules,
		GroupedFindings:   health.GroupedFindings,
	}

	type pair struct {
		label string
		n     int
	}
	var events []pair
	for k, v := range byType {
		events = append(events, pair{k, v})
	}
	sort.Slice(events, func(i, j int) bool { return events[i].n > events[j].n })
	if len(events) > 8 {
		events = events[:8]
	}
	for _, e := range events {
		payload.EventLabels = append(payload.EventLabels, humanLearningEvent(e.label))
		payload.EventValues = append(payload.EventValues, e.n)
	}

	limit := 10
	if len(noisy) > limit {
		noisy = noisy[:limit]
	}
	for _, r := range noisy {
		label := strings.TrimSpace(r.RuleID)
		if label == "" {
			label = r.Source
		}
		if len(label) > 28 {
			label = label[:28] + "…"
		}
		if r.Source != "" {
			label = r.Source + "/" + label
			if len(label) > 36 {
				label = label[:36] + "…"
			}
		}
		payload.NoisyRuleLabels = append(payload.NoisyRuleLabels, label)
		payload.NoisyRuleFPRates = append(payload.NoisyRuleFPRates, r.FalsePositiveRate*100)
		payload.NoisyRuleFindings = append(payload.NoisyRuleFindings, r.TotalFindings)
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func humanLearningEvent(eventType string) string {
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return "Unknown"
	}
	parts := strings.Split(eventType, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

func rateMeterClass(rate float64) string {
	switch {
	case rate >= 0.4:
		return "danger"
	case rate >= 0.2:
		return "warn"
	default:
		return "success"
	}
}

func rateMeterWidth(rate float64) string {
	pct := rate * 100
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return fmt.Sprintf("%.0f%%", pct)
}
