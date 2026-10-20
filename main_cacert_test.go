package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCACertPEM(t *testing.T) {
	if pem, err := loadCACertPEM(""); err != nil || pem != "" {
		t.Fatalf("empty path: pem=%q err=%v", pem, err)
	}
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.pem")
	if err := os.WriteFile(bad, []byte("hello world"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := loadCACertPEM(bad); err == nil {
		t.Fatal("expected error for non-PEM file")
	}
	good := filepath.Join(dir, "good.pem")
	if err := os.WriteFile(good, []byte("-----BEGIN CERTIFICATE-----\nabc\n-----END CERTIFICATE-----\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	pem, err := loadCACertPEM(good)
	if err != nil || pem == "" {
		t.Fatalf("good pem: pem=%q err=%v", pem, err)
	}
	if _, err := loadCACertPEM(filepath.Join(dir, "missing.pem")); err == nil {
		t.Fatal("expected error for missing file")
	}
}
