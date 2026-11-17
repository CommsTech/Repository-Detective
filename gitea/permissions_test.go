package gitea_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/gitea"
)

func TestBuildPermissionMatrix(t *testing.T) {
	m := gitea.BuildPermissionMatrix(gitea.RepoPermissions{Pull: true})
	if m.RepositoryRead != "PASS" || m.IssuesWrite != "NOT_GRANTED" {
		t.Fatalf("%+v", m)
	}
	m = gitea.BuildPermissionMatrix(gitea.RepoPermissions{Push: true})
	if m.IssuesWrite != "PASS" || m.BranchPRRemediation != "PASS" {
		t.Fatalf("%+v", m)
	}
}

func TestFindHookByURL(t *testing.T) {
	hooks := []gitea.HookSummary{{URL: "https://rd.example.com/webhook/"}}
	_, ok := gitea.FindHookByURL(hooks, "https://rd.example.com/webhook")
	if !ok {
		t.Fatal("expected match")
	}
}
