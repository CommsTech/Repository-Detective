package openclaw

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// PreflightDecision explains whether AI spend should be skipped for a scan.
type PreflightDecision struct {
	Skip   bool
	Reason string
}

var (
	preflightHTTPOnce   sync.Once
	preflightHTTPClient *http.Client
)

func sharedPreflightHTTPClient(timeout time.Duration) *http.Client {
	preflightHTTPOnce.Do(func() {
		preflightHTTPClient = &http.Client{}
	})
	// Clone timeout per call without allocating a new transport each probe.
	client := *preflightHTTPClient
	client.Timeout = timeout
	return &client
}

// EvaluatePreflight applies ponytail-style fail-closed gates: if the gateway or
// recent learning signals say AI is sick, do not burn tokens.
func EvaluatePreflight(timeoutEvents, advisoryFailEvents int, probeErr error, cfg Config) PreflightDecision {
	cfg = cfg.Normalized()
	lookback := cfg.PreflightLookbackHours
	if lookback <= 0 {
		lookback = 6
	}
	timeoutLimit := cfg.PreflightTimeoutEvents
	if timeoutLimit <= 0 {
		timeoutLimit = 10
	}
	advisoryLimit := cfg.PreflightAdvisoryFailEvents
	if advisoryLimit <= 0 {
		advisoryLimit = 5
	}
	if timeoutEvents >= timeoutLimit {
		return PreflightDecision{
			Skip: true,
			Reason: fmt.Sprintf("preflight: %d llm_auditor_timeout events in ~%dh (limit %d) — skipping AI spend",
				timeoutEvents, lookback, timeoutLimit),
		}
	}
	if advisoryFailEvents >= advisoryLimit {
		return PreflightDecision{
			Skip: true,
			Reason: fmt.Sprintf("preflight: %d ai_advisory_failed events in ~%dh (limit %d) — skipping AI spend",
				advisoryFailEvents, lookback, advisoryLimit),
		}
	}
	if probeErr != nil {
		return PreflightDecision{
			Skip:   true,
			Reason: fmt.Sprintf("preflight: AI endpoint health probe failed — skipping AI spend: %v", probeErr),
		}
	}
	return PreflightDecision{}
}

// ProbeEndpoint performs a cheap readiness check against the AI endpoint.
// It tries /health then the endpoint root; network/5xx failures count as unhealthy.
func ProbeEndpoint(ctx context.Context, endpoint string, timeout time.Duration) error {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		return fmt.Errorf("empty endpoint")
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := sharedPreflightHTTPClient(timeout)
	candidates := []string{
		endpoint + "/health",
		endpoint,
	}
	var lastErr error
	for _, url := range candidates {
		req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, url, nil)
		if err != nil {
			lastErr = err
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("%s returned HTTP %d", url, resp.StatusCode)
			continue
		}
		// 2xx/3xx/4xx (except we already skipped 5xx) means the process answered.
		return nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no healthy response from %s", endpoint)
	}
	return lastErr
}
