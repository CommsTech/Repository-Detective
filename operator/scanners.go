package operator

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// ToolStatus describes whether an external tool binary is configured and available.
type ToolStatus struct {
	Name        string `json:"name"`
	Configured  bool   `json:"configured"`
	Available   bool   `json:"available"`
	Version     string `json:"version,omitempty"`
	LastChecked string `json:"last_checked"`
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
		configured := def.always
		if def.enabled != nil {
			configured = def.enabled(cfg)
		}
		if !configured && !def.always {
			out = append(out, ToolStatus{Name: def.name, Configured: false, Available: false, LastChecked: now})
			continue
		}
		available := lookPath(def.binary)
		version := ""
		if available && len(def.versionArg) > 0 {
			version = probeVersion(def.binary, def.versionArg)
		}
		out = append(out, ToolStatus{
			Name: def.name, Configured: configured || def.always,
			Available: available, Version: version, LastChecked: now,
		})
	}
	return out
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
