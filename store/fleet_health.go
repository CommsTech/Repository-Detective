package store

import (
	"context"
	"strconv"
	"time"
)

// FleetRepoAuditRow is one repository row in a fleet scheduler/filing audit.
type FleetRepoAuditRow struct {
	RepositoryID       int64
	FullName           string
	ForgeType          string
	ScanEnabled        bool
	ScheduleEnabled    bool
	ScheduleCron       string
	FilingEnabled      bool
	ReportOnlyEnforced bool
	LastScanAt         *time.Time
	LastScanTrigger    string
	LastScanCommit     string
	LastWebhookAt      *time.Time
	StaleScan          bool
	StaleHours         float64
	OpenFindings       int
	NoMappedFindings   int
	MappedForgeIssues  int
	ScheduleEligible   bool
	ScheduleSkipReason string
}

// FleetHealthSummary is fleet-wide scan/schedule/filing health for operators.
type FleetHealthSummary struct {
	SchedulerGloballyEnabled bool
	TotalTracked             int
	ScanEnabledCount         int
	ScheduleEnabledCount     int
	ScheduleDisabledCount    int
	ScheduleEligibleCount    int
	StaleScanCount           int
	FilingEnabledCount       int
	NoMappedFindingsTotal    int
	MappedForgeIssuesTotal   int
	Warning                  string
	Rows                     []FleetRepoAuditRow
}

const defaultStaleScanAfter = 24 * time.Hour

// FleetHealthAudit builds per-repo and fleet-level scan/schedule/filing status.
func FleetHealthAudit(ctx context.Context, s *SQLiteStore, global GlobalSettingsSnapshot, schedulerEnabled bool, staleAfter time.Duration) (FleetHealthSummary, error) {
	if staleAfter <= 0 {
		staleAfter = defaultStaleScanAfter
	}
	rows, err := s.listFleetAuditRows(ctx)
	if err != nil {
		return FleetHealthSummary{}, err
	}
	settingsByID, err := s.batchRepoSettings(ctx, fleetAuditIDs(rows))
	if err != nil {
		return FleetHealthSummary{}, err
	}
	out := FleetHealthSummary{SchedulerGloballyEnabled: schedulerEnabled}
	now := time.Now().UTC()
	for _, base := range rows {
		settings := settingsByID[base.ID]
		settings.RepositoryID = base.ID
		effective, _ := ResolveEffectiveSettingsWithMeta(global, settings)
		filing := ResolveScanFilingPolicy(ScanFilingInput{
			Kind:      ScanKindManual,
			Effective: effective,
		})
		eligible, skipReason := ScheduleEligible(base.ConnectedRepo, effective)
		row := FleetRepoAuditRow{
			RepositoryID:       base.ID,
			FullName:           base.FullName,
			ForgeType:          base.ForgeType,
			ScanEnabled:        effective.Enabled,
			ScheduleEnabled:    effective.ScheduleEnabled,
			ScheduleCron:       effective.ScheduleCron,
			FilingEnabled:      filing.IssueFilingAllowed,
			ReportOnlyEnforced: !filing.IssueFilingAllowed || base.DryRunReportOnly,
			LastScanTrigger:    base.LastScanTrigger,
			LastScanCommit:     base.LastScanCommit,
			OpenFindings:       base.OpenFindings,
			NoMappedFindings:   base.NoMappedFindings,
			MappedForgeIssues:  base.MappedForgeIssues,
			ScheduleEligible:   eligible,
			ScheduleSkipReason: skipReason,
		}
		if base.LastScanAt != nil {
			t := *base.LastScanAt
			row.LastScanAt = &t
			age := now.Sub(t)
			row.StaleHours = age.Hours()
			row.StaleScan = age > staleAfter
		} else {
			row.StaleScan = true
		}
		if base.LastWebhookAt != nil {
			t := *base.LastWebhookAt
			row.LastWebhookAt = &t
		}
		out.Rows = append(out.Rows, row)
		out.TotalTracked++
		if row.ScanEnabled {
			out.ScanEnabledCount++
		}
		if row.ScheduleEnabled {
			out.ScheduleEnabledCount++
		} else if row.ScanEnabled {
			out.ScheduleDisabledCount++
		}
		if row.ScheduleEligible {
			out.ScheduleEligibleCount++
		}
		if row.StaleScan && row.ScanEnabled {
			out.StaleScanCount++
		}
		if row.FilingEnabled {
			out.FilingEnabledCount++
		}
		out.NoMappedFindingsTotal += row.NoMappedFindings
		out.MappedForgeIssuesTotal += row.MappedForgeIssues
	}
	if out.ScheduleDisabledCount > 0 && out.ScanEnabledCount > 0 {
		out.Warning = "Nightly fleet scans are disabled for " +
			strconv.Itoa(out.ScheduleDisabledCount) + "/" + strconv.Itoa(out.ScanEnabledCount) +
			" scan-enabled repositories."
	}
	if !schedulerEnabled {
		if out.Warning != "" {
			out.Warning += " "
		}
		out.Warning += "In-process scheduler is globally disabled."
	}
	return out, nil
}

type fleetAuditBase struct {
	ID                int64
	FullName          string
	ForgeType         string
	ConnectedRepo     bool
	LastScanAt        *time.Time
	LastScanTrigger   string
	LastScanCommit    string
	LastWebhookAt     *time.Time
	DryRunReportOnly  bool
	OpenFindings      int
	NoMappedFindings  int
	MappedForgeIssues int
}

func fleetAuditIDs(bases []fleetAuditBase) []int64 {
	ids := make([]int64, len(bases))
	for i, b := range bases {
		ids[i] = b.ID
	}
	return ids
}
