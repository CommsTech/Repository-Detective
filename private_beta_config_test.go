package main_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// privateBetaConfig mirrors keys we enforce for private beta safety.
type privateBetaConfig struct {
	APIKey                  string `yaml:"api_key"`
	GiteaToken              string `yaml:"gitea_token"`
	WebhookSecret           string `yaml:"webhook_secret"`
	AIAPIKey                string `yaml:"ai_api_key"`
	AutoCreateIssues        bool   `yaml:"auto_create_issues"`
	RemediationPREnabled    bool   `yaml:"remediation_pr_enabled"`
	RunnerDelegationEnabled bool   `yaml:"runner_delegation_enabled"`
	NotificationsEnabled    bool   `yaml:"notifications_enabled"`
	LLMSanityGateEnabled    bool   `yaml:"llm_sanity_gate_enabled"`
	EnableLLMAuditors       bool   `yaml:"enable_llm_auditors"`
	EvidenceClosureEnabled  bool   `yaml:"evidence_closure_enabled"`
	BacklogControlEnabled   bool   `yaml:"dogfood_backlog_control_enabled"`
	ScanProfile             string `yaml:"scan_profile"`
	Reporting               struct {
		MaxIssuesPerScan int `yaml:"max_issues_per_scan"`
	} `yaml:"reporting"`
}

func loadPrivateBetaExample(t *testing.T) privateBetaConfig {
	t.Helper()
	root := findRepoRoot(t)
	path := filepath.Join(root, "config", "private-beta.example.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read private beta config: %v", err)
	}
	var cfg privateBetaConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("parse private beta config: %v", err)
	}
	return cfg
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func TestPrivateBetaExampleLoads(t *testing.T) {
	cfg := loadPrivateBetaExample(t)
	if cfg.ScanProfile == "" {
		t.Fatal("scan_profile must be set")
	}
}

func TestPrivateBetaExampleNoEmbeddedSecrets(t *testing.T) {
	cfg := loadPrivateBetaExample(t)
	for _, pair := range []struct {
		name, val string
	}{
		{"api_key", cfg.APIKey},
		{"gitea_token", cfg.GiteaToken},
		{"webhook_secret", cfg.WebhookSecret},
		{"ai_api_key", cfg.AIAPIKey},
	} {
		if strings.TrimSpace(pair.val) != "" {
			t.Errorf("%s must be empty in example config", pair.name)
		}
	}
}

func TestPrivateBetaExampleSafetyDefaults(t *testing.T) {
	cfg := loadPrivateBetaExample(t)
	if cfg.AutoCreateIssues {
		t.Error("auto_create_issues must be false")
	}
	if cfg.RemediationPREnabled {
		t.Error("remediation_pr_enabled must be false")
	}
	if cfg.RunnerDelegationEnabled {
		t.Error("runner_delegation_enabled must be false")
	}
	if cfg.NotificationsEnabled {
		t.Error("notifications_enabled must be false")
	}
	if cfg.LLMSanityGateEnabled {
		t.Error("llm_sanity_gate_enabled must be false")
	}
	if cfg.EnableLLMAuditors {
		t.Error("enable_llm_auditors must be false")
	}
	if !cfg.EvidenceClosureEnabled {
		t.Error("evidence_closure_enabled must be true")
	}
	if !cfg.BacklogControlEnabled {
		t.Error("dogfood_backlog_control_enabled must be true")
	}
	if cfg.Reporting.MaxIssuesPerScan != 0 {
		t.Errorf("reporting.max_issues_per_scan want 0 got %d", cfg.Reporting.MaxIssuesPerScan)
	}
}
