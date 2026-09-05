package scanners

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// EnsureScannerTempDir creates and exports a durable TMPDIR under dataDir so
// scanner scratch (especially grype) does not fill the container overlay /tmp,
// and so leftovers survive only under an operator-managed path.
func EnsureScannerTempDir(dataDir string, logger *logrus.Logger) string {
	dataDir = strings.TrimSpace(dataDir)
	if dataDir == "" {
		dataDir = "/app/data"
	}
	tmpDir := filepath.Join(dataDir, "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		if logger != nil {
			logger.WithError(err).Warn("could not create scanner TMPDIR; using process default")
		}
		return strings.TrimSpace(os.Getenv("TMPDIR"))
	}
	if err := os.Setenv("TMPDIR", tmpDir); err != nil && logger != nil {
		logger.WithError(err).Warn("could not set TMPDIR for scanners")
	}
	if cache := filepath.Join(dataDir, "cache"); true {
		if err := os.MkdirAll(cache, 0o755); err != nil && logger != nil {
			logger.WithError(err).Debug("could not create scanner cache dir")
		}
		if os.Getenv("XDG_CACHE_HOME") == "" {
			if err := os.Setenv("XDG_CACHE_HOME", cache); err != nil && logger != nil {
				logger.WithError(err).Debug("could not set XDG_CACHE_HOME for scanners")
			}
		}
	}
	return tmpDir
}

// CleanupStaleScannerScratch removes abandoned grype/scanner temp directories.
// maxAge of 0 deletes all matching scratch dirs (safe at startup when no scan runs yet).
func CleanupStaleScannerScratch(tmpDir string, maxAge time.Duration, logger *logrus.Logger) int {
	tmpDir = strings.TrimSpace(tmpDir)
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return 0
	}
	removed := 0
	cutoff := time.Now().Add(-maxAge)
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "grype-scratch") &&
			!strings.HasPrefix(name, "grype-cache") &&
			!strings.HasPrefix(name, "trivy-") &&
			!strings.HasPrefix(name, "getter") {
			continue
		}
		info, ierr := entry.Info()
		if ierr != nil {
			continue
		}
		if maxAge > 0 && info.ModTime().After(cutoff) {
			continue
		}
		path := filepath.Join(tmpDir, name)
		if err := os.RemoveAll(path); err != nil {
			if logger != nil {
				logger.WithError(err).Debugf("scanner scratch cleanup skipped: %s", path)
			}
			continue
		}
		removed++
	}
	if removed > 0 && logger != nil {
		logger.Infof("Removed %d stale scanner scratch dir(s) under %s", removed, tmpDir)
	}
	return removed
}

// WarmGrypeDB ensures the grype vulnerability database is present and usable.
// Failures are logged and non-fatal so startup is not blocked on network.
func WarmGrypeDB(ctx context.Context, logger *logrus.Logger) {
	if !commandAvailable("grype") {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	statusCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	statusOut, statusErr := exec.CommandContext(statusCtx, "grype", "db", "status").CombinedOutput()
	statusText := strings.ToLower(string(statusOut))
	needsUpdate := statusErr != nil ||
		strings.Contains(statusText, "invalid") ||
		strings.Contains(statusText, "not found") ||
		strings.Contains(statusText, "malformed") ||
		strings.Contains(statusText, "no database")
	if !needsUpdate {
		if logger != nil {
			logger.Debug("grype vulnerability DB status OK")
		}
		return
	}
	if logger != nil {
		logger.Warn("grype vulnerability DB missing or invalid; attempting update")
	}
	updateCtx, updateCancel := context.WithTimeout(ctx, 10*time.Minute)
	defer updateCancel()
	out, err := exec.CommandContext(updateCtx, "grype", "db", "update").CombinedOutput()
	if err != nil {
		if logger != nil {
			logger.WithError(err).Warnf("grype db update failed: %s", firstLine(string(out)))
		}
		return
	}
	if logger != nil {
		logger.Info("grype vulnerability DB updated")
	}
}
