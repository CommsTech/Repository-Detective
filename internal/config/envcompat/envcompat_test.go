package envcompat_test

import (
	"os"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/internal/config/envcompat"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/spf13/viper"
)

func TestNewPrefixWinsOverLegacy(t *testing.T) {
	t.Setenv("BUGBOT_ENABLE_SEMGREP", "false")
	t.Setenv("REPOSITORY_DETECTIVE_ENABLE_SEMGREP", "true")

	v := viper.New()
	v.SetDefault("enable_semgrep", false)
	envcompat.Apply(v, logrus.New())

	if !v.GetBool("enable_semgrep") {
		t.Fatal("expected REPOSITORY_DETECTIVE_ENABLE_SEMGREP to win")
	}
}

func TestLegacyAPIKeyAppliedWhenNewPrefixUnset(t *testing.T) {
	t.Setenv("REPOSITORY_DETECTIVE_API_KEY", "")
	os.Unsetenv("REPOSITORY_DETECTIVE_API_KEY")
	t.Setenv("BUGBOT_API_KEY", "legacy-test-key")

	v := viper.New()
	envcompat.Apply(v, logrus.New())

	if got := v.GetString("api_key"); got != "legacy-test-key" {
		t.Fatalf("expected legacy api_key, got %q", got)
	}
}

func TestLegacyPrefixStillWorks(t *testing.T) {
	os.Unsetenv("REPOSITORY_DETECTIVE_ENABLE_TRIVY")
	t.Setenv("BUGBOT_ENABLE_TRIVY", "false")

	v := viper.New()
	v.SetDefault("enable_trivy", true)
	envcompat.Apply(v, logrus.New())

	if v.GetBool("enable_trivy") {
		t.Fatal("expected BUGBOT_ENABLE_TRIVY=false to apply")
	}
}

func TestResolvePrefersNewPrefix(t *testing.T) {
	t.Setenv("BUGBOT_CORE_URL", "https://legacy.example")
	t.Setenv("REPOSITORY_DETECTIVE_CORE_URL", "https://new.example")

	value, ok := envcompat.Resolve("CORE_URL")
	if !ok || value != "https://new.example" {
		t.Fatalf("Resolve() = %q ok=%v", value, ok)
	}
}

func TestConflictLogsNewPrefixWin(t *testing.T) {
	t.Setenv("BUGBOT_PORT", "8080")
	t.Setenv("REPOSITORY_DETECTIVE_PORT", "9090")

	logger, hook := test.NewNullLogger()
	v := viper.New()
	envcompat.Apply(v, logger)

	if v.GetString("port") != "9090" {
		t.Fatalf("expected new port, got %q", v.GetString("port"))
	}
	found := false
	for _, entry := range hook.AllEntries() {
		if strings.Contains(entry.Message, "REPOSITORY_DETECTIVE_PORT") {
			found = true
		}
	}
	if !found {
		t.Fatal("expected conflict log when both prefixes set")
	}
}
