package openclaw_test

import (
	"errors"
	"testing"

	"git.commsnet.org/commstech/repository-detective/openclaw"
)

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
