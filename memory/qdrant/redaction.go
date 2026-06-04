package qdrant

import (
	"fmt"
	"regexp"
	"strings"

	"git.commsnet.org/commstech/bugbot/redact"
)

var (
	secretPattern = regexp.MustCompile(`(?i)(api[_-]?key|secret|token|password|credential)[^\n]{0,40}`)
	codeBlock     = regexp.MustCompile("(?s)```.+?```")
)

// CAHFindingPayload is the redacted Qdrant payload for cah_findings compatibility.
type CAHFindingPayload struct {
	ID               string  `json:"id"`
	FindingID        string  `json:"finding_id"`
	BugClass         string  `json:"bug_class"`
	Severity         string  `json:"severity"`
	FilePath         string  `json:"file_path"`
	Line             int     `json:"line,omitempty"`
	Description      string  `json:"description"`
	ExploitPrimitive string  `json:"exploit_primitive,omitempty"`
	Subsystem        string  `json:"subsystem,omitempty"`
	CWEID            string  `json:"cwe_id,omitempty"`
	Timestamp        string  `json:"timestamp,omitempty"`
	GiteaIssue       int     `json:"gitea_issue,omitempty"`
	Target           string  `json:"target,omitempty"`
	Verdict          string  `json:"verdict,omitempty"`
	Confidence       float64 `json:"confidence,omitempty"`
	RunID            string  `json:"run_id,omitempty"`
	Repository       string  `json:"repository,omitempty"`
	Source           string  `json:"source,omitempty"`
	RuleID           string  `json:"rule_id,omitempty"`
	IssueURL         string  `json:"issue_url,omitempty"`
	ClusterID        string  `json:"cluster_id,omitempty"`
}

// RedactSummary returns a safe, non-secret description for storage and embedding.
func RedactSummary(title, description, evidence string) string {
	parts := []string{
		strings.TrimSpace(title),
		strings.TrimSpace(description),
	}
	raw := strings.TrimSpace(strings.Join(parts, ". "))
	if raw == "" {
		return "Repository finding detected by deterministic scanner."
	}
	raw = redact.SecretEvidence(raw)
	raw = codeBlock.ReplaceAllString(raw, "[code redacted]")
	raw = secretPattern.ReplaceAllString(raw, "[credential-like value redacted]")
	if len(raw) > 512 {
		raw = raw[:512] + "…"
	}
	return raw
}

// BuildCAHPayload maps a match input to a redacted cah_findings payload.
func BuildCAHPayload(input MatchInput, issueURL string, issueNumber int, runID string) CAHFindingPayload {
	pointID := fingerprintValue(input)
	desc := RedactSummary(input.Title, input.Description, "")
	return CAHFindingPayload{
		ID:               pointID,
		FindingID:        pointID,
		BugClass:         normalizeBugClass(input.Category, input.Source),
		Severity:         normalizeSeverity(input.Severity),
		FilePath:         safePath(input.File, input.Line),
		Line:             input.Line,
		Description:      desc,
		ExploitPrimitive: normalizePrimitive(input.Category, input.Source, input.RuleID),
		Subsystem:        subsystemFromPath(input.File),
		Confidence:       input.Confidence,
		RunID:            runID,
		Repository:       input.Repository,
		Source:           input.Source,
		RuleID:           input.RuleID,
		GiteaIssue:       issueNumber,
		Target:           input.Repository,
		Verdict:          input.Verdict,
		IssueURL:         issueURL,
		ClusterID:        input.ClusterID,
	}
}

// PayloadContainsRawSecrets reports whether payload text still looks sensitive.
func PayloadContainsRawSecrets(payload CAHFindingPayload) bool {
	combined := strings.Join([]string{
		payload.Description, payload.ExploitPrimitive, payload.Subsystem, payload.FilePath,
	}, "\n")
	sanitized := redact.SecretEvidence(combined)
	return sanitized != combined
}

// EmbeddingText builds normalized, redacted text for vector embedding.
func EmbeddingText(input MatchInput) string {
	return strings.TrimSpace(strings.Join([]string{
		"source: " + strings.ToLower(strings.TrimSpace(input.Source)),
		"rule: " + strings.ToLower(strings.TrimSpace(input.RuleID)),
		"category: " + strings.ToLower(strings.TrimSpace(input.Category)),
		"path_class: " + pathClass(input.File),
		"verdict: " + strings.ToLower(strings.TrimSpace(input.Verdict)),
		"summary: " + RedactSummary(input.Title, input.Description, ""),
	}, "\n"))
}

func normalizeSeverity(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical", "crit":
		return "Critical"
	case "high", "error":
		return "High"
	case "medium", "warning", "warn":
		return "Medium"
	default:
		return "Low"
	}
}

func normalizeBugClass(category, source string) string {
	cat := strings.ToLower(strings.TrimSpace(category))
	if cat != "" {
		return cat
	}
	return strings.ToLower(strings.TrimSpace(source))
}

func normalizePrimitive(category, source, ruleID string) string {
	value := strings.ToLower(strings.TrimSpace(ruleID))
	if value == "" {
		value = strings.ToLower(strings.TrimSpace(category))
	}
	if value == "" {
		value = strings.ToLower(strings.TrimSpace(source))
	}
	return value
}

func subsystemFromPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "repository"
	}
	parts := strings.Split(path, "/")
	if len(parts) > 1 {
		return parts[0]
	}
	return "repository"
}

func safePath(path string, line int) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if line > 0 {
		return fmt.Sprintf("%s:%d", path, line)
	}
	return path
}

func pathClass(path string) string {
	path = strings.ToLower(path)
	switch {
	case strings.Contains(path, "test"), strings.Contains(path, "fixture"):
		return "test_fixture"
	case strings.Contains(path, "docs"), strings.HasSuffix(path, ".md"):
		return "documentation"
	case strings.Contains(path, "vendor"), strings.Contains(path, "node_modules"):
		return "vendor"
	case strings.Contains(path, "config"), strings.HasSuffix(path, ".env"), strings.HasSuffix(path, ".yaml"), strings.HasSuffix(path, ".yml"):
		return "config_template"
	default:
		return "source"
	}
}
