// Package edition provides Community vs Commercial feature gates (RD-COMMERCIAL-001).
package edition

import (
	"strings"
)

const (
	Community  = "community"
	Commercial = "commercial"
	Enterprise = "enterprise" // reserved; treated as commercial capability set for now
)

// Config is the runtime edition gate.
type Config struct {
	Name             string
	LicenseKey       string
	CommunityMaxRepos int
}

// Normalize returns a sanitized edition config with defaults.
func Normalize(name, licenseKey string, maxRepos int) Config {
	n := strings.ToLower(strings.TrimSpace(name))
	switch n {
	case Commercial, "team", "pro":
		n = Commercial
	case Enterprise:
		n = Enterprise
	default:
		n = Community
	}
	if maxRepos <= 0 {
		maxRepos = 10
	}
	key := strings.TrimSpace(licenseKey)
	// Empty key always forces community limits even if name says commercial
	// (operators must set a license key to unlock commercial).
	if n != Community && key == "" {
		n = Community
	}
	return Config{Name: n, LicenseKey: key, CommunityMaxRepos: maxRepos}
}

func (c Config) IsCommercial() bool {
	return c.Name == Commercial || c.Name == Enterprise
}

func (c Config) DisplayName() string {
	switch c.Name {
	case Commercial:
		return "Commercial"
	case Enterprise:
		return "Enterprise"
	default:
		return "Community"
	}
}

// MaxRepos returns the connected-repo limit (0 = unlimited).
func (c Config) MaxRepos() int {
	if c.IsCommercial() {
		return 0
	}
	return c.CommunityMaxRepos
}

// AllowsRemediationPR is true when commercial edition (or community with explicit
// operator override handled elsewhere). Community may still enable PR via config,
// but the UI surfaces a commercial upsell when gated.
func (c Config) AllowsAdvancedCalibrationReports() bool {
	return c.IsCommercial()
}

func (c Config) AllowsMultiUserRBAC() bool {
	return c.IsCommercial()
}

func (c Config) AllowsBrandedReports() bool {
	return c.IsCommercial()
}
