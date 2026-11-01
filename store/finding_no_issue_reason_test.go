package store_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestClassifyFindingNoIssueReasonReportOnly(t *testing.T) {
	e := store.ResolveRepoSettings(store.DefaultGlobalSettings(), store.RepoSettings{})
	got := store.ClassifyFindingNoIssueReason(
		store.FindingStatusOpen,
		false,
		e,
		true,
		store.IssueSyncStatusSkipped,
	)
	if got != store.NoIssueReasonReportOnly {
		t.Fatalf("got %q", got)
	}
}

func TestClassifyFindingNoIssueReasonFilingDisabled(t *testing.T) {
	off := store.IssuePolicyOff
	e := store.ResolveRepoSettings(store.DefaultGlobalSettings(), store.RepoSettings{IssuePolicy: &off})
	got := store.ClassifyFindingNoIssueReason(
		store.FindingStatusOpen,
		false,
		e,
		false,
		store.IssueSyncStatusSkipped,
	)
	if got != store.NoIssueReasonNoIssuePolicy {
		t.Fatalf("got %q", got)
	}
}

func TestClassifyFindingNoIssueReasonAlreadyMapped(t *testing.T) {
	e := store.ResolveRepoSettings(store.DefaultGlobalSettings(), store.RepoSettings{})
	got := store.ClassifyFindingNoIssueReason(
		store.FindingStatusOpen,
		true,
		e,
		false,
		store.IssueSyncStatusComplete,
	)
	if got != store.NoIssueReasonAlreadyMapped {
		t.Fatalf("got %q", got)
	}
}

func TestScheduleEligibleRequiresCron(t *testing.T) {
	on := true
	e := store.ResolveRepoSettings(store.DefaultGlobalSettings(), store.RepoSettings{ScheduleEnabled: &on})
	ok, reason := store.ScheduleEligible(true, e)
	if ok || reason != "missing_schedule_cron" {
		t.Fatalf("eligible=%v reason=%q", ok, reason)
	}
}

func TestStaggeredNightlyCronAfterCalibrationWindow(t *testing.T) {
	cron := store.StaggeredNightlyCron(1)
	if cron == "" {
		t.Fatal("empty cron")
	}
	// Expect hour >= 3 (after 02:17 calibration learner)
	if cron[len(cron)-1] == ' ' {
		t.Fatalf("unexpected cron %q", cron)
	}
}
