package operator

import "strings"

// InstallState maps StatusState to legacy UI values.
func (t ToolStatus) InstallState() string {
	switch t.StatusState {
	case StatusEnabledAvailable:
		return "available"
	case StatusEnabledMissingBinary:
		return "missing"
	case StatusDisabledByConfig, StatusInstalledButDisabled, StatusNotApplicable:
		return "disabled"
	case StatusNotChecked:
		return "unknown"
	}
	if t.EnabledInConfig || t.Configured {
		if t.BinaryInstalled || t.Available {
			return "available"
		}
		return "missing"
	}
	return "disabled"
}

func (t ToolStatus) IsOptional() bool {
	return !(t.EnabledInConfig || t.Configured) && t.Name != "git"
}

func GrypeAvailable(tools []ToolStatus) bool {
	for _, tool := range tools {
		if tool.Name == "grype" && (tool.BinaryInstalled || tool.Available) {
			return true
		}
	}
	return false
}

func TrivyBypassedByGrype(tool ToolStatus, tools []ToolStatus) bool {
	enabled := tool.EnabledInConfig || tool.Configured
	installed := tool.BinaryInstalled || tool.Available
	return tool.Name == "trivy" && enabled && !installed && GrypeAvailable(tools)
}

func (t ToolStatus) IsRequiredInProfile() bool {
	return t.EnabledInConfig || t.Configured
}

func (t ToolStatus) VersionDisplay() string {
	v := strings.TrimSpace(t.Version)
	if v != "" {
		return v
	}
	if !t.EnabledInConfig && !t.Configured {
		return "—"
	}
	if t.BinaryInstalled || t.Available {
		if strings.TrimSpace(t.Version) == "" {
			return "unknown"
		}
		return v
	}
	return "—"
}

func (t ToolStatus) CoverageImpact() string {
	if !t.EnabledInConfig && !t.Configured {
		return "inactive"
	}
	if t.BinaryInstalled || t.Available {
		return "none"
	}
	return "degraded"
}

func (t ToolStatus) RemediationHint() string {
	if strings.TrimSpace(t.Action) != "" {
		return t.Action
	}
	switch t.StatusState {
	case StatusDisabledByConfig:
		return t.Name + " is disabled by configuration."
	case StatusInstalledButDisabled:
		return t.Name + " is installed but disabled in the effective scan profile."
	case StatusEnabledMissingBinary:
		return t.Name + " is enabled but the binary is not on PATH."
	case StatusEnabledAvailable:
		if strings.TrimSpace(t.Version) == "" {
			return "Installed; version could not be parsed from scanner output."
		}
		return ""
	}
	if !t.EnabledInConfig && !t.Configured {
		switch t.Name {
		case "hadolint":
			return "hadolint is not configured; Dockerfile linting is currently skipped."
		case "checkov":
			return "checkov is not configured; IaC scanning is currently skipped."
		default:
			return t.Name + " is not configured in scanner settings (optional)."
		}
	}
	if t.BinaryInstalled || t.Available {
		if strings.TrimSpace(t.Version) == "" {
			return "Installed; version could not be parsed from scanner output."
		}
		return ""
	}
	if t.Name == "trivy" {
		return "trivy is not installed. Optional when grype is available; otherwise install trivy or set enable_trivy: false."
	}
	return t.Name + " is configured but not installed. Install " + t.Name + " in PATH or disable it in scanner settings."
}

// CountEnabledMissing returns tools enabled in config but missing from PATH.
func CountEnabledMissing(tools []ToolStatus) int {
	n := 0
	for _, t := range tools {
		if t.StatusState == StatusEnabledMissingBinary || ((t.EnabledInConfig || t.Configured) && !(t.BinaryInstalled || t.Available)) {
			n++
		}
	}
	return n
}
