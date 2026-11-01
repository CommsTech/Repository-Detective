package ai

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
)

// NewHTTPClient returns an HTTP client for AI provider transports.
// InsecureSkipTLSVerify is for homelab endpoints with private CAs only, and is
// ignored when caCertPEM is provided: certificate verification then stays
// fully enabled against the system roots plus the supplied CA(s).
//
// Timeout is left unset (0): callers must bound requests with context deadlines.
// A fixed client timeout (previously 120s) raced with OpenClaw agent turns that
// often need 2–5 minutes and aborted mid-flight with "Client.Timeout exceeded
// while awaiting headers" even when the review context still had budget.
func NewHTTPClient(insecureSkipTLSVerify bool, caCertPEM string) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if caCertPEM != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(caCertPEM)) {
			// Keep verification enabled and fall back to system roots; an
			// invalid CA bundle must never disable TLS validation.
			pool = nil
		}
		tlsCfg.RootCAs = pool
	} else if insecureSkipTLSVerify {
		tlsCfg.InsecureSkipVerify = true
	}
	transport.TLSClientConfig = tlsCfg
	return &http.Client{
		Timeout:   0,
		Transport: transport,
	}
}
