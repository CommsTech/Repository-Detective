package privacy

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// Operating modes (RD-007).
const (
	ModeStandard          = "standard" // legacy alias for hybrid-compatible default
	ModeLocalOnly         = "local_only"
	ModeHybrid            = "hybrid"
	ModeExternalAIEnabled = "external_ai_enabled"
)

// EndpointClass describes whether a destination is local to the operator network.
const (
	ClassLocal    = "LOCAL"
	ClassExternal = "EXTERNAL"
	ClassUnknown  = "UNKNOWN"
)

// NormalizeMode maps config strings to canonical modes.
func NormalizeMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", ModeStandard, "default":
		return ModeHybrid
	case ModeLocalOnly, "local-only", "localonly", "airgap", "air_gap":
		return ModeLocalOnly
	case ModeHybrid:
		return ModeHybrid
	case ModeExternalAIEnabled, "external", "external_ai", "cloud_ai":
		return ModeExternalAIEnabled
	default:
		return strings.ToLower(strings.TrimSpace(mode))
	}
}

// ValidMode reports whether mode is recognized.
func ValidMode(mode string) bool {
	switch NormalizeMode(mode) {
	case ModeLocalOnly, ModeHybrid, ModeExternalAIEnabled:
		return true
	default:
		return false
	}
}

// ClassifyHost classifies a hostname or literal IP without DNS (literal IPs + well-known local names).
// Hostnames that are not loopback literals are UNKNOWN until ResolveAndClassify is used.
func ClassifyHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return ClassUnknown
	}
	// Strip brackets for IPv6 literals.
	host = strings.Trim(host, "[]")
	if host == "localhost" || host == "localhost." {
		return ClassLocal
	}
	if ip := net.ParseIP(host); ip != nil {
		return classifyIP(ip)
	}
	return ClassUnknown
}

// ResolveAndClassify resolves host and classifies all returned addresses.
// Returns EXTERNAL if any address is public; LOCAL if all are local; UNKNOWN on resolution failure.
func ResolveAndClassify(host string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return ClassUnknown, fmt.Errorf("empty host")
	}
	host = strings.Trim(host, "[]")
	if c := ClassifyHost(host); c == ClassLocal || c == ClassExternal {
		return c, nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return ClassUnknown, err
	}
	if len(ips) == 0 {
		return ClassUnknown, fmt.Errorf("no addresses for %s", host)
	}
	allLocal := true
	for _, ip := range ips {
		switch classifyIP(ip) {
		case ClassLocal:
			continue
		case ClassExternal:
			allLocal = false
		default:
			allLocal = false
		}
	}
	if allLocal {
		return ClassLocal, nil
	}
	return ClassExternal, nil
}

func classifyIP(ip net.IP) string {
	if ip == nil {
		return ClassUnknown
	}
	if ip.IsLoopback() {
		return ClassLocal
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4.IsPrivate() || ip4.IsLinkLocalUnicast() {
			return ClassLocal
		}
		// RFC1918 covered by IsPrivate; CGNAT 100.64/10 treat as local for homelab
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return ClassLocal
		}
		return ClassExternal
	}
	// IPv6
	if ip.IsPrivate() || ip.IsLinkLocalUnicast() { // ULA is IsPrivate in Go 1.20+
		return ClassLocal
	}
	return ClassExternal
}

// ClassifyURL returns locality of the host portion of a URL.
func ClassifyURL(raw string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ClassUnknown, "", fmt.Errorf("empty url")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ClassUnknown, "", err
	}
	host := u.Hostname()
	if host == "" {
		return ClassUnknown, "", fmt.Errorf("url missing host")
	}
	if c := ClassifyHost(host); c != ClassUnknown {
		return c, host, nil
	}
	c, err := ResolveAndClassify(host)
	return c, host, err
}

// AIEgressDecision is the result of checking AI config under a privacy mode.
type AIEgressDecision struct {
	Allowed       bool
	Mode          string
	Provider      string
	Endpoint      string
	EndpointClass string
	Reason        string
}

// EvaluateAIEgress decides whether AI calls are allowed under mode.
// LOCAL_ONLY requires a LOCAL classified endpoint (or empty AI when auditors off).
func EvaluateAIEgress(mode, provider, baseURL string, llmAuditorsEnabled bool) AIEgressDecision {
	mode = NormalizeMode(mode)
	provider = strings.ToLower(strings.TrimSpace(provider))
	d := AIEgressDecision{Mode: mode, Provider: provider, Endpoint: strings.TrimSpace(baseURL)}

	if !llmAuditorsEnabled && strings.TrimSpace(baseURL) == "" && provider == "" {
		d.Allowed = true
		d.EndpointClass = ClassLocal
		d.Reason = "AI Analysis: Disabled — no AI egress"
		return d
	}

	switch mode {
	case ModeExternalAIEnabled:
		d.Allowed = true
		if baseURL == "" {
			d.EndpointClass = ClassExternal
			d.Reason = "external AI mode — repository context may leave this network"
			return d
		}
		class, _, err := ClassifyURL(baseURL)
		if err != nil {
			d.EndpointClass = ClassUnknown
			d.Reason = "external AI mode; endpoint classification inconclusive: " + err.Error()
			return d
		}
		d.EndpointClass = class
		d.Reason = "external AI mode enabled"
		return d

	case ModeHybrid:
		d.Allowed = true
		if baseURL == "" {
			// Cloud provider names without base URL imply external defaults.
			switch provider {
			case "openai", "anthropic", "openrouter":
				d.EndpointClass = ClassExternal
				d.Reason = "hybrid — cloud provider may send repository context externally"
			default:
				d.EndpointClass = ClassUnknown
				d.Reason = "hybrid — configure base URL for locality disclosure"
			}
			return d
		}
		class, _, err := ClassifyURL(baseURL)
		if err != nil {
			d.EndpointClass = ClassUnknown
			d.Reason = "hybrid; could not classify endpoint: " + err.Error()
			return d
		}
		d.EndpointClass = class
		if class == ClassExternal {
			d.Reason = "hybrid — external endpoint; repository context may leave this network"
		} else {
			d.Reason = "hybrid — endpoint classified " + class
		}
		return d

	case ModeLocalOnly:
		if !llmAuditorsEnabled && baseURL == "" {
			d.Allowed = true
			d.EndpointClass = ClassLocal
			d.Reason = "LOCAL_ONLY — AI disabled"
			return d
		}
		switch provider {
		case "openai", "anthropic", "openrouter":
			d.Allowed = false
			d.EndpointClass = ClassExternal
			d.Reason = "LOCAL_ONLY rejects cloud AI providers"
			return d
		}
		if baseURL == "" {
			d.Allowed = false
			d.EndpointClass = ClassUnknown
			d.Reason = "LOCAL_ONLY requires an explicit local AI base URL"
			return d
		}
		class, _, err := ClassifyURL(baseURL)
		if err != nil {
			d.Allowed = false
			d.EndpointClass = ClassUnknown
			d.Reason = "LOCAL_ONLY cannot classify AI endpoint: " + err.Error()
			return d
		}
		d.EndpointClass = class
		if class != ClassLocal {
			d.Allowed = false
			d.Reason = "LOCAL_ONLY blocks non-local AI endpoints (got " + class + ")"
			return d
		}
		d.Allowed = true
		d.Reason = "LOCAL_ONLY — AI endpoint classified LOCAL"
		return d
	}

	d.Allowed = false
	d.Reason = "unknown privacy mode"
	return d
}

// URLEgressDecision is the result of checking a webhook/outbound URL under privacy mode.
type URLEgressDecision struct {
	Allowed       bool
	Mode          string
	URL           string
	EndpointClass string
	Reason        string
}

// EvaluateURLEgress decides whether an outbound content URL is allowed.
// Forge URLs are not gated here (product sink); use for notifications / advisory endpoints.
func EvaluateURLEgress(mode, rawURL string) URLEgressDecision {
	mode = NormalizeMode(mode)
	d := URLEgressDecision{Mode: mode, URL: strings.TrimSpace(rawURL)}
	if d.URL == "" {
		d.Allowed = true
		d.EndpointClass = ClassLocal
		d.Reason = "no URL configured"
		return d
	}
	class, _, err := ClassifyURL(d.URL)
	if err != nil {
		d.EndpointClass = ClassUnknown
		if mode == ModeLocalOnly {
			d.Allowed = false
			d.Reason = "LOCAL_ONLY cannot classify destination: " + err.Error()
			return d
		}
		d.Allowed = true
		d.Reason = "classification inconclusive; allowed under " + mode
		return d
	}
	d.EndpointClass = class
	switch mode {
	case ModeLocalOnly:
		if class != ClassLocal {
			d.Allowed = false
			d.Reason = "LOCAL_ONLY blocks EXTERNAL/UNKNOWN content destinations"
			return d
		}
		d.Allowed = true
		d.Reason = "LOCAL_ONLY — destination classified LOCAL"
		return d
	default:
		d.Allowed = true
		if class == ClassExternal {
			d.Reason = mode + " — external destination; repository context may leave this network"
		} else {
			d.Reason = mode + " — destination classified " + class
		}
		return d
	}
}

// LocalOnlyGuarantee is the operator-facing contract text.
const LocalOnlyGuarantee = `LOCAL_ONLY guarantees Repository Detective will not intentionally send
repository source, diffs, finding snippets, or AI prompts to destinations
classified as EXTERNAL.

Still permitted (operator-controlled):
- Local scanners and subprocesses
- Local Ollama / OpenAI-compatible endpoints classified LOCAL
- Local runners on the operator network
- The configured forge (Gitea/Forgejo) — if that forge is hosted outside
  this network, issue bodies and PR summaries still egress there by design.
  Point Gitea_URL at a local forge for a fully air-gapped loop.

Blocked:
- Cloud AI providers and EXTERNAL AI base URLs
- OpenClaw / advisory AI sending snippets to EXTERNAL endpoints
- Notification webhooks to EXTERNAL destinations (enforced at startup)
`
