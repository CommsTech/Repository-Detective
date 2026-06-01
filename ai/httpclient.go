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
	if insecureSkipTLSVerify {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}
		transport.TLSClientConfig.InsecureSkipVerify = true
	}
	return &http.Client{
		Timeout:   120 * time.Second,
		Transport: transport,
	}
}
