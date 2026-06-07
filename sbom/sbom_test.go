package sbom_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"git.commsnet.org/commstech/bugbot/sbom"
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
