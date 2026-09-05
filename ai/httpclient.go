package ai

import (
	"crypto/tls"
	"net/http"
	"time"
)

// NewHTTPClient returns an HTTP client for AI provider transports.
// InsecureSkipTLSVerify is for homelab endpoints with private CAs only.
func NewHTTPClient(insecureSkipTLSVerify bool) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if insecureSkipTLSVerify {
		tlsCfg.InsecureSkipVerify = true
	}
	transport.TLSClientConfig = tlsCfg
	return &http.Client{
		Timeout:   120 * time.Second,
		Transport: transport,
	}
}
