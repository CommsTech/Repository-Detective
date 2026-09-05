package ai

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// NewTransport builds the correct transport for a resolved provider config.
func NewTransport(cfg Config, logger *logrus.Logger) (ChatTransport, error) {
	switch cfg.Provider {
	case ProviderAnthropic:
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("ai_api_key is required for anthropic provider")
		}
		return NewAnthropicTransport(cfg.BaseURL, cfg.APIKey, cfg.InsecureSkipTLSVerify, logger), nil
	case ProviderOpenAI:
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("ai_api_key is required for openai provider")
		}
		return NewOpenAICompatibleTransport(string(ProviderOpenAI), cfg.BaseURL, cfg.APIKey, cfg.ExtraHeaders, cfg.InsecureSkipTLSVerify, logger), nil
	case ProviderOpenRouter:
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("ai_api_key is required for openrouter provider")
		}
		return NewOpenAICompatibleTransport(string(ProviderOpenRouter), cfg.BaseURL, cfg.APIKey, cfg.ExtraHeaders, cfg.InsecureSkipTLSVerify, logger), nil
	case ProviderOllama:
		return NewOpenAICompatibleTransport(string(ProviderOllama), cfg.BaseURL, cfg.APIKey, cfg.ExtraHeaders, cfg.InsecureSkipTLSVerify, logger), nil
	case ProviderOpenWebUI:
		return NewOpenAICompatibleTransport(string(ProviderOpenWebUI), cfg.BaseURL, cfg.APIKey, cfg.ExtraHeaders, cfg.InsecureSkipTLSVerify, logger), nil
	case ProviderOpenClaw:
		return NewOpenAICompatibleTransport(string(ProviderOpenClaw), cfg.BaseURL, cfg.APIKey, cfg.ExtraHeaders, cfg.InsecureSkipTLSVerify, logger), nil
	default:
		return nil, fmt.Errorf("unsupported provider %q", cfg.Provider)
	}
}

// NewClient creates a fully configured AI client for the CAH pipeline.
func NewClient(cfg Config, legacy LegacyConfig, logger *logrus.Logger) (*Client, error) {
	resolved, err := ResolveConfig(cfg, legacy)
	if err != nil {
		return nil, err
	}

	transport, err := NewTransport(resolved, logger)
	if err != nil {
		return nil, err
	}

	logger.Infof("AI provider: %s (model=%s, transport=%s)", resolved.Provider, resolved.Model, transport.Name())

	return &Client{
		transport: transport,
		provider:  resolved.Provider,
		model:     resolved.Model,
		logger:    logger,
	}, nil
}

// NewClientWithTransport is primarily for tests.
func NewClientWithTransport(transport ChatTransport, model string, logger *logrus.Logger) *Client {
	return &Client{
		transport: transport,
		provider:  ProviderOpenAI,
		model:     model,
		logger:    logger,
	}
}

// Provider returns the configured provider type.
func (c *Client) Provider() ProviderType {
	return c.provider
}

// Model returns the configured model name.
func (c *Client) Model() string {
	return c.modelName()
}
