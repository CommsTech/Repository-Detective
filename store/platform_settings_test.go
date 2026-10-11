package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestPlatformSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "ps.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()
	in := store.PlatformSettings{
		ScanProfile:      store.ScanProfileStandard,
		SchedulerEnabled: store.BoolPtr(true),
		EnableGitleaks:   store.BoolPtr(false),
		SeverityGate:     "medium",
		AnalysisDepth:    store.IntPtr(2),
		UpdatedBy:        "test",
	}
	if err := s.SavePlatformSettings(ctx, in); err != nil {
		t.Fatal(err)
	}
	out, err := s.GetPlatformSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if out.ScanProfile != store.ScanProfileStandard {
		t.Fatalf("scan_profile=%q", out.ScanProfile)
	}
	if out.SchedulerEnabled == nil || !*out.SchedulerEnabled {
		t.Fatal("expected scheduler enabled")
	}
	if out.EnableGitleaks == nil || *out.EnableGitleaks {
		t.Fatal("expected gitleaks disabled")
	}
	if out.UpdatedAt == "" {
		t.Fatal("expected updated_at")
	}

	base := store.DefaultGlobalSettings()
	base.EnableGitleaks = true
	merged := store.ApplyPlatformSettingsToGlobal(base, out)
	if merged.EnableGitleaks {
		t.Fatal("expected gitleaks overlay false")
	}
	if merged.SeverityGate != "medium" {
		t.Fatalf("severity=%q", merged.SeverityGate)
	}
}
