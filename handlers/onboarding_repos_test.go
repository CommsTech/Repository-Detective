package handlers

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/gitea"
)

func TestMergeOnboardOrgListDedupes(t *testing.T) {
	got := mergeOnboardOrgList([]string{" commstech ", "Commstech"}, []string{"commstech", "other"})
	if len(got) != 2 {
		t.Fatalf("expected 2 orgs, got %v", got)
	}
	if got[0] != "commstech" || got[1] != "other" {
		t.Fatalf("unexpected order/content: %v", got)
	}
}

func TestSortRepositoriesForOnboardPinsDogfood(t *testing.T) {
	repos := []gitea.RepositorySummary{
		{FullName: "commstech/AMMBER"},
		{FullName: "commstech/repository-detective"},
		{FullName: "commstech/wiki"},
	}
	sortRepositoriesForOnboard(repos)
	if repos[0].FullName != "commstech/repository-detective" {
		t.Fatalf("dogfood repo should be first, got %q", repos[0].FullName)
	}
}

func TestIsDogfoodRepository(t *testing.T) {
	cases := []struct {
		fullName string
		want     bool
	}{
		{"commstech/Repository-Detective", true},
		{"commstech/repository-detective", true},
		{"commstech/Bugbot", true},
		{"commstech/AMMBER", false},
	}
	for _, tc := range cases {
		repo := gitea.RepositorySummary{FullName: tc.fullName}
		if got := isDogfoodRepository(repo); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.fullName, got, tc.want)
		}
	}
}
