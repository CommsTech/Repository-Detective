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
	// BootstrapToken, when set, must be supplied as ?token= or form field
	// bootstrap_token to claim the first owner. Use this when the UI is
	// reachable beyond loopback (LAN / reverse proxy).
	BootstrapToken string
	// BootstrapAllowRemote permits first-owner creation from non-loopback
	// clients when a BootstrapToken is also configured.
	BootstrapAllowRemote    bool
	PublicURL               string
	RejectQueryStringAPIKey bool
	WarnQueryStringAPIKey   bool
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
