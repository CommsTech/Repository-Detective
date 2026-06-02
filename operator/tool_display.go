package operator

import "strings"

// InstallState describes runtime install status for UI and APIs.
// Values: available, missing, disabled, unknown.
func (t ToolStatus) InstallState() string {
	if !t.Configured {
		return "disabled"
	}
	if t.Available {
		return "available"
	}
	if t.LastChecked != "" {
		return "missing"
	}
	return "unknown"
}

// IsOptional reports scanners not enabled in the effective profile (excluding git).
func (t ToolStatus) IsOptional() bool {
	return !t.Configured && t.Name != "git"
}

// IsRequiredInProfile reports scanners expected for scans per configuration.
func (t ToolStatus) IsRequiredInProfile() bool {
	return t.Configured
}

// VersionDisplay returns a safe version label for operator tables.
func (t ToolStatus) VersionDisplay() string {
	v := strings.TrimSpace(t.Version)
	if v != "" {
		return v
	}
	if !t.Configured {
		return "—"
	}
	if t.Available {
		return "unknown"
	}
	return "—"
}

// CoverageImpact classifies operator impact: none, degraded, inactive.
func (t ToolStatus) CoverageImpact() string {
	if !t.Configured {
		return "inactive"
	}
	if t.Available {
		return "none"
	}
	return "degraded"
}

// RemediationHint returns short operator guidance (no secrets).
func (t ToolStatus) RemediationHint() string {
	switch {
	case !t.Configured:
		switch t.Name {
		case "hadolint":
			return "hadolint is not configured; Dockerfile linting is currently skipped."
		case "checkov":
			return "checkov is not configured; IaC scanning is currently skipped."
		default:
			return t.Name + " is not configured in scanner settings (optional)."
		}
	case t.Available:
		if strings.TrimSpace(t.Version) == "" {
			return "Installed; version could not be parsed from scanner output."
		}
		return ""
	case t.Name == "trivy":
		return "trivy is configured but not installed. Install trivy or disable enable_trivy in config."
	default:
		return t.Name + " is configured but not installed. Install " + t.Name + " in PATH or disable it in scanner settings."
	}
}
