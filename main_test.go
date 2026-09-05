package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func TestNeedsAIProvider(t *testing.T) {
	cfg := &Config{
		AnalysisDepth:     1,
		EnableLLMAuditors: false,
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
}

func TestBuildReadinessAIAnalysisDisabledWithoutClient(t *testing.T) {
	prevClient := aiClient
	prevConfig := config
	aiClient = nil
	config = &Config{}
	t.Cleanup(func() {
		aiClient = prevClient
		config = prevConfig
	})

	r := buildReadiness("healthy")
	if r.AIAnalysis != "Disabled" {
		t.Fatalf("AIAnalysis=%q want Disabled", r.AIAnalysis)
	}
	if r.AIProvider != "disabled" {
		t.Fatalf("AIProvider=%q want disabled", r.AIProvider)
	}
}

func TestGiteaStatusConfigDefaults(t *testing.T) {
	v := viper.New()
	v.SetDefault("enable_gitea_status", false)
	v.SetDefault("gitea_status_context", "repository-detective/security-scan")
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
	if cfg.GiteaStatusContext != "repository-detective/security-scan" {
		t.Fatalf("unexpected context default %q", cfg.GiteaStatusContext)
	}
	if cfg.GiteaStatusFailOn != "high" || cfg.GiteaStatusWarnOn != "medium" {
		t.Fatalf("unexpected severity defaults fail=%q warn=%q", cfg.GiteaStatusFailOn, cfg.GiteaStatusWarnOn)
	}
	if !cfg.GiteaStatusIncludeScannerFailures {
		t.Fatal("expected scanner failure inclusion default true")
	}
}

func TestRequireAPIKeyAuthAcceptsRepositoryDetectiveHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	saved := config
	defer func() { config = saved }()
	config = &Config{APIKey: "test-secret-key"}

	r := gin.New()
	r.GET("/protected", requireAPIKeyAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	cases := []struct {
		name       string
		setHeaders func(*http.Request)
		wantStatus int
	}{
		{
			name: "X-Repository-Detective-API-Key",
			setHeaders: func(req *http.Request) {
				req.Header.Set("X-Repository-Detective-API-Key", "test-secret-key")
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing key",
			setHeaders: func(req *http.Request) {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "wrong key",
			setHeaders: func(req *http.Request) {
				req.Header.Set("X-Repository-Detective-API-Key", "wrong")
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "unknown legacy API key header rejected",
			setHeaders: func(req *http.Request) {
				req.Header.Set("X-Bugbot-API-Key", "test-secret-key")
			},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			tc.setHeaders(req)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}
