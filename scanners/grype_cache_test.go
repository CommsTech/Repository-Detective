package scanners_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/scanners"
	"github.com/sirupsen/logrus"
)

func TestCleanupStaleScannerScratch(t *testing.T) {
	root := t.TempDir()
	stale := filepath.Join(root, "grype-scratch123")
	fresh := filepath.Join(root, "grype-scratch999")
	other := filepath.Join(root, "keep-me")
	if err := os.Mkdir(stale, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(fresh, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(other, 0o755); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}

	removed := scanners.CleanupStaleScannerScratch(root, time.Hour, logrus.New())
	if removed != 1 {
		t.Fatalf("removed=%d want 1", removed)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale dir still present")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatalf("fresh dir removed: %v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatalf("unrelated dir removed: %v", err)
	}
}

func TestEnsureScannerTempDir(t *testing.T) {
	prevTMPDIR := os.Getenv("TMPDIR")
	prevCache := os.Getenv("XDG_CACHE_HOME")
	t.Cleanup(func() {
		if prevTMPDIR == "" {
			_ = os.Unsetenv("TMPDIR")
		} else {
			_ = os.Setenv("TMPDIR", prevTMPDIR)
		}
		if prevCache == "" {
			_ = os.Unsetenv("XDG_CACHE_HOME")
		} else {
			_ = os.Setenv("XDG_CACHE_HOME", prevCache)
		}
	})

	data := t.TempDir()
	got := scanners.EnsureScannerTempDir(data, logrus.New())
	want := filepath.Join(data, "tmp")
	if got != want {
		t.Fatalf("tmpdir=%q want %q", got, want)
	}
	if os.Getenv("TMPDIR") != want {
		t.Fatalf("TMPDIR env=%q want %q", os.Getenv("TMPDIR"), want)
	}
}
