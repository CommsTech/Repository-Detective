package preinstall_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/preinstall"
)

func TestValidateRepoURLRejectsFileScheme(t *testing.T) {
	_, err := preinstall.ValidateRepoURL("file:///etc/passwd", false)
	if err == nil {
		t.Fatal("expected file:// to be rejected")
	}
}

func TestValidateRepoURLRejectsLocalPath(t *testing.T) {
	_, err := preinstall.ValidateRepoURL("/tmp/evil", false)
	if err == nil {
		t.Fatal("expected local path to be rejected")
	}
}

func TestValidateRepoURLRejectsEmbeddedCredentials(t *testing.T) {
	_, err := preinstall.ValidateRepoURL("https://user:pass@github.com/o/r", false)
	if err == nil {
		t.Fatal("expected embedded credentials to be rejected")
	}
}

func TestValidateRepoURLRejectsLocalhost(t *testing.T) {
	_, err := preinstall.ValidateRepoURL("https://localhost/o/r", false)
	if err == nil {
		t.Fatal("expected localhost to be rejected")
	}
}

func TestValidateRepoURLAcceptsHTTPSGitHub(t *testing.T) {
	defer mockPublicDNS()()
	parsed, err := preinstall.ValidateRepoURL("https://github.com/owner/repo", false)
	if err != nil {
		t.Fatalf("valid github url: %v", err)
	}
	if parsed.Owner != "owner" || parsed.Name != "repo" {
		t.Fatalf("unexpected parse: %+v", parsed)
	}
	if parsed.Normalized != "https://github.com/owner/repo" {
		t.Fatalf("normalized: %s", parsed.Normalized)
	}
	if parsed.CloneURL != "https://github.com/owner/repo.git" {
		t.Fatalf("clone url: %s", parsed.CloneURL)
	}
}

func TestValidateRepoURLWithGitSuffixDoesNotDoubleDotGit(t *testing.T) {
	defer mockPublicDNS()()
	parsed, err := preinstall.ValidateRepoURL("https://github.com/octocat/Hello-World.git", false)
	if err != nil {
		t.Fatalf("valid github url with .git suffix: %v", err)
	}
	if parsed.Name != "Hello-World" {
		t.Fatalf("name should strip .git: %q", parsed.Name)
	}
	if parsed.CloneURL != "https://github.com/octocat/Hello-World.git" {
		t.Fatalf("clone url must not double .git: %s", parsed.CloneURL)
	}
}

func TestCloneEnvDoesNotExposeSecrets(t *testing.T) {
	t.Setenv("REPOSITORY_DETECTIVE_GITEA_TOKEN", "secret-token-value")
	if preinstall.CloneEnvExposesSecrets() {
		t.Fatal("minimal git env must not include operator secrets")
	}
}

func TestNormalizeAuditDepth(t *testing.T) {
	if preinstall.NormalizeAuditDepth("") != "standard" {
		t.Fatal("empty depth should be standard")
	}
	if preinstall.NormalizeAuditDepth("quick") != "quick" {
		t.Fatal("quick depth")
	}
}
