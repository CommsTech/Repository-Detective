package openclaw_test

import (
	"testing"

	"git.commsnet.org/commstech/bugbot/openclaw"
)

func TestDisabledByDefault(t *testing.T) {
	cfg := openclaw.DefaultConfig()
	if cfg.Enabled {
		t.Fatal("must be disabled by default")
	}
	if cfg.MaxTokensPerScan != 0 {
		t.Fatal("max tokens must be 0 by default")
	}
	if !cfg.CanInvoke() {
		return
	}
	t.Fatal("CanInvoke should be false when disabled")
}

func TestMaxTokensZeroBlocksCall(t *testing.T) {
	cfg := openclaw.DefaultConfig()
	cfg.Enabled = true
	cfg.FallbackEndpoint = "http://127.0.0.1:1/v1"
	if cfg.CanInvoke() {
		t.Fatal("max tokens 0 must block invoke")
	}
}

func TestMissingEndpointDisablesSafely(t *testing.T) {
	cfg := openclaw.DefaultConfig()
	cfg.Enabled = true
	cfg.MaxTokensPerScan = 1000
	if cfg.CanInvoke() {
		t.Fatal("missing endpoint must block invoke")
	}
}

func TestFullFilesBlocked(t *testing.T) {
	cfg := openclaw.DefaultConfig()
	cfg.Enabled = true
	cfg.MaxTokensPerScan = 1000
	cfg.FallbackEndpoint = "http://example/v1"
	cfg.SendFullFiles = true
	if cfg.CanInvoke() {
		t.Fatal("full files must block invoke")
	}
}

func TestPreinstallDisabledByDefault(t *testing.T) {
	cfg := openclaw.DefaultConfig()
	if cfg.AllowsScanType("preinstall") {
		t.Fatal("preinstall must be off by default")
	}
}
