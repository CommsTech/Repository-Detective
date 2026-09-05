package ai

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestResolveConfigLegacyOpenWebUI(t *testing.T) {
	cfg, err := ResolveConfig(Config{}, LegacyConfig{
		OpenWebUIURL:   "http://openwebui:8080",
		OpenWebUIToken: "token",
		OpenWebUIModel: "llama3",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Provider != ProviderOpenWebUI {
		t.Fatalf("expected openwebui provider, got %s", cfg.Provider)
	}
	if cfg.BaseURL != "http://openwebui:8080" {
		t.Fatalf("unexpected base URL: %s", cfg.BaseURL)
	}
	if cfg.Model != "llama3" {
		t.Fatalf("unexpected model: %s", cfg.Model)
	}
}

func TestResolveConfigOpenAI(t *testing.T) {
	cfg, err := ResolveConfig(Config{
		Provider: ProviderOpenAI,
		APIKey:   "sk-test",
		Model:    "gpt-4o",
	}, LegacyConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BaseURL != defaultBaseURLs[ProviderOpenAI] {
		t.Fatalf("unexpected base URL: %s", cfg.BaseURL)
	}
	if cfg.Model != "gpt-4o" {
		t.Fatalf("unexpected model: %s", cfg.Model)
	}
}

func TestResolveConfigUnsupportedProvider(t *testing.T) {
	_, err := ResolveConfig(Config{Provider: "unknown"}, LegacyConfig{})
	if err == nil {
		t.Fatal("expected unsupported provider error")
	}
}

func TestNewClientOpenRouter(t *testing.T) {
	logger := logrus.New()
	client, err := NewClient(Config{
		Provider: ProviderOpenRouter,
		APIKey:   "test-key",
		Model:    "anthropic/claude-3.5-sonnet",
	}, LegacyConfig{}, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.Provider() != ProviderOpenRouter {
		t.Fatalf("expected openrouter, got %s", client.Provider())
	}
}

func TestNewClientRequiresAnthropicKey(t *testing.T) {
	_, err := NewClient(Config{Provider: ProviderAnthropic}, LegacyConfig{}, logrus.New())
	if err == nil {
		t.Fatal("expected missing API key error")
	}
}

func TestNewClientOpenClawDefaults(t *testing.T) {
	client, err := NewClient(Config{Provider: ProviderOpenClaw}, LegacyConfig{}, logrus.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.Model() != "openclaw/default" {
		t.Fatalf("expected openclaw/default model, got %s", client.Model())
	}
}
