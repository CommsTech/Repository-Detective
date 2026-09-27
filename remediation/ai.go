package remediation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const providerAdvisorSystemPrompt = `You are Repository Detective's remediation coach (OpenClaw software-engineer agent).
Deterministic scanners remain the source of truth. Enrich the provided draft plan only.

OUTPUT CONTRACT (mandatory):
- Reply with a single JSON object only. No markdown fences. No prose before or after.

JSON schema:
{
  "summary": "string",
  "fix_strategy": "string",
  "affected_files": ["string"],
  "required_tests": ["string"],
  "validation_commands": ["string"],
  "regression_risk": "low | medium | high",
  "fix_complexity": "small | medium | large",
  "blocked_reasons": ["string"],
  "requires_human_review": true
}
Rules:
- Never invent secret values or recommend committing credentials.
- Prefer small, testable fixes with clear validation commands.
- Keep requires_human_review true unless the draft was already low-risk and safe.
- Do not recommend auto-merge, force-push, or disabling scanners.
- validation_commands must be allowlisted-style checks (go test, go vet, staticcheck, hadolint) when applicable.`

// ChatCompleter is the AI chat surface used by ProviderAdvisor (typically *ai.Client).
type ChatCompleter interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResult, error)
}

// ChatRequest is a provider-agnostic remediation chat request.
type ChatRequest struct {
	Messages    []ChatMessage
	Temperature float64
	MaxTokens   int
	User        string
}

// ChatMessage is a provider-agnostic chat message for remediation enrichment.
type ChatMessage struct {
	Role    string
	Content string
}

// ChatResult is a normalized chat completion result.
type ChatResult struct {
	Content string
}

// ProviderAdvisor enriches remediation plans via the configured AI provider (e.g. OpenClaw).
type ProviderAdvisor struct {
	Client  ChatCompleter
	Timeout time.Duration
	MaxTok  int
}

type providerPlanEnrichment struct {
	Summary             string   `json:"summary"`
	FixStrategy         string   `json:"fix_strategy"`
	AffectedFiles       []string `json:"affected_files"`
	RequiredTests       []string `json:"required_tests"`
	ValidationCommands  []string `json:"validation_commands"`
	RegressionRisk      string   `json:"regression_risk"`
	FixComplexity       string   `json:"fix_complexity"`
	BlockedReasons      []string `json:"blocked_reasons"`
	RequiresHumanReview *bool    `json:"requires_human_review"`
}

// SuggestPlan calls the configured AI provider to enrich a draft remediation plan.
func (a ProviderAdvisor) SuggestPlan(fctx FindingContext, draft Plan) (Plan, error) {
	if a.Client == nil {
		return draft, fmt.Errorf("ai client not configured")
	}
	timeout := a.Timeout
	if timeout <= 0 {
		timeout = 180 * time.Second
	}
	maxTok := a.MaxTok
	if maxTok <= 0 {
		maxTok = 1200
	}
	payload, err := json.Marshal(map[string]any{
		"finding": map[string]any{
			"fingerprint":  fctx.Fingerprint,
			"category":     fctx.Category,
			"severity":     fctx.Severity,
			"source":       fctx.Source,
			"rule_id":      fctx.RuleID,
			"title":        fctx.Title,
			"summary":      fctx.Summary,
			"confidence":   fctx.Confidence,
			"file_path":    fctx.FilePath,
			"line":         fctx.Line,
			"package_name": fctx.PackageName,
			"repo":         fctx.RepoFullName,
		},
		"draft_plan": map[string]any{
			"summary":               draft.Summary,
			"fix_strategy":         draft.FixStrategy,
			"affected_files":        draft.AffectedFiles,
			"required_tests":        draft.RequiredTests,
			"validation_commands":   draft.ValidationCommands,
			"regression_risk":       draft.RegressionRisk,
			"fix_complexity":       draft.FixComplexity,
			"safe_for_auto_pr":      draft.SafeForAutoPR,
			"requires_human_review": draft.RequiresHumanReview,
			"blocked_reasons":       draft.BlockedReasons,
		},
	})
	if err != nil {
		return draft, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	resp, err := a.Client.Chat(ctx, ChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: providerAdvisorSystemPrompt},
			{Role: "user", Content: string(payload)},
		},
		Temperature: 0.1,
		MaxTokens:   maxTok,
		User:        remediationSessionUser(fctx),
	})
	if err != nil {
		return draft, err
	}
	if resp == nil || strings.TrimSpace(resp.Content) == "" {
		return draft, fmt.Errorf("empty AI remediation response")
	}
	enriched, err := parseProviderPlanEnrichment(resp.Content)
	if err != nil {
		return draft, err
	}
	out := draft
	if s := strings.TrimSpace(enriched.Summary); s != "" {
		out.Summary = s
	}
	if s := strings.TrimSpace(enriched.FixStrategy); s != "" {
		out.FixStrategy = s
	}
	if len(enriched.AffectedFiles) > 0 {
		out.AffectedFiles = enriched.AffectedFiles
	}
	if len(enriched.RequiredTests) > 0 {
		out.RequiredTests = enriched.RequiredTests
	}
	if len(enriched.ValidationCommands) > 0 {
		out.ValidationCommands = enriched.ValidationCommands
	}
	if s := normalizeRisk(enriched.RegressionRisk); s != "" {
		out.RegressionRisk = s
	}
	if s := normalizeComplexity(enriched.FixComplexity); s != "" {
		out.FixComplexity = s
	}
	if len(enriched.BlockedReasons) > 0 {
		out.BlockedReasons = enriched.BlockedReasons
	}
	if enriched.RequiresHumanReview != nil {
		out.RequiresHumanReview = *enriched.RequiresHumanReview
	} else {
		out.RequiresHumanReview = true
	}
	out.Advisory = true
	out.SafeForAutoPR = false
	return out, nil
}

func parseProviderPlanEnrichment(raw string) (providerPlanEnrichment, error) {
	raw = extractJSONObjectLocal(raw)
	var out providerPlanEnrichment
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return providerPlanEnrichment{}, fmt.Errorf("parse AI remediation JSON: %w", err)
	}
	return out, nil
}

func extractJSONObjectLocal(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```JSON")
		s = strings.TrimPrefix(s, "```")
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}
	start := strings.Index(s, "{")
	if start < 0 {
		return s
	}
	depth := 0
	inString := false
	escape := false
	for i := start; i < len(s); i++ {
		ch := s[i]
		if inString {
			if escape {
				escape = false
				continue
			}
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(s[start : i+1])
			}
		}
	}
	return strings.TrimSpace(s[start:])
}

func normalizeRisk(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case RiskLow, RiskMedium, RiskHigh:
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return ""
	}
}

func normalizeComplexity(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case ComplexitySmall, ComplexityMedium, ComplexityLarge:
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return ""
	}
}

func remediationSessionUser(fctx FindingContext) string {
	if fp := strings.TrimSpace(fctx.Fingerprint); fp != "" {
		return "rd:remediation:" + fp
	}
	if fctx.FindingID > 0 {
		return fmt.Sprintf("rd:remediation:finding-%d", fctx.FindingID)
	}
	return "rd:remediation"
}
