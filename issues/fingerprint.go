package issues

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/redact"
)

const lineBlockSize = 10

// SanitizeSecretEvidence redacts likely secret material from snippets.
func SanitizeSecretEvidence(value string) string {
	return redact.SecretEvidence(value)
}

// FingerprintInput carries fields used to compute a stable finding fingerprint.
type FingerprintInput struct {
	Repository   string
	Category     string
	Source       string
	RuleID       string
	File         string
	Line         int
	PackageName  string
	EvidenceHash string
}

// ComputeFingerprint returns a stable Repository Detective fingerprint for cross-scan tracking.
func ComputeFingerprint(in FingerprintInput) string {
	lineBlock := (in.Line / lineBlockSize) * lineBlockSize
	if in.Line > 0 && lineBlock == 0 {
		lineBlock = 1
	}

	parts := []string{
		strings.ToLower(strings.TrimSpace(in.Repository)),
		NormalizeCategory(in.Category, in.Source),
		strings.ToLower(strings.TrimSpace(in.Source)),
		strings.ToLower(strings.TrimSpace(in.RuleID)),
		normalizePath(in.File),
		fmt.Sprintf("block:%d", lineBlock),
		strings.ToLower(strings.TrimSpace(in.PackageName)),
		strings.TrimSpace(in.EvidenceHash),
	}

	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return "rd-" + hex.EncodeToString(sum[:8])
}

// FingerprintFromIssue builds fingerprint input from a CodeIssue.
func FingerprintFromIssue(repository string, issue *ai.CodeIssue) FingerprintInput {
	if issue == nil {
		return FingerprintInput{}
	}
	return FingerprintInput{
		Repository:   repository,
		Category:     issue.Category,
		Source:       issue.Source,
		RuleID:       firstNonEmpty(issue.RuleID, issue.ClusterID),
		File:         issue.File,
		Line:         issue.LineNumber,
		PackageName:  issue.PackageName,
		EvidenceHash: SanitizedEvidenceHash(issue.CodeSnippet),
	}
}

// SanitizedEvidenceHash hashes redacted evidence for fingerprint stability without storing secrets.
func SanitizedEvidenceHash(evidence string) string {
	evidence = SanitizeSecretEvidence(evidence)
	evidence = strings.TrimSpace(evidence)
	if evidence == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(evidence))
	return hex.EncodeToString(sum[:6])
}

// ExtractFingerprintFromBody reads a fingerprint marker from an issue body.
func ExtractFingerprintFromBody(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		for _, marker := range []string{
			"- " + FingerprintBodyMarker,
			FingerprintBodyMarker,
		} {
			if strings.HasPrefix(line, marker) {
				return strings.TrimSpace(strings.TrimPrefix(line, marker))
			}
		}
	}
	return ""
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.TrimPrefix(path, "./")
	return filepath.ToSlash(path)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
