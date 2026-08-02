package preinstall_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/preinstall"
	"git.commsnet.org/commstech/repository-detective/scanners"
)

func TestScannerConfigForDepthQuickSkipsGoScanners(t *testing.T) {
	base := scanners.Config{
		EnableGovulncheck: true,
		EnableGosec:       true,
		EnableStaticcheck: true,
		EnableCheckov:     true,
	}
	cfg := preinstall.ScannerConfigForDepthForTest("quick", base)
	if cfg.EnableGovulncheck || cfg.EnableGosec || cfg.EnableStaticcheck {
		t.Fatal("quick audit should skip Go scanner trio")
	}
	if cfg.EnableCheckov {
		t.Fatal("quick audit should skip checkov")
	}
}

func TestScannerConfigForDepthStandardKeepsGoScanners(t *testing.T) {
	base := scanners.Config{
		EnableGovulncheck: true,
		EnableGosec:       true,
		EnableStaticcheck: true,
		EnableHadolint:    true,
		EnableCheckov:     true,
	}
	cfg := preinstall.ScannerConfigForDepthForTest("standard", base)
	if !cfg.EnableGovulncheck || !cfg.EnableGosec || !cfg.EnableStaticcheck {
		t.Fatal("standard audit should keep enabled Go scanners")
	}
	if !cfg.EnableHadolint || !cfg.EnableCheckov {
		t.Fatal("standard audit should keep enabled IaC scanners")
	}
}
