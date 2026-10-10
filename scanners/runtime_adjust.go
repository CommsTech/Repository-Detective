package scanners

import (
	"github.com/sirupsen/logrus"
)

// ApplyRuntimeAvailability turns off configured scanners whose binaries are not on PATH.
// Trivy is optional when Grype is installed (overlapping dependency/container coverage).
func ApplyRuntimeAvailability(cfg Config, logger *logrus.Logger) Config {
	if cfg.EnableTrivy && !commandAvailable("trivy") {
		if cfg.EnableGrype && commandAvailable("grype") {
			if logger != nil {
				logger.Infof("[SCANNER:trivy] not installed — bypassed (grype is available for dependency scanning)")
			}
			cfg.EnableTrivy = false
		} else {
			if logger != nil {
				logger.Debugf("[SCANNER:trivy] not installed — skipping trivy (install trivy or enable grype for dependency scans)")
			}
			cfg.EnableTrivy = false
		}
	}
	return cfg
}

// TrivyBypassed reports whether trivy is configured but skipped because grype is available.
func TrivyBypassed(cfg Config) bool {
	return !cfg.EnableTrivy && commandAvailable("grype")
}
