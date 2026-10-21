package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func TestApplySecurityDefaultsFailSafe(t *testing.T) {
	v := viper.New()
	applySecurityDefaults(v)

	if !v.GetBool("reject_query_string_api_key") {
		t.Fatal("query-string API keys must be rejected by default")
	}
	if !v.GetBool("webhook_replay_protection_enabled") {
		t.Fatal("webhook replay protection must be enabled by default")
	}
	if got := v.GetInt("webhook_replay_retention_hours"); got != 168 {
		t.Fatalf("replay retention=%d, want 168", got)
	}
	if got := v.GetInt("webhook_replay_max_rows"); got != 100000 {
		t.Fatalf("replay max rows=%d, want 100000", got)
	}
	if got := v.GetStringSlice("trusted_proxies"); len(got) != 0 {
		t.Fatalf("trusted proxies must default empty, got %#v", got)
	}
}

func TestNormalizeTrustedProxies(t *testing.T) {
	got, err := normalizeTrustedProxies([]string{" 127.0.0.1 ", "10.0.0.0/8", "2001:db8::1", ""})
	if err != nil {
		t.Fatalf("valid proxies: %v", err)
	}
	want := []string{"127.0.0.1", "10.0.0.0/8", "2001:db8::1"}
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("proxy[%d]=%q, want %q", i, got[i], want[i])
		}
	}
	for _, invalid := range []string{"proxy.internal", "10.0.0.0/999", "not-an-ip"} {
		if _, err := normalizeTrustedProxies([]string{invalid}); err == nil {
			t.Fatalf("expected invalid trusted proxy %q to fail", invalid)
		}
	}
}

func TestApplyTrustedProxiesEmptyIgnoresForwardedHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if err := applyTrustedProxies(router, nil); err != nil {
		t.Fatalf("applyTrustedProxies: %v", err)
	}
	router.GET("/ip", func(c *gin.Context) { c.String(http.StatusOK, c.ClientIP()) })

	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = "10.0.0.7:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.99")
	req.Header.Set("X-Real-IP", "203.0.113.100")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if got := w.Body.String(); got != "10.0.0.7" {
		t.Fatalf("ClientIP=%q, want direct peer 10.0.0.7", got)
	}
}

func TestAPIKeyAuthRejectsQueryAndAcceptsBothHeaders(t *testing.T) {
	previousConfig, previousLogger := config, logger
	config = &Config{APIKey: "fixture-value", RejectQueryStringAPIKey: true}
	logger = logrus.New()
	t.Cleanup(func() { config, logger = previousConfig, previousLogger })

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ok", requireAPIKeyAuth(), func(c *gin.Context) { c.Status(http.StatusOK) })

	tests := []struct {
		name   string
		url    string
		header string
		value  string
		want   int
	}{
		{name: "no key", url: "/ok", want: http.StatusUnauthorized},
		{name: "query rejected", url: "/ok?api_key=fixture-value", want: http.StatusUnauthorized},
		{name: "product header", url: "/ok", header: "X-Repository-Detective-API-Key", value: "fixture-value", want: http.StatusOK},
		{name: "legacy header", url: "/ok", header: "X-API-Key", value: "fixture-value", want: http.StatusOK},
		{name: "bad header", url: "/ok", header: "X-API-Key", value: "wrong", want: http.StatusUnauthorized},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			if tc.header != "" {
				req.Header.Set(tc.header, tc.value)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("status=%d body=%s, want %d", w.Code, w.Body.String(), tc.want)
			}
		})
	}
}
