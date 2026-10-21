package openclaw

import (
	"context"
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
