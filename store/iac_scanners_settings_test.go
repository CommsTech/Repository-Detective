package store_test

import (
	"context"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func iacScannerGlobal() store.GlobalSettingsSnapshot {
	g := store.DefaultGlobalSettings()
	g.EnableHadolint = false
	g.EnableCheckov = false
	g.IACScannerMaxFindings = 100
	return g
}

func TestMigrationAddsIACScannerColumns(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "iac", FullName: "o/iac"})
	enabled := true
	maxFindings := 200
	if err := s.SaveRepoSettings(ctx, store.RepoSettings{
		RepositoryID: repo.ID, EnableCheckov: &enabled, IACScannerMaxFindings: &maxFindings,
	}); err != nil {
		t.Fatalf("save iac settings: %v", err)
	}
	got, err := s.GetRepoSettings(ctx, repo.ID)
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if got.EnableCheckov == nil || !*got.EnableCheckov {
		t.Fatal("expected stored enable_checkov=true")
	}
	if got.IACScannerMaxFindings == nil || *got.IACScannerMaxFindings != 200 {
		t.Fatalf("expected iac_scanner_max_findings=200, got %v", got.IACScannerMaxFindings)
	}
}

func TestIACScannerSettingsInheritGlobal(t *testing.T) {
	global := iacScannerGlobal()
	effective := store.ResolveRepoSettings(global, store.RepoSettings{})
	if effective.EnableHadolint || effective.EnableCheckov {
		t.Fatal("expected IaC scanners disabled by default")
	}
	if effective.IACScannerMaxFindings != 100 {
		t.Fatalf("expected max findings 100, got %d", effective.IACScannerMaxFindings)
	}
}

func TestInvalidIACScannerThresholdRejected(t *testing.T) {
	bad := 5000
	err := store.ValidateSettingsUpdate(store.SettingsUpdate{IACScannerMaxFindings: &bad})
	if err == nil {
		t.Fatal("expected validation error for iac_scanner_max_findings above maximum")
	}
}

func TestEnabledScannersListIncludesIACScanners(t *testing.T) {
	list := store.EnabledScannersList(store.EffectiveSettings{
		EnableHadolint: true,
		EnableCheckov:  true,
	})
	found := map[string]bool{}
	for _, name := range list {
		found[name] = true
	}
	for _, want := range []string{"hadolint", "checkov"} {
		if !found[want] {
			t.Fatalf("expected %q in enabled scanners list, got %v", want, list)
		}
	}
}
