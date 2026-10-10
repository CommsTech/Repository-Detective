package sbom_test

import (
	"os"
	"path/filepath"
	"testing"

	"git.commsnet.org/commstech/repository-detective/sbom"
)

func TestPersistArtifactSurvivesWorkspaceCleanup(t *testing.T) {
	dataRoot := t.TempDir()
	workspace := filepath.Join(dataRoot, "tmp", "rd-scan-123", ".rd-sbom")
	if err := os.MkdirAll(workspace, 0o750); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(workspace, "sbom-go.cdx.json")
	payload := []byte(`{"bomFormat":"CycloneDX","components":[{"name":"demo"}]}`)
	if err := os.WriteFile(src, payload, 0o600); err != nil {
		t.Fatal(err)
	}

	dest, err := sbom.PersistArtifact(dataRoot, 1, "scan-abc", src)
	if err != nil {
		t.Fatal(err)
	}
	wantDir := filepath.Join(dataRoot, "sbom", "1", "scan-abc")
	if filepath.Dir(dest) != wantDir {
		t.Fatalf("dest dir %q want under %q", dest, wantDir)
	}
	if !sbom.ArtifactFileAvailable(dest) {
		t.Fatalf("durable artifact missing: %s", dest)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("payload mismatch")
	}

	// Simulate scan workspace cleanup — durable copy must remain.
	if err := os.RemoveAll(filepath.Join(dataRoot, "tmp")); err != nil {
		t.Fatal(err)
	}
	if sbom.ArtifactFileAvailable(src) {
		t.Fatal("expected workspace source to be gone")
	}
	if !sbom.ArtifactFileAvailable(dest) {
		t.Fatal("durable SBOM disappeared after workspace cleanup")
	}
}

func TestDataRootFromDatabasePath(t *testing.T) {
	if got := sbom.DataRootFromDatabasePath("/app/data/repository-detective.db"); got != "/app/data" {
		t.Fatalf("got %q", got)
	}
	if got := sbom.DataRootFromDatabasePath(""); got != "/app/data" {
		t.Fatalf("empty got %q", got)
	}
}

func TestArtifactFileAvailable(t *testing.T) {
	if sbom.ArtifactFileAvailable("") {
		t.Fatal("empty path should be unavailable")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "sbom.json")
	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !sbom.ArtifactFileAvailable(path) {
		t.Fatal("expected file available")
	}
	if sbom.ArtifactFileAvailable(filepath.Join(dir, "missing.json")) {
		t.Fatal("missing file should be unavailable")
	}
}
