package sbom_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"git.commsnet.org/commstech/repository-detective/sbom"
)

func TestGenerateAndCheckNoManifest(t *testing.T) {
	dir := t.TempDir()
	res, err := sbom.GenerateAndCheck(context.Background(), dir, dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != sbom.StatusNoSupportedManifest {
		t.Fatalf("status %q want %q", res.Status, sbom.StatusNoSupportedManifest)
	}
}

func TestGenerateAndCheckPyprojectToml(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\nname=\"demo\"\nversion=\"0.1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := sbom.GenerateAndCheck(context.Background(), dir, dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status == sbom.StatusNoSupportedManifest {
		t.Fatalf("pyproject.toml should be detected, got %q detail=%q", res.Status, res.Detail)
	}
	if res.Status == sbom.StatusToolMissing {
		t.Skip("syft not installed")
	}
}

func TestStatusExpectsArtifact(t *testing.T) {
	if !sbom.StatusExpectsArtifact(string(sbom.StatusVulnerabilitiesFound)) {
		t.Fatal("vuln SBOM should expect an artifact file")
	}
	if sbom.StatusExpectsArtifact(string(sbom.StatusNoSupportedManifest)) {
		t.Fatal("no-manifest must not expect an artifact file")
	}
	if sbom.StatusExpectsArtifact(string(sbom.StatusToolMissing)) {
		t.Fatal("tool-missing must not expect an artifact file")
	}
}

func TestGenerateAndCheckNestedManifest(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "apps", "api")
	if err := os.MkdirAll(nested, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "package.json"), []byte(`{"name":"api","version":"1.0.0"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := sbom.GenerateAndCheck(context.Background(), dir, dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status == sbom.StatusNoSupportedManifest {
		t.Fatalf("nested package.json should be detected, got %q detail=%q", res.Status, res.Detail)
	}
	if res.Status == sbom.StatusToolMissing {
		t.Skip("syft not installed")
	}
}

func TestGenerateAndCheckGoModuleWithoutTools(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := sbom.GenerateAndCheck(context.Background(), dir, dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != sbom.StatusToolMissing && res.Status != sbom.StatusGenerated && res.Status != sbom.StatusCheckClean {
		t.Fatalf("unexpected status %q detail=%q", res.Status, res.Detail)
	}
}

func TestEmptyDirReturnsNoManifest(t *testing.T) {
	res, err := sbom.GenerateAndCheck(context.Background(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != sbom.StatusNoSupportedManifest {
		t.Fatalf("status %q", res.Status)
	}
}

func TestEmptyWorkspaceWithSyftStillNoManifest(t *testing.T) {
	if _, err := exec.LookPath("syft"); err != nil {
		t.Skip("syft not installed")
	}
	dir := t.TempDir()
	res, err := sbom.GenerateAndCheck(context.Background(), dir, dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != sbom.StatusNoSupportedManifest {
		t.Fatalf("empty dir with syft installed: status %q want %q (detail=%q)", res.Status, sbom.StatusNoSupportedManifest, res.Detail)
	}
}
