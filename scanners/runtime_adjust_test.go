package scanners_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/scanners"
	"github.com/sirupsen/logrus"
)

func TestApplyRuntimeAvailabilitySkipsMissingTrivyWhenGrypePresent(t *testing.T) {
	if !scanners.CommandAvailableForTest("grype") {
		t.Skip("grype not installed")
	}
	if scanners.CommandAvailableForTest("trivy") {
		t.Skip("trivy installed")
	}
	cfg := scanners.ApplyRuntimeAvailability(scanners.Config{
		EnableTrivy: true,
		EnableGrype: true,
	}, logrus.New())
	if cfg.EnableTrivy {
		t.Fatal("expected trivy disabled when binary missing and grype present")
	}
}
