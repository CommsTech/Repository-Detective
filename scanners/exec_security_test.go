package scanners_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/internal/security"
)

func TestSubprocessEnvExcludesSecrets(t *testing.T) {
	t.Setenv("REPOSITORY_DETECTIVE_GITEA_TOKEN", "secret-value")
	if security.SubprocessEnvExposesSecrets() {
		t.Fatal("scanner/git env must not inherit operator secrets")
	}
	env := security.MinimalSubprocessEnv()
	for _, entry := range env {
		if len(entry) > 35 && entry[:35] == "REPOSITORY_DETECTIVE_GITEA_TOKEN=" {
			t.Fatalf("secret leaked in env: %s", entry)
		}
	}
}
