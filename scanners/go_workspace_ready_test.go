package scanners

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGoTypecheckWorkspaceReadyMixedPackages(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.go"), []byte("package other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, reason := GoTypecheckWorkspaceReady(dir, nil)
	if ok {
		t.Fatalf("expected mixed packages to be not ready, reason=%q", reason)
	}
	if reason == "" {
		t.Fatal("expected reason")
	}
}

func TestGoTypecheckWorkspaceReadySinglePackage(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, reason := GoTypecheckWorkspaceReady(dir, nil)
	if !ok {
		t.Fatalf("expected ready workspace, reason=%q", reason)
	}
}

func TestIsStaticcheckWorkspaceOrCompileFailure(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"found packages main (fixture.go) and proof (proof_secret.go)", true},
		{"graph/calibrate.go:19:32: undefined: builder", true},
		{"requires at least go1.26", true},
		{"unexpected end of JSON input", false},
	}
	for _, tc := range cases {
		if got := isStaticcheckWorkspaceOrCompileFailure(tc.in); got != tc.want {
			t.Fatalf("%q => %v want %v", tc.in, got, tc.want)
		}
	}
}
