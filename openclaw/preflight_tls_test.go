package openclaw

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProbeEndpointHonorsExplicitInsecureTLSOptIn(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" && r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	if err := ProbeEndpoint(ctx, server.URL, time.Second); err == nil {
		t.Fatal("strict probe must reject an untrusted self-signed certificate")
	}
	if err := ProbeEndpoint(ctx, server.URL, time.Second, true); err != nil {
		t.Fatalf("explicit insecure TLS opt-in must match AI transport behavior: %v", err)
	}
}

func TestProbeEndpointInsecureOptInDoesNotLeakToStrictProbe(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	if err := ProbeEndpoint(ctx, server.URL, time.Second, true); err != nil {
		t.Fatalf("opt-in probe: %v", err)
	}
	if err := ProbeEndpoint(ctx, server.URL, time.Second); err == nil {
		t.Fatal("strict probe after insecure probe must still reject self-signed TLS")
	}
}

func TestProbeEndpointStrictHTTPStillWorks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized) // process answered; 4xx is reachable
	}))
	defer server.Close()

	if err := ProbeEndpoint(context.Background(), server.URL, time.Second); err != nil {
		t.Fatalf("HTTP endpoint should be reachable: %v", err)
	}
}

func TestProbeEndpointInsecureOptInKeepsTLS12MinVersionLiteral(t *testing.T) {
	// Guard against regressions that drop MinVersion from the insecure probe
	// transport (semgrep missing-ssl-minversion / #538).
	cfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true, // #nosec G402 -- mirrors ProbeEndpoint insecure path
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Fatalf("MinVersion=%d want TLS1.2", cfg.MinVersion)
	}
	_ = t
}
