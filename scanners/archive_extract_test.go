package scanners_test

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"git.commsnet.org/commstech/repository-detective/scanners"
)

func writeTestZip(t *testing.T, entries map[string]string) string {
	t.Helper()
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "test.zip")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExtractZipArchiveBlocksZipSlip(t *testing.T) {
	zipPath := writeTestZip(t, map[string]string{
		"repo-main/evil.txt":              "ok",
		"repo-main/../../outside.txt":     "bad",
	})
	dest := t.TempDir()

	_, _, _, err := scanners.ExtractZipArchive(zipPath, dest, 100, 1024*1024)
	if err == nil {
		t.Fatal("expected zip slip to be rejected")
	}
}

func TestExtractZipArchiveAcceptsNormalPaths(t *testing.T) {
	zipPath := writeTestZip(t, map[string]string{
		"repo-main/go.mod":         "module example.com/test\n\ngo 1.21\n",
		"repo-main/cmd/main.go":    "package main\n",
		"repo-main/README.md":      "# hi\n",
	})
	dest := t.TempDir()

	count, _, truncated, err := scanners.ExtractZipArchive(zipPath, dest, 100, 1024*1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 files, got %d", count)
	}
	if truncated {
		t.Fatal("did not expect truncation")
	}

	if _, err := os.Stat(filepath.Join(dest, "go.mod")); err != nil {
		t.Fatalf("expected stripped go.mod: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "cmd", "main.go")); err != nil {
		t.Fatalf("expected cmd/main.go: %v", err)
	}
}

func TestExtractZipArchiveEnforcesMaxFiles(t *testing.T) {
	entries := map[string]string{
		"repo-main/a.txt": "a",
		"repo-main/b.txt": "b",
		"repo-main/c.txt": "c",
	}
	zipPath := writeTestZip(t, entries)
	dest := t.TempDir()

	count, _, truncated, err := scanners.ExtractZipArchive(zipPath, dest, 2, 1024*1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 files extracted, got %d", count)
	}
	if !truncated {
		t.Fatal("expected truncated flag")
	}
}

func TestExtractZipArchiveEnforcesMaxBytes(t *testing.T) {
	zipPath := writeTestZip(t, map[string]string{
		"repo-main/big.txt": string(make([]byte, 2048)),
	})
	dest := t.TempDir()

	_, _, truncated, err := scanners.ExtractZipArchive(zipPath, dest, 10, 512)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !truncated {
		t.Fatal("expected truncated due to byte limit")
	}
}
