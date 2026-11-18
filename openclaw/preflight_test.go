package openclaw_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/openclaw"
)

func TestProbeEndpointStripsV1SuffixForGatewayHealth(t *testing.T) {
	var hitHealth, hitV1Health bool
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		hitHealth = true
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		hitV1Health = true
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/v1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	if err := openclaw.ProbeEndpoint(context.Background(), server.URL+"/v1", time.Second); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if !hitHealth {
		t.Fatal("expected gateway /health probe")
	}
	if hitV1Health {
		t.Fatal("should prefer origin /health before /v1/health")
	}
}

func TestEvaluatePreflightSkipsOnTimeoutFlood(t *testing.T) {
	cfg := openclaw.DefaultConfig()
	cfg.PreflightTimeoutEvents = 10
	d := openclaw.EvaluatePreflight(10, 0, nil, cfg)
	if !d.Skip {
		t.Fatalf("expected skip on timeout flood, got %+v", d)
	}
}

func TestEvaluatePreflightSkipsOnAdvisoryFails(t *testing.T) {
	cfg := openclaw.DefaultConfig()
	cfg.PreflightAdvisoryFailEvents = 5
	d := openclaw.EvaluatePreflight(0, 5, nil, cfg)
	if !d.Skip {
		t.Fatalf("expected skip on advisory failures, got %+v", d)
	}
}

func TestEvaluatePreflightSkipsOnProbeError(t *testing.T) {
	cfg := openclaw.DefaultConfig()
	d := openclaw.EvaluatePreflight(0, 0, errors.New("connection refused"), cfg)
	if !d.Skip {
		t.Fatalf("expected skip on probe error, got %+v", d)
	}
}

func TestEvaluatePreflightAllowsHealthy(t *testing.T) {
	cfg := openclaw.DefaultConfig()
	d := openclaw.EvaluatePreflight(2, 1, nil, cfg)
	if d.Skip {
		t.Fatalf("expected allow, got %+v", d)
	}
}
