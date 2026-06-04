package operator

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Scanner status state constants for operator clarity.
const (
	StatusDisabledByConfig     = "disabled_by_config"
	StatusEnabledMissingBinary = "enabled_missing_binary"
	StatusEnabledAvailable     = "enabled_available"
	StatusInstalledButDisabled = "installed_but_disabled"
	StatusNotApplicable        = "not_applicable"
	StatusNotChecked           = "not_checked"
)

// ToolStatus describes whether an external tool binary is configured and available.
type ToolStatus struct {
	Name              string `json:"name"`
	Configured        bool   `json:"configured"`
	EnabledInConfig   bool   `json:"enabled_in_config"`
	BinaryInstalled   bool   `json:"binary_installed"`
	Available         bool   `json:"available"`
	StatusState       string `json:"status_state"`
	Action            string `json:"action,omitempty"`
	Version           string `json:"version,omitempty"`
	LastChecked       string `json:"last_checked"`
}

// ScannerConfig toggles which tools are enabled in configuration.
type ScannerConfig struct {
	EnableTrivy       bool
	EnableGrype       bool
	EnableGitleaks    bool
	EnableSemgrep     bool
	EnableGovulncheck bool
	EnableGosec       bool
	EnableStaticcheck bool
	EnableHadolint    bool
	EnableCheckov     bool
	EnableLinters     bool
	PreinstallGit     bool
	RemediationGit    bool
}

var toolDefs = []struct {
	name       string
	binary     string
	versionArg []string
	enabled    func(ScannerConfig) bool
	always     bool
}{
	{name: "git", binary: "git", versionArg: []string{"--version"}, always: true},
	{name: "trivy", binary: "trivy", versionArg: []string{"--version"}, enabled: func(c ScannerConfig) bool { return c.EnableTrivy }},
	{name: "grype", binary: "grype", versionArg: []string{"version"}, enabled: func(c ScannerConfig) bool { return c.EnableGrype }},
	{name: "gitleaks", binary: "gitleaks", versionArg: []string{"version"}, enabled: func(c ScannerConfig) bool { return c.EnableGitleaks }},
	{name: "semgrep", binary: "semgrep", versionArg: []string{"--version"}, enabled: func(c ScannerConfig) bool { return c.EnableSemgrep }},
	{name: "govulncheck", binary: "govulncheck", versionArg: []string{"-version"}, enabled: func(c ScannerConfig) bool { return c.EnableGovulncheck }},
	{name: "gosec", binary: "gosec", versionArg: []string{"-version"}, enabled: func(c ScannerConfig) bool { return c.EnableGosec }},
	{name: "staticcheck", binary: "staticcheck", versionArg: []string{"-version"}, enabled: func(c ScannerConfig) bool { return c.EnableStaticcheck }},
	{name: "hadolint", binary: "hadolint", versionArg: []string{"--version"}, enabled: func(c ScannerConfig) bool { return c.EnableHadolint }},
	{name: "checkov", binary: "checkov", versionArg: []string{"--version"}, enabled: func(c ScannerConfig) bool { return c.EnableCheckov }},
}

// CheckTools probes PATH for configured scanner binaries.
func CheckTools(cfg ScannerConfig) []ToolStatus {
	now := time.Now().UTC().Format(time.RFC3339)
	out := make([]ToolStatus, 0, len(toolDefs))
	for _, def := range toolDefs {
		enabled := def.always
		if def.enabled != nil {
			enabled = def.enabled(cfg)
		}
		installed := lookPath(def.binary)
		version := ""
		if installed && len(def.versionArg) > 0 {
			version = probeVersion(def.binary, def.versionArg)
		}
		state, action := resolveToolState(def.name, enabled, installed, def.always)
		out = append(out, ToolStatus{
			Name:            def.name,
			Configured:      enabled || def.always,
			EnabledInConfig: enabled || def.always,
			BinaryInstalled: installed,
			Available:       enabled && installed,
			StatusState:     state,
			Action:          action,
			Version:         version,
			LastChecked:     now,
		})
	}
	return out
}

func resolveToolState(name string, enabled, installed, always bool) (state, action string) {
	if always {
		if installed {
			return StatusEnabledAvailable, ""
		}
		return StatusEnabledMissingBinary, fmt.Sprintf("Install %s in the core/all-in-one image or runner image.", name)
	}
	if enabled && installed {
		return StatusEnabledAvailable, ""
	}
	if enabled && !installed {
		return StatusEnabledMissingBinary, fmt.Sprintf("Install %s in the core/all-in-one image or runner image, or disable it in scan profile/repo settings.", name)
	}
	if !enabled && installed {
		return StatusInstalledButDisabled, fmt.Sprintf("Enable %s in repo settings or scan profile if you want this scanner active.", name)
	}
	return StatusDisabledByConfig, fmt.Sprintf("Enable %s in repo settings/profile if desired.", name)
}

func lookPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func probeVersion(binary string, args []string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, binary, args...).CombinedOutput()
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(strings.Split(string(out), "\n")[0])
	if len(line) > 120 {
		line = line[:120] + "…"
	}
	return line
}
