package preinstall

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/redact"
	"git.commsnet.org/commstech/repository-detective/store"
)

// Failure stage labels for operator debugging.
const (
	FailureStageURLValidation    = "url_validation"
	FailureStageClone            = "clone"
	FailureStageSandboxSetup     = "sandbox_setup"
	FailureStageScannerSetup     = "scanner_setup"
	FailureStageScannerTimeout   = "scanner_timeout"
	FailureStageReportGeneration = "report_generation"
	FailureStageUnknown          = "unknown"
)

// RiskScoreUnavailable marks audits that did not complete scoring.
const RiskScoreUnavailable = -1

var failureEnvSecretPattern = regexp.MustCompile(`(?i)(REPOSITORY_DETECTIVE_|GITEA_)[A-Z0-9_]*TOKEN=[^\s]+`)

// ApplyAuditFailure records a failed audit without implying a safe score.
func ApplyAuditFailure(req *store.AuditRequest, stage, rawErr string, sandbox SandboxMeta) {
	stage = strings.TrimSpace(stage)
	if stage == "" {
		stage = ClassifyFailureStage(rawErr)
	}
	sanitized := SanitizeFailureMessage(rawErr)
	now := time.Now().UTC()

	req.Status = store.AuditStatusFailed
	req.Recommendation = store.AuditRecommendationAuditFailed
	req.RiskScore = RiskScoreUnavailable
	req.Error = sanitized
	req.FinishedAt = &now

	summary := map[string]any{
		"failure_stage":    stage,
		"failure_reason":   sanitized,
		"next_action":      NextActionForStage(stage),
		"conclusion":       "none",
		"issues_created":   0,
		"prs_created":      0,
		"risk_unavailable": true,
	}
	if sandbox.SandboxID != "" {
		summary["sandbox"] = sandbox
	}
	if raw, err := json.Marshal(summary); err == nil {
		req.SummaryJSON = raw
	}
}

// ClassifyFailureStage maps error text to a failure stage.
func ClassifyFailureStage(msg string) string {
	lower := strings.ToLower(strings.TrimSpace(msg))
	switch {
	case strings.Contains(lower, "only https"), strings.Contains(lower, "private"), strings.Contains(lower, "localhost"),
		strings.Contains(lower, "invalid url"), strings.Contains(lower, "not allowed"), strings.Contains(lower, "resolve"):
		return FailureStageURLValidation
	case strings.Contains(lower, "git clone"), strings.Contains(lower, "git operation"), strings.Contains(lower, "rev-parse"):
		return FailureStageClone
	case strings.Contains(lower, "path escape"), strings.Contains(lower, "path traversal"), strings.Contains(lower, "sandbox"),
		strings.Contains(lower, "temp dir"), strings.Contains(lower, "workspace"), strings.Contains(lower, "file count"),
		strings.Contains(lower, "repo size"), strings.Contains(lower, "file size"):
		return FailureStageSandboxSetup
	case strings.Contains(lower, "timeout"), strings.Contains(lower, "deadline"), strings.Contains(lower, "context canceled"):
		return FailureStageScannerTimeout
	case strings.Contains(lower, "scanner"), strings.Contains(lower, "binary_missing"), strings.Contains(lower, "not installed"):
		return FailureStageScannerSetup
	case strings.Contains(lower, "report"), strings.Contains(lower, "disclosure"):
		return FailureStageReportGeneration
	default:
		return FailureStageUnknown
	}
}

// SanitizeFailureMessage redacts secrets and shortens operator-facing errors.
func SanitizeFailureMessage(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "audit failed before completion"
	}
	msg = redact.SecretEvidence(msg)
	msg = failureEnvSecretPattern.ReplaceAllString(msg, "[REDACTED]")
	if len(msg) > 500 {
		msg = msg[:500] + "…"
	}
	return msg
}

// NextActionForStage suggests operator next steps.
func NextActionForStage(stage string) string {
	switch stage {
	case FailureStageURLValidation:
		return "Use a public HTTPS repository URL. Private and loopback hosts are blocked by default."
	case FailureStageClone:
		return "Verify git is installed, outbound HTTPS is allowed, and the repository URL is reachable."
	case FailureStageSandboxSetup:
		return "Review sandbox limits (repo size, file count, path rules) or retry with a smaller repository."
	case FailureStageScannerSetup:
		return "Install missing scanner binaries or reduce audit depth to quick."
	case FailureStageScannerTimeout:
		return "Increase preinstall_sandbox_timeout_seconds or retry with audit depth quick."
	case FailureStageReportGeneration:
		return "Check audit logs and database connectivity, then retry the audit."
	default:
		return "Review the failure reason and audit logs, then retry."
	}
}

// RiskScoreDisplay returns UI/API label for risk score.
func RiskScoreDisplay(audit store.AuditRequest) string {
	if audit.Status == store.AuditStatusFailed || audit.RiskScore < 0 {
		return "unavailable"
	}
	return fmt.Sprintf("%d / 100", audit.RiskScore)
}

// RecommendationDisplay returns UI/API recommendation label.
func RecommendationDisplay(audit store.AuditRequest) string {
	if audit.Status == store.AuditStatusFailed {
		return "audit failed"
	}
	switch audit.Recommendation {
	case store.AuditRecommendationSafe:
		return "safe"
	case store.AuditRecommendationCaution:
		return "caution"
	case store.AuditRecommendationDoNotInstall:
		return "do not install"
	case store.AuditRecommendationAuditFailed:
		return "audit failed"
	default:
		return audit.Recommendation
	}
}

// FailureStageFromSummary reads failure_stage from summary JSON.
func FailureStageFromSummary(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var summary map[string]any
	if json.Unmarshal(raw, &summary) != nil {
		return ""
	}
	if s, ok := summary["failure_stage"].(string); ok {
		return s
	}
	return ""
}

// AuditConcluded returns false when audit failed before a risk conclusion.
func AuditConcluded(audit store.AuditRequest) bool {
	return audit.Status == store.AuditStatusCompleted
}
