package ai

import (
	"fmt"
	"strings"
)

// ProviderType identifies the AI backend.
type ProviderType string

const (
	ProviderOpenAI     ProviderType = "openai"
	ProviderAnthropic  ProviderType = "anthropic"
	ProviderOpenRouter ProviderType = "openrouter"
	ProviderOllama     ProviderType = "ollama"
	ProviderOpenWebUI  ProviderType = "openwebui"
	ProviderOpenClaw   ProviderType = "openclaw"
)

// Config holds AI provider settings.
type Config struct {
	Provider              ProviderType
	BaseURL               string
	APIKey                string
	Model                 string
	ExtraHeaders          map[string]string
	InsecureSkipTLSVerify bool
}

// LegacyConfig holds deprecated OpenWebUI-only settings for backward compatibility.
type LegacyConfig struct {
	OpenWebUIURL   string
	OpenWebUIToken string
	OpenWebUIModel string
}

var defaultBaseURLs = map[ProviderType]string{
	ProviderOpenAI:     "https://api.openai.com/v1",
	ProviderAnthropic:  "https://api.anthropic.com/v1",
	ProviderOpenRouter: "https://openrouter.ai/api/v1",
	ProviderOllama:     "http://127.0.0.1:11434/v1",
	ProviderOpenClaw:   "http://127.0.0.1:18789/v1",
}

var defaultModels = map[ProviderType]string{
	ProviderOpenAI:     "gpt-4o-mini",
	ProviderAnthropic:  "claude-3-5-haiku-latest",
	ProviderOpenRouter: "openai/gpt-4o-mini",
	ProviderOllama:     "llama3.2",
	ProviderOpenWebUI:  "default",
	ProviderOpenClaw:   "openclaw/default",
}

// ResolveConfig normalizes provider settings and applies legacy OpenWebUI fallbacks.
func ResolveConfig(cfg Config, legacy LegacyConfig) (Config, error) {
	if cfg.Provider == "" {
		if legacy.OpenWebUIURL != "" {
			cfg.Provider = ProviderOpenWebUI
			if cfg.BaseURL == "" {
				cfg.BaseURL = legacy.OpenWebUIURL
			}
			if cfg.APIKey == "" {
				cfg.APIKey = legacy.OpenWebUIToken
			}
			if cfg.Model == "" {
				cfg.Model = legacy.OpenWebUIModel
			}
		} else {
			cfg.Provider = ProviderOpenAI
		}
	}

	cfg.Provider = ProviderType(strings.ToLower(strings.TrimSpace(string(cfg.Provider))))
	if err := validateProvider(cfg.Provider); err != nil {
		return Config{}, err
	}

	if cfg.BaseURL == "" {
		if cfg.Provider == ProviderOpenWebUI {
			return Config{}, fmt.Errorf("ai_base_url or openwebui_url is required for openwebui provider")
		}
		cfg.BaseURL = defaultBaseURLs[cfg.Provider]
	}

	if cfg.Model == "" {
		cfg.Model = defaultModels[cfg.Provider]
	}

	if cfg.Provider == ProviderOpenRouter && cfg.ExtraHeaders == nil {
		cfg.ExtraHeaders = map[string]string{
			"HTTP-Referer": "https://git.commsnet.org/commstech/repository-detective",
			"X-Title":      "Repository Detective",
		}
	}

	return cfg, nil
}

func validateProvider(provider ProviderType) error {
	switch provider {
	case ProviderOpenAI, ProviderAnthropic, ProviderOpenRouter, ProviderOllama, ProviderOpenWebUI, ProviderOpenClaw:
		return nil
	default:
		return fmt.Errorf("unsupported ai_provider %q (supported: openai, anthropic, openrouter, ollama, openwebui, openclaw)", provider)
	}
}

func normalizeBaseURL(baseURL string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/")
}
