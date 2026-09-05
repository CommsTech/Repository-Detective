package openclaw

import "strings"

// Config controls optional AI recommendations (disabled by default).
// Preferred keys use ai_recommendations_*; openclaw_ai_* remain legacy aliases.
type Config struct {
	Enabled                 bool   `mapstructure:"ai_recommendations_enabled"`
	Provider                string `mapstructure:"ai_recommendations_provider"`
	Endpoint                string `mapstructure:"ai_recommendations_endpoint"`
	Model                   string `mapstructure:"ai_recommendations_model"`
	TimeoutSeconds          int    `mapstructure:"ai_recommendations_timeout_seconds"`
	MaxFindingsPerScan      int    `mapstructure:"ai_recommendations_max_findings_per_scan"`
	MaxTokensPerScan        int    `mapstructure:"ai_recommendations_max_tokens_per_scan"`
	SendSourceSnippets      bool   `mapstructure:"ai_recommendations_send_source_snippets"`
	SendFullFiles           bool   `mapstructure:"ai_recommendations_send_full_files"`
	RedactSecrets           bool   `mapstructure:"ai_recommendations_redact_secrets"`
	RedactPII               bool   `mapstructure:"ai_recommendations_redact_pii"`
	AllowPreinstall         bool   `mapstructure:"ai_recommendations_allow_preinstall"`
	AllowContainerScans     bool   `mapstructure:"ai_recommendations_allow_container_scans"`
	AllowRepoScans          bool   `mapstructure:"ai_recommendations_allow_repo_scans"`
	RequireOperatorApproval bool   `mapstructure:"ai_recommendations_require_operator_approval"`
	StorePrompts            bool   `mapstructure:"ai_recommendations_store_prompts"`
	StoreResponses          bool   `mapstructure:"ai_recommendations_store_responses"`
	AdvisoryOnly            bool   `mapstructure:"ai_recommendations_advisory_only"`
	UseCAHHarness           bool   `mapstructure:"ai_recommendations_use_cah_harness"`
	AutoAfterScan           bool   `mapstructure:"ai_recommendations_auto_after_scan"`

	LegacyEnabled                 bool   `mapstructure:"openclaw_ai_review_enabled"`
	LegacyEndpoint                string `mapstructure:"openclaw_ai_endpoint"`
	LegacyModel                   string `mapstructure:"openclaw_ai_model"`
	LegacyTimeoutSeconds          int    `mapstructure:"openclaw_ai_timeout_seconds"`
	LegacyMaxFindingsPerScan      int    `mapstructure:"openclaw_ai_max_findings_per_scan"`
	LegacyMaxTokensPerScan        int    `mapstructure:"openclaw_ai_max_tokens_per_scan"`
	LegacySendSourceSnippets      bool   `mapstructure:"openclaw_ai_send_source_snippets"`
	LegacySendFullFiles           bool   `mapstructure:"openclaw_ai_send_full_files"`
	LegacyRedactSecrets           bool   `mapstructure:"openclaw_ai_redact_secrets"`
	LegacyRedactPII               bool   `mapstructure:"openclaw_ai_redact_pii"`
	LegacyAllowPreinstall         bool   `mapstructure:"openclaw_ai_allow_preinstall"`
	LegacyAllowContainerScans     bool   `mapstructure:"openclaw_ai_allow_container_scans"`
	LegacyAllowRepoScans          bool   `mapstructure:"openclaw_ai_allow_repo_scans"`
	LegacyRequireOperatorApproval bool   `mapstructure:"openclaw_ai_require_operator_approval"`
	LegacyStorePrompts            bool   `mapstructure:"openclaw_ai_store_prompts"`
	LegacyStoreResponses          bool   `mapstructure:"openclaw_ai_store_responses"`
	LegacyAdvisoryOnly            bool   `mapstructure:"openclaw_ai_advisory_only"`

	CAH CAHConfig `mapstructure:",squash"`

	FallbackEndpoint string
	FallbackModel    string
	FallbackAPIKey   string
}

// DefaultConfig returns safe defaults (off, redacted, advisory-only).
func DefaultConfig() Config {
	return Config{
		Enabled:                 false,
		Provider:                "openclaw",
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
		UseCAHHarness:           true,
		AutoAfterScan:           true,
		CAH:                     DefaultCAHConfig(),
	}
}

func mergeBool(preferred, legacy bool) bool {
	return preferred || legacy
}

func mergeString(preferred, legacy string) string {
	if s := strings.TrimSpace(preferred); s != "" {
		return s
	}
	return strings.TrimSpace(legacy)
}

func mergeInt(preferred, legacy, def int) int {
	if preferred > 0 {
		return preferred
	}
	if legacy > 0 {
		return legacy
	}
	if def > 0 {
		return def
	}
	return preferred
}

// Normalized applies defaults and merges legacy OpenClaw keys.
func (c Config) Normalized() Config {
	out := c
	def := DefaultConfig()
	out.Enabled = mergeBool(out.Enabled, out.LegacyEnabled)
	out.Endpoint = mergeString(out.Endpoint, out.LegacyEndpoint)
	out.Model = mergeString(out.Model, out.LegacyModel)
	out.TimeoutSeconds = mergeInt(out.TimeoutSeconds, out.LegacyTimeoutSeconds, def.TimeoutSeconds)
	if out.MaxFindingsPerScan <= 0 {
		if out.LegacyMaxFindingsPerScan > 0 {
			out.MaxFindingsPerScan = out.LegacyMaxFindingsPerScan
		} else {
			out.MaxFindingsPerScan = def.MaxFindingsPerScan
		}
	}
	if out.MaxTokensPerScan == 0 && out.LegacyMaxTokensPerScan != 0 {
		out.MaxTokensPerScan = out.LegacyMaxTokensPerScan
	}
	if out.MaxTokensPerScan < 0 {
		out.MaxTokensPerScan = 0
	}
	out.SendSourceSnippets = out.SendSourceSnippets || out.LegacySendSourceSnippets
	out.SendFullFiles = out.SendFullFiles || out.LegacySendFullFiles
	if !out.LegacyRedactSecrets {
		// legacy explicit false ignored — secrets always redacted
	}
	out.RedactSecrets = true
	if !out.RedactPII && !out.LegacyRedactPII {
		out.RedactPII = def.RedactPII
	} else {
		out.RedactPII = out.RedactPII || out.LegacyRedactPII || def.RedactPII
	}
	out.AllowPreinstall = out.AllowPreinstall || out.LegacyAllowPreinstall
	if !out.LegacyAllowContainerScans && out.LegacyAllowContainerScans != out.AllowContainerScans {
		// keep preferred when legacy explicitly set false on fresh installs
	}
	if out.LegacyAllowContainerScans {
		out.AllowContainerScans = true
	}
	if out.LegacyAllowRepoScans {
		out.AllowRepoScans = true
	}
	out.RequireOperatorApproval = out.RequireOperatorApproval || out.LegacyRequireOperatorApproval || def.RequireOperatorApproval
	out.StorePrompts = out.StorePrompts || out.LegacyStorePrompts
	out.StoreResponses = out.StoreResponses || out.LegacyStoreResponses || def.StoreResponses
	out.AdvisoryOnly = true
	if strings.TrimSpace(out.Provider) == "" {
		out.Provider = def.Provider
	}
	if !out.UseCAHHarness {
		out.UseCAHHarness = def.UseCAHHarness
	}
	out.CAH = normalizeCAH(out.CAH, out)
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
