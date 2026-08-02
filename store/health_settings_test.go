package store_test

import (
	"context"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func healthGlobal() store.GlobalSettingsSnapshot {
	g := store.DefaultGlobalSettings()
	g.EnableHealthChecks = true
	g.EnableAIRiskChecks = false
	g.HealthLargeFileLines = 1000
	return g
}

func TestMigrationAddsHealthColumns(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	disabled := false
	if err := s.SaveRepoSettings(ctx, store.RepoSettings{
		RepositoryID: repo.ID, EnableHealthChecks: &disabled,
	}); err != nil {
		t.Fatalf("save health settings: %v", err)
	}
	got, err := s.GetRepoSettings(ctx, repo.ID)
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if got.EnableHealthChecks == nil || *got.EnableHealthChecks {
		t.Fatal("expected stored enable_health_checks=false")
	}
}

func TestHealthSettingsInheritGlobal(t *testing.T) {
	global := healthGlobal()
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{})
	if !effective.EnableHealthChecks || effective.EnableAIRiskChecks {
		t.Fatalf("expected global health defaults, got %+v", effective)
	}
	if effective.HealthLargeFileLines != 1000 {
		t.Fatalf("expected large file lines 1000, got %d", effective.HealthLargeFileLines)
	}
}

func TestHealthSettingsOverrideGlobal(t *testing.T) {
	global := healthGlobal()
	healthOff := false
	reliabilityOff := false
	largeFile := 200
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{
		EnableHealthChecks:   &healthOff,
		EnableReliabilityChecks: &reliabilityOff,
		HealthLargeFileLines: &largeFile,
	})
	if effective.EnableHealthChecks {
		t.Fatal("repo should disable health checks")
	}
	if effective.EnableReliabilityChecks {
		t.Fatal("repo should disable reliability")
	}
	if effective.HealthLargeFileLines != 200 {
		t.Fatalf("expected threshold override 200, got %d", effective.HealthLargeFileLines)
	}
}

func TestRepoEnablesAIRiskWhileGlobalDisabled(t *testing.T) {
	global := healthGlobal()
	global.EnableAIRiskChecks = false
	aiOn := true
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{EnableAIRiskChecks: &aiOn})
	if !effective.EnableAIRiskChecks {
		t.Fatal("repo override should enable AI risk")
	}
}

func TestRepoDisablesAIRiskWhileGlobalEnabled(t *testing.T) {
	global := healthGlobal()
	global.EnableAIRiskChecks = true
	aiOff := false
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{EnableAIRiskChecks: &aiOff})
	if effective.EnableAIRiskChecks {
		t.Fatal("repo override should disable AI risk")
	}
}

func TestInvalidHealthThresholdRejected(t *testing.T) {
	bad := 99999
	err := store.ValidateSettingsUpdate(store.SettingsUpdate{HealthLargeFileLines: &bad})
	if err == nil {
		t.Fatal("expected validation error for out-of-range threshold")
	}
}

func TestSanitizeInvalidStoredHealthThreshold(t *testing.T) {
	global := healthGlobal()
	bad := 10
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{HealthLargeFileLines: &bad})
	if effective.HealthLargeFileLines != global.HealthLargeFileLines {
		t.Fatalf("invalid stored threshold should fall back to global, got %d", effective.HealthLargeFileLines)
	}
}
