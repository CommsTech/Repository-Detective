package store_test

import (
	"context"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func goScannerGlobal() store.GlobalSettingsSnapshot {
	g := store.DefaultGlobalSettings()
	g.EnableGovulncheck = false
	g.EnableGosec = false
	g.EnableStaticcheck = false
	g.GoScannerMaxFindings = 100
	return g
}

func TestMigrationAddsGoScannerColumns(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "go", FullName: "o/go"})
	enabled := true
	maxFindings := 250
	if err := s.SaveRepoSettings(ctx, store.RepoSettings{
		RepositoryID: repo.ID, EnableGovulncheck: &enabled, GoScannerMaxFindings: &maxFindings,
	}); err != nil {
		t.Fatalf("save go scanner settings: %v", err)
	}
	got, err := s.GetRepoSettings(ctx, repo.ID)
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if got.EnableGovulncheck == nil || !*got.EnableGovulncheck {
		t.Fatal("expected stored enable_govulncheck=true")
	}
	if got.GoScannerMaxFindings == nil || *got.GoScannerMaxFindings != 250 {
		t.Fatalf("expected go_scanner_max_findings=250, got %v", got.GoScannerMaxFindings)
	}
}

func TestGoScannerSettingsInheritGlobal(t *testing.T) {
	global := goScannerGlobal()
	effective := store.ResolveRepoSettings(global, store.RepoSettings{})
	if effective.EnableGovulncheck || effective.EnableGosec || effective.EnableStaticcheck {
		t.Fatal("expected Go scanners disabled by default")
	}
	if effective.GoScannerMaxFindings != 100 {
		t.Fatalf("expected max findings 100, got %d", effective.GoScannerMaxFindings)
	}
}

func TestGoScannerSettingsOverrideGlobal(t *testing.T) {
	global := goScannerGlobal()
	on := true
	timeout := 45
	maxFindings := 50
	effective := store.ResolveRepoSettings(global, store.RepoSettings{
		EnableGosec:          &on,
		GosecTimeoutSeconds:  &timeout,
		GoScannerMaxFindings: &maxFindings,
	})
	if !effective.EnableGosec {
		t.Fatal("repo should enable gosec")
	}
	if effective.GosecTimeoutSeconds != 45 || effective.GoScannerMaxFindings != 50 {
		t.Fatalf("unexpected overrides: timeout=%d max=%d", effective.GosecTimeoutSeconds, effective.GoScannerMaxFindings)
	}
}

func TestInvalidGoScannerThresholdRejected(t *testing.T) {
	bad := 5000
	err := store.ValidateSettingsUpdate(store.SettingsUpdate{GoScannerMaxFindings: &bad})
	if err == nil {
		t.Fatal("expected validation error for go_scanner_max_findings above maximum")
	}
	timeout := 4000
	err = store.ValidateSettingsUpdate(store.SettingsUpdate{GovulncheckTimeoutSeconds: &timeout})
	if err == nil {
		t.Fatal("expected validation error for govulncheck timeout above maximum")
	}
}

func TestEnabledScannersListIncludesGoScanners(t *testing.T) {
	list := store.EnabledScannersList(store.EffectiveSettings{
		EnableGovulncheck: true,
		EnableGosec:       true,
		EnableStaticcheck: true,
	})
	found := map[string]bool{}
	for _, name := range list {
		found[name] = true
	}
	for _, want := range []string{"govulncheck", "gosec", "staticcheck"} {
		if !found[want] {
			t.Fatalf("expected %q in enabled scanners list, got %v", want, list)
		}
	}
}
