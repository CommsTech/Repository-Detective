package openclaw

import "strings"

// Config controls optional OpenClaw advisory review (disabled by default).
type Config struct {
	Enabled                 bool   `mapstructure:"openclaw_ai_review_enabled"`
	Endpoint                string `mapstructure:"openclaw_ai_endpoint"`
	Model                   string `mapstructure:"openclaw_ai_model"`
	TimeoutSeconds          int    `mapstructure:"openclaw_ai_timeout_seconds"`
	MaxFindingsPerScan      int    `mapstructure:"openclaw_ai_max_findings_per_scan"`
	MaxTokensPerScan        int    `mapstructure:"openclaw_ai_max_tokens_per_scan"`
	SendSourceSnippets      bool   `mapstructure:"openclaw_ai_send_source_snippets"`
	SendFullFiles           bool   `mapstructure:"openclaw_ai_send_full_files"`
	RedactSecrets           bool   `mapstructure:"openclaw_ai_redact_secrets"`
	RedactPII               bool   `mapstructure:"openclaw_ai_redact_pii"`
	AllowPreinstall         bool   `mapstructure:"openclaw_ai_allow_preinstall"`
	AllowContainerScans     bool   `mapstructure:"openclaw_ai_allow_container_scans"`
	AllowRepoScans          bool   `mapstructure:"openclaw_ai_allow_repo_scans"`
	RequireOperatorApproval bool   `mapstructure:"openclaw_ai_require_operator_approval"`
	StorePrompts            bool   `mapstructure:"openclaw_ai_store_prompts"`
	StoreResponses          bool   `mapstructure:"openclaw_ai_store_responses"`
	AdvisoryOnly            bool   `mapstructure:"openclaw_ai_advisory_only"`

	// Runtime fallbacks from core AI settings (not mapstructure).
	FallbackEndpoint string
	FallbackModel    string
	FallbackAPIKey   string
}

// DefaultConfig returns safe defaults (off, redacted, advisory-only).
func DefaultConfig() Config {
	return Config{
		Enabled:                 false,
		TimeoutSeconds:          60,
		MaxFindingsPerScan:      25,
		MaxTokensPerScan:        0,
		SendSourceSnippets:      false,
		SendFullFiles:           false,
		RedactSecrets:           true,
		RedactPII:               true,
		AllowPreinstall:         false,
		AllowContainerScans:     true,
		AllowRepoScans:          true,
		RequireOperatorApproval: true,
		StorePrompts:            false,
		StoreResponses:          true,
		AdvisoryOnly:            true,
	}
}

// Normalized applies defaults to zero values.
func (c Config) Normalized() Config {
	out := c
	def := DefaultConfig()
	if out.TimeoutSeconds <= 0 {
		out.TimeoutSeconds = def.TimeoutSeconds
	}
	if out.MaxFindingsPerScan <= 0 {
		out.MaxFindingsPerScan = def.MaxFindingsPerScan
	}
	if out.MaxTokensPerScan < 0 {
		out.MaxTokensPerScan = 0
	}
	// Policy-enforced: secrets always redacted; advisory-only always on.
	out.RedactSecrets = true
	if !out.RedactPII {
		out.RedactPII = def.RedactPII
	}
	out.AdvisoryOnly = true
	return out
}

// EffectiveEndpoint returns configured endpoint or core AI base URL.
func (c Config) EffectiveEndpoint() string {
	if ep := strings.TrimSpace(c.Endpoint); ep != "" {
		return ep
	}
	return strings.TrimSpace(c.FallbackEndpoint)
}

// EffectiveModel returns configured model or core AI model.
func (c Config) EffectiveModel() string {
	if m := strings.TrimSpace(c.Model); m != "" {
		return m
	}
	return strings.TrimSpace(c.FallbackModel)
}

// EndpointConfigured reports whether a review endpoint is available.
func (c Config) EndpointConfigured() bool {
	return strings.TrimSpace(c.EffectiveEndpoint()) != ""
}

// CanInvoke reports whether a review call may be attempted.
func (c Config) CanInvoke() bool {
	c = c.Normalized()
	if !c.Enabled {
		return false
	}
	if !c.EndpointConfigured() {
		return false
	}
	if c.MaxTokensPerScan <= 0 {
		return false
	}
	if !c.AdvisoryOnly {
		return false
	}
	if c.SendFullFiles {
		return false
	}
	if !c.RedactSecrets {
		return false
	}
	return true
}

// AllowsScanType reports whether review is permitted for a scan type.
func (c Config) AllowsScanType(scanType string) bool {
	c = c.Normalized()
	switch strings.ToLower(strings.TrimSpace(scanType)) {
	case "preinstall":
		return c.AllowPreinstall
	case "container":
		return c.AllowContainerScans
	default:
		return c.AllowRepoScans
	}
}
