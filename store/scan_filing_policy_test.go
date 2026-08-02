package store_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func productionEffective() store.EffectiveSettings {
	return store.EffectiveFromGlobalSnapshot(store.DefaultGlobalSettings())
}

func TestResolveScanFilingPolicyConnectedRepoFilesByDefault(t *testing.T) {
	p := store.ResolveScanFilingPolicy(store.ScanFilingInput{
		Kind:      store.ScanKindManual,
		Effective: productionEffective(),
	})
	if !p.IssueFilingAllowed || p.ReportOnlyDryRun || !p.WillFileIssues {
		t.Fatalf("expected filing by default: %+v", p)
	}
	if p.Mode != store.ScanModeProductionSelfHosted {
		t.Fatalf("mode=%s", p.Mode)
	}
}

func TestResolveScanFilingPolicyManualDryRunSkipsIssues(t *testing.T) {
	p := store.ResolveScanFilingPolicy(store.ScanFilingInput{
		Kind:          store.ScanKindManual,
		Effective:     productionEffective(),
		RequestDryRun: true,
	})
	if p.WillFileIssues || !p.ReportOnlyDryRun {
		t.Fatalf("expected dry run: %+v", p)
	}
	if p.Mode != store.ScanModeReportOnlyDryRun {
		t.Fatalf("mode=%s", p.Mode)
	}
}

func TestResolveScanFilingPolicyBetaSafeNeverFiles(t *testing.T) {
	e := productionEffective()
	e.IssuePolicy = store.IssuePolicyOff
	e.PolicyLevel = store.PolicyMonitorOnly
	p := store.ResolveScanFilingPolicy(store.ScanFilingInput{
		Kind:      store.ScanKindManual,
		Effective: e,
	})
	if p.WillFileIssues || !p.ReportOnlyDryRun || !p.DryRunCheckboxLocked {
		t.Fatalf("expected enforced report-only: %+v", p)
	}
	if p.Mode != store.ScanModePrivateBetaSafe {
		t.Fatalf("mode=%s", p.Mode)
	}
}

func TestResolveScanFilingPolicyPreinstallAlwaysReportOnly(t *testing.T) {
	p := store.ResolveScanFilingPolicy(store.ScanFilingInput{
		Kind:          store.ScanKindPreinstall,
		Effective:     productionEffective(),
		RequestDryRun: false,
	})
	if p.WillFileIssues || !p.ReportOnlyDryRun || !p.DryRunCheckboxLocked {
		t.Fatalf("preinstall must be report-only: %+v", p)
	}
	if p.Mode != store.ScanModePreinstallAudit {
		t.Fatalf("mode=%s", p.Mode)
	}
}

func TestDeploymentScanMode(t *testing.T) {
	g := store.DefaultGlobalSettings()
	if store.DeploymentScanMode(g) != store.ScanModeProductionSelfHosted {
		t.Fatal("default global should allow filing")
	}
	g.IssuePolicy = store.IssuePolicyOff
	if store.DeploymentScanMode(g) != store.ScanModePrivateBetaSafe {
		t.Fatal("off policy should be beta safe mode")
	}
}
