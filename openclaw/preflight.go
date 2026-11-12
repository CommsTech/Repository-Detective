package openclaw

import (
	"context"
	"crypto/tls"
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
//
// insecureSkipTLSVerify must match the explicit operator setting used by the
// real AI transport. It never mutates the shared strict client: opt-in insecure
// probes receive a dedicated cloned transport, preventing TLS policy leakage to
// later strict probes.
func ProbeEndpoint(ctx context.Context, endpoint string, timeout time.Duration, insecureSkipTLSVerify ...bool) error {
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
	if len(insecureSkipTLSVerify) > 0 && insecureSkipTLSVerify[0] {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec G402 -- explicit operator opt-in, parity with AI transport
		client = &http.Client{Transport: transport, Timeout: timeout}
	}
	candidates := probeCandidateURLs(endpoint)
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

// probeCandidateURLs prefers the gateway origin /health when the configured
// OpenAI-compatible base ends in /v1 (OpenClaw, OpenWebUI, etc.). Probing
// "{base}/v1/health" false-negatives healthy gateways and disables AI spend.
func probeCandidateURLs(endpoint string) []string {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	origin := endpoint
	lower := strings.ToLower(origin)
	if strings.HasSuffix(lower, "/v1") {
		origin = strings.TrimRight(origin[:len(origin)-3], "/")
	}
	seen := make(map[string]struct{}, 4)
	out := make([]string, 0, 4)
	add := func(u string) {
		u = strings.TrimRight(strings.TrimSpace(u), "/")
		if u == "" {
			return
		}
		if _, ok := seen[u]; ok {
			return
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	if origin != "" {
		add(origin + "/health")
		add(origin)
	}
	if endpoint != origin {
		add(endpoint + "/health")
		add(endpoint)
	} else {
		add(endpoint + "/health")
		add(endpoint)
	}
	return out
}
