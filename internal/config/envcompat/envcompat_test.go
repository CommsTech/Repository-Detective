package envcompat_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/internal/config/envcompat"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func TestApplySetsRepositoryDetectiveEnv(t *testing.T) {
	t.Setenv("REPOSITORY_DETECTIVE_ENABLE_SEMGREP", "true")

	v := viper.New()
	v.SetDefault("enable_semgrep", false)
	envcompat.Apply(v, logrus.New())

	if !v.GetBool("enable_semgrep") {
		t.Fatal("expected REPOSITORY_DETECTIVE_ENABLE_SEMGREP to apply")
	}
}

func TestApplyIgnoresUnknownLegacyPrefix(t *testing.T) {
	t.Setenv("BUGBOT_ENABLE_TRIVY", "false")
	t.Setenv("REPOSITORY_DETECTIVE_ENABLE_TRIVY", "true")

	v := viper.New()
	v.SetDefault("enable_trivy", true)
	envcompat.Apply(v, logrus.New())

	if !v.GetBool("enable_trivy") {
		t.Fatal("only REPOSITORY_DETECTIVE_* must apply; unknown legacy prefixes are ignored")
	}
}

func TestResolveReturnsRepositoryDetectiveValue(t *testing.T) {
	t.Setenv("REPOSITORY_DETECTIVE_CORE_URL", "https://new.example")

	value, ok := envcompat.Resolve("CORE_URL")
	if !ok || value != "https://new.example" {
		t.Fatalf("Resolve CORE_URL = %q, %v", value, ok)
	}
}

func TestEnvExampleUsesPreferredPrefix(t *testing.T) {
	got := envcompat.EnvExample("API_KEY")
	if got != "REPOSITORY_DETECTIVE_API_KEY" {
		t.Fatalf("EnvExample = %q", got)
	}
}
