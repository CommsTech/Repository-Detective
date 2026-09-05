package operator

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
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
	Name            string `json:"name"`
	Configured      bool   `json:"configured"`
	EnabledInConfig bool   `json:"enabled_in_config"`
	BinaryInstalled bool   `json:"binary_installed"`
	Available       bool   `json:"available"`
	StatusState     string `json:"status_state"`
	Action          string `json:"action,omitempty"`
	Version         string `json:"version,omitempty"`
	LastChecked     string `json:"last_checked"`
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
	{name: "syft", binary: "syft", versionArg: []string{"version"}, always: true},
	{name: "cyclonedx-gomod", binary: "cyclonedx-gomod", versionArg: []string{"version"}, always: true},
	{name: "gitleaks", binary: "gitleaks", versionArg: []string{"version"}, enabled: func(c ScannerConfig) bool { return c.EnableGitleaks }},
	{name: "semgrep", binary: "semgrep", versionArg: []string{"--version"}, enabled: func(c ScannerConfig) bool { return c.EnableSemgrep }},
	{name: "govulncheck", binary: "govulncheck", versionArg: []string{"-version"}, enabled: func(c ScannerConfig) bool { return c.EnableGovulncheck }},
	{name: "gosec", binary: "gosec", versionArg: []string{"-version"}, enabled: func(c ScannerConfig) bool { return c.EnableGosec }},
	{name: "staticcheck", binary: "staticcheck", versionArg: []string{"-version"}, enabled: func(c ScannerConfig) bool { return c.EnableStaticcheck }},
	{name: "hadolint", binary: "hadolint", versionArg: []string{"--version"}, enabled: func(c ScannerConfig) bool { return c.EnableHadolint }},
	{name: "checkov", binary: "checkov", versionArg: []string{"--version"}, enabled: func(c ScannerConfig) bool { return c.EnableCheckov }},
}

var (
	toolsCacheMu  sync.Mutex
	toolsCacheAt  time.Time
	toolsCacheKey string
	toolsCacheOut []ToolStatus
	toolsInflight sync.Mutex // serialize uncached probes so health/UI don't stampede
)

const toolsCacheTTL = 5 * time.Minute

// CheckTools probes PATH for configured scanner binaries and records version strings.
// Results are cached (5m) with singleflight so health/UI pages do not re-exec scanners on every request.
func CheckTools(cfg ScannerConfig) []ToolStatus {
	key := scannerConfigCacheKey(cfg)
	toolsCacheMu.Lock()
	if toolsCacheOut != nil && toolsCacheKey == key && time.Since(toolsCacheAt) < toolsCacheTTL {
		out := cloneToolStatuses(toolsCacheOut)
		toolsCacheMu.Unlock()
		return out
	}
	toolsCacheMu.Unlock()

	// Only one goroutine probes at a time; others wait then reuse the fresh cache.
	toolsInflight.Lock()
	defer toolsInflight.Unlock()

	toolsCacheMu.Lock()
	if toolsCacheOut != nil && toolsCacheKey == key && time.Since(toolsCacheAt) < toolsCacheTTL {
		out := cloneToolStatuses(toolsCacheOut)
		toolsCacheMu.Unlock()
		return out
	}
	toolsCacheMu.Unlock()

	out := checkToolsUncached(cfg)

	toolsCacheMu.Lock()
	toolsCacheAt = time.Now()
	toolsCacheKey = key
	toolsCacheOut = cloneToolStatuses(out)
	toolsCacheMu.Unlock()
	return out
}

// InvalidateToolsCache clears the scanner probe cache (e.g. after config changes).
func InvalidateToolsCache() {
	toolsCacheMu.Lock()
	toolsCacheOut = nil
	toolsCacheKey = ""
	toolsCacheAt = time.Time{}
	toolsCacheMu.Unlock()
}

func scannerConfigCacheKey(cfg ScannerConfig) string {
	return fmt.Sprintf("%v|%v|%v|%v|%v|%v|%v|%v|%v|%v",
		cfg.EnableTrivy, cfg.EnableGrype, cfg.EnableGitleaks, cfg.EnableSemgrep,
		cfg.EnableGovulncheck, cfg.EnableGosec, cfg.EnableStaticcheck, cfg.EnableHadolint,
		cfg.EnableCheckov, cfg.EnableLinters)
}

func cloneToolStatuses(in []ToolStatus) []ToolStatus {
	out := make([]ToolStatus, len(in))
	copy(out, in)
	return out
}

func checkToolsUncached(cfg ScannerConfig) []ToolStatus {
	now := time.Now().UTC().Format(time.RFC3339)
	out := make([]ToolStatus, len(toolDefs))
	var wg sync.WaitGroup

	for i, def := range toolDefs {
		enabled := def.always
		if def.enabled != nil {
			enabled = def.enabled(cfg)
		}
		installed := lookPath(def.binary)
		state, action := resolveToolState(def.name, enabled, installed, def.always)
		out[i] = ToolStatus{
			Name:            def.name,
			Configured:      enabled || def.always,
			EnabledInConfig: enabled || def.always,
			BinaryInstalled: installed,
			Available:       enabled && installed,
			StatusState:     state,
			Action:          action,
			Version:         "",
			LastChecked:     now,
		}
		if !installed {
			continue
		}
		wg.Add(1)
		go func(idx int, binary string, args []string) {
			defer wg.Done()
			out[idx].Version = probeVersion(binary, args)
		}(i, def.binary, def.versionArg)
	}
	wg.Wait()
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
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run() // some tools print a version then exit non-zero; still capture output
	return pickVersionLine(stdout.String(), stderr.String())
}

func pickVersionLine(stdout, stderr string) string {
	prefer := func(blob string) string {
		lines := strings.Split(blob, "\n")
		var firstUseful string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			lower := strings.ToLower(line)
			if strings.Contains(lower, "new version of") || strings.Contains(lower, "see https://") {
				continue
			}
			if strings.HasPrefix(lower, "scanner:") || strings.HasPrefix(lower, "version:") {
				if len(line) > 120 {
					return line[:120] + "…"
				}
				return line
			}
			if firstUseful == "" {
				firstUseful = line
			}
		}
		if len(firstUseful) > 120 {
			return firstUseful[:120] + "…"
		}
		return firstUseful
	}
	if v := prefer(stdout); v != "" {
		return v
	}
	return prefer(stderr)
}
