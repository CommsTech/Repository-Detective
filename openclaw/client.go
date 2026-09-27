package openclaw

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
)

const systemPrompt = `You are Repository Detective's advisory security reviewer (OpenClaw agent).
Deterministic scanners are the source of truth. You only triage redacted finding summaries.

OUTPUT CONTRACT (mandatory):
- Reply with a single JSON object only. No markdown fences. No prose before or after.
- JSON must be strictly valid: escape newlines inside strings as \\n (never put raw line breaks inside quotes).
- Every recommendations[].fingerprint MUST exactly match a finding fingerprint from the packet.
- Provide one recommendation per packet finding when possible.

JSON schema:
{
  "review_id": "string",
  "overall_assessment": "string",
  "recommendations": [
    {
      "fingerprint": "string",
      "classification": "likely_true_positive | possible_false_positive | needs_human_review",
      "suggested_action": "fix | calibrate_repo_scope | leave_visible | escalate | ignore_none",
      "suggested_severity": "critical | high | medium | low | info",
      "suggested_confidence": "high | medium | low",
      "reason": "string",
      "evidence_gaps": ["string"]
    }
  ]
}

Policy:
- Never recommend auto-closing issues, auto-suppressing findings, creating PRs, or rotating secrets yourself.
- If finding.protected_from_downgrade is true OR coach_lane is "protected_coach":
  classification MUST be likely_true_positive or needs_human_review (never possible_false_positive),
  and do not lower suggested_severity below the packet severity.
- Use coach_focus as the primary lens for each finding.
- Prefer calibrate_repo_scope / leave_visible for repetitive lint/tech-debt noise.
- Prefer fix / escalate for reachable secrets, vulns, and confirmed insecure code patterns.`

// Client calls OpenClaw for advisory review.
type Client struct {
	transport ai.ChatTransport
	model     string
	timeout   time.Duration
}

// NewClient creates an OpenClaw review client.
func NewClient(cfg Config, transport ai.ChatTransport) (*Client, error) {
	cfg = cfg.Normalized()
	if transport == nil {
		return nil, fmt.Errorf("ai transport required")
	}
	if !cfg.EndpointConfigured() {
		return nil, fmt.Errorf("openclaw endpoint not configured")
	}
	return &Client{
		transport: transport,
		model:     cfg.EffectiveModel(),
		timeout:   time.Duration(cfg.TimeoutSeconds) * time.Second,
	}, nil
}

// Review sends a redacted packet and parses the advisory response.
func (c *Client) Review(ctx context.Context, cfg Config, reviewID string, pkt ReviewPacket) (ReviewResult, error) {
	if c == nil {
		return ReviewResult{ReviewID: reviewID, Status: "failed", Error: "client is nil"}, fmt.Errorf("client is nil")
	}
	result := ReviewResult{ReviewID: reviewID, Status: "failed", Model: c.model}
	cfg = cfg.Normalized()
	if cfg.MaxTokensPerScan <= 0 {
		result.Status = "skipped"
		result.Error = "max tokens per scan is 0"
		return result, nil
	}
	if strings.TrimSpace(pkt.HarnessVersion) == "" {
		pkt.HarnessVersion = HarnessVersion
	}
	if strings.TrimSpace(pkt.Task) == "" {
		pkt.Task = "advisory_finding_triage"
	}
	if strings.TrimSpace(pkt.ScanID) == "" {
		// keep empty; agents still get review_id via prompt
	}
	userPrompt, err := buildReviewUserPrompt(reviewID, pkt)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := c.transport.Complete(ctx, ai.ChatRequest{
		Model: c.model,
		Messages: []ai.ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.1,
		MaxTokens:   cfg.MaxTokensPerScan,
		User:        "rd:ai-review:" + reviewID,
	})
	if err != nil {
		result.Status = classifyTransportError(err, ctx)
		result.Error = err.Error()
		return result, nil
	}
	content := RepairRelaxedJSON(ExtractJSONObject(resp.Content))
	var parsed ReviewResponse
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		snippet := content
		if len(snippet) > 240 {
			snippet = snippet[:240] + "…"
		}
		result.Error = "malformed response: " + err.Error() + "; body=" + snippet
		return result, nil
	}
	if parsed.ReviewID == "" {
		parsed.ReviewID = reviewID
	}
	parsed = sanitizeRecommendations(parsed, pkt)
	result.Status = "completed"
	result.Response = &parsed
	result.OverallAssessment = parsed.OverallAssessment
	result.RecommendationsCount = len(parsed.Recommendations)
	result.FindingsSent = len(pkt.Findings)
	return result, nil
}

func buildReviewUserPrompt(reviewID string, pkt ReviewPacket) (string, error) {
	body, err := json.Marshal(pkt)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("Task: triage the findings in this Repository Detective review packet.\n")
	b.WriteString("review_id: ")
	b.WriteString(reviewID)
	b.WriteString("\n")
	b.WriteString("Return ONLY the JSON object defined in the system contract.\n")
	b.WriteString("Packet JSON:\n")
	b.Write(body)
	return b.String(), nil
}

func classifyTransportError(err error, ctx context.Context) string {
	if err == nil {
		return "failed"
	}
	if ctx != nil && ctx.Err() != nil {
		return "timeout"
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "deadline exceeded") || strings.Contains(msg, "context canceled") || strings.Contains(msg, "timeout") {
		return "timeout"
	}
	return "failed"
}

func sanitizeRecommendations(parsed ReviewResponse, pkt ReviewPacket) ReviewResponse {
	allowed := map[string]FindingInput{}
	for _, f := range pkt.Findings {
		allowed[f.Fingerprint] = f
	}
	out := make([]Recommendation, 0, len(parsed.Recommendations))
	for _, rec := range parsed.Recommendations {
		f, ok := allowed[rec.Fingerprint]
		if !ok {
			continue
		}
		if f.ProtectedFromDowngrade {
			if rec.Classification == ClassPossibleFalsePositive {
				rec.Classification = ClassNeedsHumanReview
			}
			if severityRank(rec.SuggestedSeverity) < severityRank(f.Severity) {
				rec.SuggestedSeverity = f.Severity
			}
		}
		out = append(out, rec)
	}
	parsed.Recommendations = out
	return parsed
}

// RepairRelaxedJSON escapes literal newlines/tabs/carriage-returns inside JSON
// string values. Models often emit multi-line "reason" fields without \n escapes,
// which encoding/json rejects with: invalid character '\n' in string literal.
func RepairRelaxedJSON(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 32)
	inString := false
	escape := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inString {
			if escape {
				b.WriteByte(ch)
				escape = false
				continue
			}
			if ch == '\\' {
				b.WriteByte(ch)
				escape = true
				continue
			}
			if ch == '"' {
				inString = false
				b.WriteByte(ch)
				continue
			}
			switch ch {
			case '\n':
				b.WriteString(`\n`)
			case '\r':
				b.WriteString(`\r`)
			case '\t':
				b.WriteString(`\t`)
			default:
				b.WriteByte(ch)
			}
			continue
		}
		if ch == '"' {
			inString = true
		}
		b.WriteByte(ch)
	}
	return b.String()
}

// ExtractJSONObject pulls the first top-level JSON object from model output.
// Handles prose prefixes and optional markdown fences (historical OpenClaw failure mode).
func ExtractJSONObject(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
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
		return strings.TrimSpace(s)
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
