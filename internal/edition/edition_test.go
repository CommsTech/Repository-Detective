package edition_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/internal/edition"
)

func TestNormalizeForcesCommunityWithoutLicense(t *testing.T) {
	c := edition.Normalize("commercial", "", 10)
	if c.Name != edition.Community {
		t.Fatalf("expected community without license, got %s", c.Name)
	}
}

func TestCommercialWithLicense(t *testing.T) {
	c := edition.Normalize("commercial", "RD-TEST-KEY", 10)
	if !c.IsCommercial() {
		t.Fatal("expected commercial")
	}
	if c.MaxRepos() != 0 {
		t.Fatalf("expected unlimited repos, got %d", c.MaxRepos())
	}
}

func TestCommunityRepoCap(t *testing.T) {
	c := edition.Normalize("community", "", 10)
	if c.MaxRepos() != 10 {
		t.Fatalf("max repos: %d", c.MaxRepos())
	}
}
