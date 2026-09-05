package ui

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
