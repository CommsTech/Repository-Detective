package openclaw

import (
	"fmt"
	"regexp"
	"strings"

	"git.commsnet.org/commstech/repository-detective/redact"
)

const maxFieldLen = 2000

var (
	privateKeyBlock = regexp.MustCompile(`(?s)-----BEGIN[A-Z ]*PRIVATE KEY-----.*?-----END[A-Z ]*PRIVATE KEY-----`)
	dotEnvLine      = regexp.MustCompile(`(?im)^[A-Z0-9_]+=.+$`)
	urlToken        = regexp.MustCompile(`(?i)(https?://)[^/\s]+@`)
	piiEmail        = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	awsAccessKey    = regexp.MustCompile(`AKIA[0-9A-Z]{16}`)
)

// RedactText sanitizes a single text field for outbound AI review.
func RedactText(value string, redactPII bool) (string, int) {
	count := 0
	before := value
	value = redact.SecretEvidence(value)
	if value != before {
		count++
	}
	if awsAccessKey.MatchString(value) {
		value = awsAccessKey.ReplaceAllString(value, "[REDACTED_AWS_KEY]")
		count++
	}
	value = privateKeyBlock.ReplaceAllString(value, "[REDACTED_PRIVATE_KEY]")
	if strings.Contains(value, "[REDACTED_PRIVATE_KEY]") {
		count++
	}
	value = dotEnvLine.ReplaceAllString(value, "[REDACTED_ENV_LINE]")
	value = urlToken.ReplaceAllString(value, "${1}***@")
	if redactPII {
		if piiEmail.ReplaceAllString(value, "[REDACTED_EMAIL]") != value {
			count++
			value = piiEmail.ReplaceAllString(value, "[REDACTED_EMAIL]")
		}
	}
	value = truncate(value, maxFieldLen)
	return strings.TrimSpace(value), count
}

// RedactPacket sanitizes all finding fields. Returns error if redaction incomplete.
func RedactPacket(pkt *ReviewPacket, cfg Config) (int, error) {
	if pkt == nil {
		return 0, fmt.Errorf("packet is nil")
	}
	cfg = cfg.Normalized()
	if !cfg.RedactSecrets {
		return 0, fmt.Errorf("secret redaction required for AI recommendations")
	}
	total := 0
	for i := range pkt.Findings {
		f := &pkt.Findings[i]
		var c int
		f.Title, c = RedactText(f.Title, cfg.RedactPII)
		total += c
		f.DescriptionRedacted, c = RedactText(f.DescriptionRedacted, cfg.RedactPII)
		total += c
		f.EvidenceRedacted, c = RedactText(f.EvidenceRedacted, cfg.RedactPII)
		total += c
		f.Path, c = RedactText(f.Path, cfg.RedactPII)
		total += c
		f.RuleID, c = RedactText(f.RuleID, cfg.RedactPII)
		total += c
		if looksLikeSecretMaterial(f.EvidenceRedacted) || looksLikeSecretMaterial(f.DescriptionRedacted) {
			return total, fmt.Errorf("redaction incomplete for fingerprint %s", f.Fingerprint)
		}
	}
	return total, nil
}

func looksLikeSecretMaterial(s string) bool {
	if awsAccessKey.MatchString(s) {
		return true
	}
	lower := strings.ToLower(s)
	if strings.Contains(lower, "akia") && len(s) > 16 {
		return true
	}
	if strings.Contains(lower, "-----begin") && strings.Contains(lower, "private key") {
		return true
	}
	if strings.Contains(lower, "bearer ") && !strings.Contains(s, "[REDACTED]") {
		return true
	}
	return false
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…[truncated]"
}
