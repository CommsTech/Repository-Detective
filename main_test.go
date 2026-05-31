package main

import (
	"testing"

	"github.com/spf13/viper"
)

func TestNeedsAIProvider(t *testing.T) {
	cfg := &Config{
		AnalysisDepth:     1,
		EnableLLMAuditors: false,
		QdrantEnabled:     false,
	}
	if cfg.needsAIProvider() {
		t.Fatal("deterministic-only config should not require AI")
	}

	cfg.AnalysisDepth = 3
	cfg.EnableLLMAuditors = false
	if cfg.needsAIProvider() {
		t.Fatal("depth 3 with LLM auditors disabled should not require AI")
	}

	cfg.EnableLLMAuditors = true
	if !cfg.needsAIProvider() {
		t.Fatal("depth 3 with LLM auditors enabled should require AI")
	}

	cfg.EnableLLMAuditors = false
	cfg.QdrantEnabled = true
	if !cfg.needsAIProvider() {
		t.Fatal("Qdrant enabled should require AI for embeddings")
	}
}

func TestGiteaStatusConfigDefaults(t *testing.T) {
	v := viper.New()
	v.SetDefault("enable_gitea_status", false)
	v.SetDefault("gitea_status_context", "bugbot/security-scan")
	v.SetDefault("gitea_status_fail_on", "high")
	v.SetDefault("gitea_status_warn_on", "medium")
	v.SetDefault("gitea_status_include_scanner_failures", true)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if cfg.EnableGiteaStatus {
		t.Fatal("expected enable_gitea_status default false")
	}
	if cfg.GiteaStatusContext != "bugbot/security-scan" {
		t.Fatalf("unexpected context default %q", cfg.GiteaStatusContext)
	}
	if cfg.GiteaStatusFailOn != "high" || cfg.GiteaStatusWarnOn != "medium" {
		t.Fatalf("unexpected severity defaults fail=%q warn=%q", cfg.GiteaStatusFailOn, cfg.GiteaStatusWarnOn)
	}
	if !cfg.GiteaStatusIncludeScannerFailures {
		t.Fatal("expected scanner failure inclusion default true")
	}
}
