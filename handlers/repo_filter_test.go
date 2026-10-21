package handlers

import "testing"

func TestRepoAllowedIncludeExclude(t *testing.T) {
	include := []string{"commstech/*"}
	exclude := []string{"commstech/archived-*"}

	if !RepoAllowed("commstech/Repository-Detective", include, exclude) {
		t.Fatal("expected commstech/Repository-Detective to be allowed")
	}
	if RepoAllowed("commstech/archived-old", include, exclude) {
		t.Fatal("expected archived repo to be excluded")
	}
	if RepoAllowed("other/project", include, exclude) {
		t.Fatal("expected other/project to be denied by include list")
	}
}

func TestRepoAllowedOpenInclude(t *testing.T) {
	if !RepoAllowed("anyone/repo", nil, nil) {
		t.Fatal("expected open include to allow repo")
	}
	if RepoAllowed("anyone/repo", nil, []string{"anyone/repo"}) {
		t.Fatal("expected exclude to block repo")
	}
}
