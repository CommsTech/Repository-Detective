package ui

import (
	"fmt"
	"strings"

	"git.commsnet.org/commstech/repository-detective/operator"
	"git.commsnet.org/commstech/repository-detective/store"
)

// HealthToolRow is one scanner availability row with optional report action.
type HealthToolRow struct {
	Tool   operator.ToolStatus
	Report *HealthReportLink
}

// HealthCapabilityRow is one capability row with optional report action.
type HealthCapabilityRow struct {
	Capability CapabilityStatus
	Report     *HealthReportLink
}

// HealthScannerFailureRow is a drill-down row for scanner run failures.
type HealthScannerFailureRow struct {
	Event   store.ScannerFailureEvent
	ScanURL string
	Report  HealthReportLink
}

// HealthFailedScanRow is a drill-down row for failed repository scans.
type HealthFailedScanRow struct {
	Brief   store.FailedScanBrief
	ScanURL string
	Report  HealthReportLink
}

// HealthPageModel powers /ui/health.
type HealthPageModel struct {
	Summary            store.DashboardSummary
	Readiness          *operator.Readiness
	ActiveScans        int
	RunnerTelemetry    operator.RunnerTelemetryView
	ToolRows           []HealthToolRow
	CapabilityRows     []HealthCapabilityRow
	ScannerFailures    []HealthScannerFailureRow
	FailedScans        []HealthFailedScanRow
	ProductIssueBase   string
	HasReportableIssue bool
}

func buildHealthPageModel(
	summary store.DashboardSummary,
	readiness *operator.Readiness,
	active int,
	runner operator.RunnerTelemetryView,
	capabilities []CapabilityStatus,
	failures []store.ScannerFailureEvent,
	basePath, publicURL string,
) HealthPageModel {
	uiBase := healthPublicUIBase(publicURL, basePath)
	version := "unknown"
	if readiness != nil && strings.TrimSpace(readiness.Version) != "" {
		version = readiness.Version
	}

	page := HealthPageModel{
		Summary:          summary,
		Readiness:        readiness,
		ActiveScans:      active,
		RunnerTelemetry:  runner,
		ProductIssueBase: defaultProductIssueBase,
	}

	if readiness != nil {
		for _, tool := range readiness.Tools {
			row := HealthToolRow{Tool: tool}
			if toolNeedsHealthReport(tool) {
				rep := BuildToolHealthReport(tool, version, uiBase)
				row.Report = &rep
				page.HasReportableIssue = true
			}
			page.ToolRows = append(page.ToolRows, row)
		}
	}

	for _, cap := range capabilities {
		row := HealthCapabilityRow{Capability: cap}
		if capabilityNeedsHealthReport(cap) {
			rep := BuildCapabilityHealthReport(cap, version, uiBase)
			row.Report = &rep
			page.HasReportableIssue = true
		}
		page.CapabilityRows = append(page.CapabilityRows, row)
	}

	for _, ev := range failures {
		rep := BuildScannerFailureReport(ev, version, uiBase)
		page.ScannerFailures = append(page.ScannerFailures, HealthScannerFailureRow{
			Event:   ev,
			ScanURL: fmt.Sprintf("%s/scans/%s", strings.TrimRight(basePath, "/"), ev.ScanID),
			Report:  rep,
		})
		page.HasReportableIssue = true
	}

	for _, brief := range summary.ScanHealth.RecentFailedScans {
		rep := BuildFailedScanReport(brief, version, uiBase)
		page.FailedScans = append(page.FailedScans, HealthFailedScanRow{
			Brief:   brief,
			ScanURL: fmt.Sprintf("%s/scans/%s", strings.TrimRight(basePath, "/"), brief.ScanID),
			Report:  rep,
		})
		page.HasReportableIssue = true
	}

	return page
}

func toolNeedsHealthReport(tool operator.ToolStatus) bool {
	switch tool.StatusState {
	case operator.StatusEnabledMissingBinary:
		return true
	}
	if (tool.EnabledInConfig || tool.Configured) && (tool.BinaryInstalled || tool.Available) {
		v := strings.TrimSpace(tool.VersionDisplay())
		return v == "" || v == "unknown"
	}
	return false
}

func capabilityNeedsHealthReport(cap CapabilityStatus) bool {
	switch strings.ToLower(strings.TrimSpace(cap.State)) {
	case "degraded", "unavailable":
		return true
	default:
		return false
	}
}
