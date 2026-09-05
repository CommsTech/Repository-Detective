package openclaw

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
)

const systemPrompt = `You are an advisory security reviewer for Repository Detective.
Deterministic scanners are the source of truth. You review redacted finding summaries only.
Respond with strict JSON matching this schema:
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
Never recommend auto-closing issues, auto-suppressing findings, or creating PRs.
Do not downgrade critical/high findings without explicit human review classification needs_human_review.`

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
	body, err := json.Marshal(pkt)
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
			{Role: "user", Content: string(body)},
		},
		Temperature: 0.1,
		MaxTokens:   cfg.MaxTokensPerScan,
	})
	if err != nil {
		result.Status = "timeout"
		if ctx.Err() != nil {
			result.Status = "timeout"
		}
		result.Error = err.Error()
		return result, nil
	}
	content := strings.TrimSpace(resp.Content)
	content = extractJSON(content)
	var parsed ReviewResponse
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		result.Error = "malformed response: " + err.Error()
		return result, nil
	}
	if parsed.ReviewID == "" {
		parsed.ReviewID = reviewID
	}
	result.Status = "completed"
	result.Response = &parsed
	result.OverallAssessment = parsed.OverallAssessment
	result.RecommendationsCount = len(parsed.Recommendations)
	result.FindingsSent = len(pkt.Findings)
	return result, nil
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
	}
	return strings.TrimSpace(s)
}
