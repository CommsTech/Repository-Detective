package ui

import "strings"

// AuthConfig holds local session auth settings for the operator UI.
type AuthConfig struct {
	Mode                       string // api_key_only | local
	SessionSecret              string
	SessionCookieName          string
	SessionTTLHours            int
	CSRFEnabled                bool
	LocalAdminBootstrapEnabled bool
	PublicURL                  string
	RejectQueryStringAPIKey    bool
	WarnQueryStringAPIKey      bool
}

// IsLocal returns true when browser sessions are required for the UI.
func (a AuthConfig) IsLocal() bool {
	return a.Mode == "local"
}

// CookieSecure reports whether UI cookies should set the Secure attribute.
// Homelab installs often expose the UI over plain HTTP (public_url=http://...);
// forcing Secure there makes unlock/login appear to fail because browsers drop the cookie.
func (a AuthConfig) CookieSecure() bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(a.PublicURL)), "https://")
}
