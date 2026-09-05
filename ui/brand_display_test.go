package ui

import "testing"

func TestScrubLegacyBrand(t *testing.T) {
	in := `failed to make request: Get "https://git.commsnet.org/api/v1/repos/commstech/Bugbot": context deadline exceeded`
	got := scrubLegacyBrand(in)
	if want := `failed to make request: Get "https://git.commsnet.org/api/v1/repos/commstech/Repository-Detective": context deadline exceeded`; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if scrubLegacyBrand("Bugbot control plane") != "Repository Detective control plane" {
		t.Fatal("expected product rename")
	}
}

func TestDisplayPathStripsScratch(t *testing.T) {
	in := "app/data/tmp/bugbot-scan-2096934807/data/tmp/bugbot-scan-2096934807/src/main.py"
	got := displayPath(in)
	if got != "src/main.py" {
		t.Fatalf("got %q", got)
	}
	if displayPath("src/ok.go") != "src/ok.go" {
		t.Fatal("clean path must pass through")
	}
}
