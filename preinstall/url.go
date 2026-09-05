package preinstall

import (
	"fmt"
	"net"
	"net/url"
	"path"
	"strings"
)

const maxRepoURLLength = 2048

var blockedHostSuffixes = []string{
	".local",
	".internal",
	".localhost",
}

// ParsedRepoURL is a validated public HTTPS repository URL.
type ParsedRepoURL struct {
	Original      string
	Normalized    string
	CloneURL      string
	Host          string
	Owner         string
	Name          string
	DefaultBranch string
}

var blockedHosts = map[string]struct{}{
	"localhost": {},
	"127.0.0.1": {},
	"::1":       {},
	"0.0.0.0":   {},
}

// lookupHostIPs resolves hostnames for SSRF checks. Tests may override.
var lookupHostIPs = net.LookupIP

// ValidateRepoURL validates and normalizes a third-party repository HTTPS URL.
func ValidateRepoURL(raw string, allowPrivateNetworks bool) (ParsedRepoURL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ParsedRepoURL{}, fmt.Errorf("repo URL is required")
	}
	if len(raw) > maxRepoURLLength {
		return ParsedRepoURL{}, fmt.Errorf("repository URL is too long")
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "file://") || strings.HasPrefix(lower, "git@") || strings.HasPrefix(lower, "ssh://") {
		return ParsedRepoURL{}, fmt.Errorf("unsupported repository URL scheme")
	}
	if strings.Contains(raw, `\`) || strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "./") || strings.HasPrefix(raw, "../") {
		return ParsedRepoURL{}, fmt.Errorf("local filesystem paths are not allowed")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return ParsedRepoURL{}, fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return ParsedRepoURL{}, fmt.Errorf("only HTTPS repository URLs are supported")
	}
	if parsed.User != nil {
		return ParsedRepoURL{}, fmt.Errorf("URLs with embedded credentials are not allowed")
	}
	if strings.Contains(parsed.Host, "@") {
		return ParsedRepoURL{}, fmt.Errorf("URLs with embedded credentials are not allowed")
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return ParsedRepoURL{}, fmt.Errorf("repository host is required")
	}
	if _, blocked := blockedHosts[host]; blocked {
		return ParsedRepoURL{}, fmt.Errorf("localhost and loopback hosts are not allowed")
	}
	if hasBlockedHostSuffix(host) {
		return ParsedRepoURL{}, fmt.Errorf("repository host is not allowed")
	}

	if err := RevalidateHost(host, allowPrivateNetworks); err != nil {
		return ParsedRepoURL{}, err
	}

	segments := splitRepoPath(parsed.Path)
	if len(segments) < 2 {
		return ParsedRepoURL{}, fmt.Errorf("repository URL must include owner and name")
	}
	owner := strings.Join(segments[:len(segments)-1], "/")
	name := segments[len(segments)-1]
	name = strings.TrimSuffix(name, ".git")
	name = strings.TrimSuffix(name, ".GIT")
	if owner == "" || name == "" {
		return ParsedRepoURL{}, fmt.Errorf("invalid repository path")
	}
	if !validRepoSegment(owner) || !validRepoSegment(name) {
		return ParsedRepoURL{}, fmt.Errorf("invalid repository owner or name")
	}

	normalized := fmt.Sprintf("https://%s/%s/%s", host, owner, name)
	cloneURL := normalized
	if !strings.HasSuffix(strings.ToLower(cloneURL), ".git") {
		cloneURL += ".git"
	}

	return ParsedRepoURL{
		Original:   raw,
		Normalized: normalized,
		CloneURL:   cloneURL,
		Host:       host,
		Owner:      owner,
		Name:       name,
	}, nil
}

func splitRepoPath(p string) []string {
	p = strings.TrimSpace(p)
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return nil
	}
	parts := strings.Split(path.Clean("/"+p), "/")
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}
	if len(parts) == 0 {
		return nil
	}
	last := parts[len(parts)-1]
	if strings.EqualFold(last, ".git") {
		parts = parts[:len(parts)-1]
	}
	if len(parts) == 0 {
		return nil
	}
	last = parts[len(parts)-1]
	if strings.EqualFold(last, "tree") || strings.EqualFold(last, "blob") || strings.EqualFold(last, "-") {
		parts = parts[:len(parts)-1]
	}
	if len(parts) == 0 {
		return nil
	}
	return parts
}

// SetLookupHostIPsForTests overrides DNS resolution for SSRF checks (tests only).
func SetLookupHostIPsForTests(fn func(string) ([]net.IP, error)) {
	if fn == nil {
		lookupHostIPs = net.LookupIP
		return
	}
	lookupHostIPs = fn
}

// LookupHostIPsForTests returns the current lookup function.
func LookupHostIPsForTests() func(string) ([]net.IP, error) {
	return lookupHostIPs
}

// RevalidateHost re-checks a host before network I/O (mitigates DNS rebinding between validation and clone).
func RevalidateHost(host string, allowPrivateNetworks bool) error {
	return checkHostNetwork(strings.ToLower(strings.TrimSpace(host)), allowPrivateNetworks)
}

func hasBlockedHostSuffix(host string) bool {
	for _, suffix := range blockedHostSuffixes {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}

func validRepoSegment(segment string) bool {
	if segment == "" || segment == "." || segment == ".." {
		return false
	}
	if strings.Contains(segment, "..") {
		return false
	}
	for _, part := range strings.Split(segment, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func checkHostNetwork(host string, allowPrivateNetworks bool) error {
	if allowPrivateNetworks {
		return nil
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("repository host resolves to a private or local network address")
		}
		return nil
	}
	ips, err := lookupHostIPs(host)
	if err != nil {
		return fmt.Errorf("unable to resolve repository host: %w", err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("repository host did not resolve")
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return fmt.Errorf("repository host resolves to a private or local network address")
		}
	}
	return nil
}

func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		// CGNAT and metadata-ish ranges
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
	}
	// Unique local IPv6 (fc00::/7)
	if ip16 := ip.To16(); ip16 != nil && ip.To4() == nil {
		if ip16[0] == 0xfc || ip16[0] == 0xfd {
			return true
		}
	}
	return false
}

// NormalizeAuditDepth returns a supported audit depth.
func NormalizeAuditDepth(depth string) string {
	switch strings.ToLower(strings.TrimSpace(depth)) {
	case "quick":
		return "quick"
	case "deep":
		return "deep"
	default:
		return "standard"
	}
}
