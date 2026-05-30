package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Client performs code analysis via any configured AI provider.
type Client struct {
	transport ChatTransport
	provider  ProviderType
	model     string
	logger    *logrus.Logger
}

func (c *Client) modelName() string {
	if c.model == "" {
		return "default"
	}
	return c.model
}

func (c *Client) chat(ctx context.Context, messages []ChatMessage, temperature float64, maxTokens int) (*ChatResponse, error) {
	return c.transport.Complete(ctx, ChatRequest{
		Model:       c.modelName(),
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	})
}

// TestConnection verifies the AI backend responds.
func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.chat(ctx, []ChatMessage{{Role: "user", Content: "Hello, this is a connection test."}}, 0, 10)
	return err
}

// AnalyzeAttackSurface identifies entry points, attack surface, and trust boundaries.
func (c *Client) AnalyzeAttackSurface(ctx context.Context, req *AttackSurfaceRequest) (*AttackSurfaceResponse, error) {
	prompt := fmt.Sprintf(`You are a security architecture analyst. Analyze the following repository structure and identify the attack surface.

Repository: %s

File Structure:
%s

TASK:
1. Identify all ENTRY POINTS — public-facing functions, HTTP handlers, API endpoints, CLI commands, service interfaces
2. Identify ATTACK SURFACE — data entry points (user input, file reads, network, env vars)
3. Identify TRUST BOUNDARIES — transitions between external/internal/privileged zones

Respond with JSON:
{
  "entry_points": [
    {"file": "src/handler.go", "line": 42, "function_name": "handleLogin", "type": "http_handler", "auth_required": false}
  ],
  "attack_surface": [
    {"file": "src/handler.go", "line": 42, "type": "user_input", "data_flow": "request → handler → db"}
  ],
  "trust_boundaries": [
    {"file": "src/auth.go", "line": 10, "from_zone": "external", "to_zone": "internal", "operation": "auth_check"}
  ]
}`, req.RepositoryName, req.Files)

	resp, err := c.chat(ctx, []ChatMessage{
		{Role: "system", Content: attackSurfaceSystemPrompt()},
		{Role: "user", Content: prompt},
	}, 0.1, 4000)
	if err != nil {
		return nil, fmt.Errorf("attack surface analysis failed: %w", err)
	}

	var result AttackSurfaceResponse
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		c.logger.Warnf("Failed to parse attack surface response as JSON: %v", err)
		return &AttackSurfaceResponse{}, nil
	}
	return &result, nil
}

// RunAuditor runs a specialized auditor agent.
func (c *Client) RunAuditor(ctx context.Context, req *AuditorRequest) (*AuditorResponse, error) {
	resp, err := c.chat(ctx, []ChatMessage{
		{Role: "system", Content: auditorSystemPrompt(req.AuditorType)},
		{Role: "user", Content: buildAuditorPrompt(req)},
	}, 0.1, 4000)
	if err != nil {
		return nil, fmt.Errorf("auditor request failed: %w", err)
	}

	var result AuditorResponse
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		c.logger.Warnf("Failed to parse auditor response: %v", err)
		return &AuditorResponse{}, nil
	}
	return &result, nil
}

// RunDebater runs a debater agent (advocate or counsel).
func (c *Client) RunDebater(ctx context.Context, req *DebaterRequest) (*DebaterResponse, error) {
	var rolePrompt, task string
	if req.Role == "advocate" {
		rolePrompt = "VULNERABILITY ADVOCATE"
		task = "Argue WHY this is exploitable. Find the attack path. Show how an attacker would trigger this."
	} else {
		rolePrompt = "DEFENSE COUNSEL"
		task = "Argue why this is NOT exploitable. Identify mitigating factors. Show why an attacker cannot reach or trigger this."
	}

	prompt := fmt.Sprintf(`You are a security expert acting as %s.

VULNERABILITY CLAIM:
- Type: %s
- File: %s, Line: %d
- Claim: %s
- Evidence: %s
- Severity: %s
- Confidence: %.2f

YOUR TASK:
%s

Analyze the code path, the attack surface, and any mitigating controls.
Provide your confidence (0.0-1.0) and specific arguments.

Respond with JSON:
{
  "confidence": 0.85,
  "arguments": "This is exploitable because..."
}`, rolePrompt, req.Finding.AuditorType, req.Finding.File, req.Finding.Line,
		req.Finding.Hypothesis, req.Finding.Evidence.Code, req.Finding.Severity,
		req.Finding.Confidence, task)

	resp, err := c.chat(ctx, []ChatMessage{
		{Role: "system", Content: debaterSystemPrompt()},
		{Role: "user", Content: prompt},
	}, 0.2, 2000)
	if err != nil {
		return nil, fmt.Errorf("debater request failed: %w", err)
	}

	var result DebaterResponse
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		return &DebaterResponse{Confidence: 0.5, Arguments: truncateString(resp.Content, 200)}, nil
	}
	return &result, nil
}

// GeneratePoC generates a proof-of-concept for a finding.
func (c *Client) GeneratePoC(ctx context.Context, req *PoCRequest) (*PoCResponse, error) {
	prompt := fmt.Sprintf(`You are a security researcher. Generate a proof-of-concept (PoC) that demonstrates this vulnerability.

VULNERABILITY:
- ID: %s
- Type: %s
- Severity: %s
- Description: %s
- Affected Files: %v
- Evidence: %s

TASK:
Generate a concrete, executable PoC that triggers this vulnerability.
For web vulnerabilities, provide a curl command.
For other vulnerabilities, provide an executable script or code snippet.

Respond with JSON:
{
  "type": "curl",
  "command": "curl 'http://target/api/endpoint?param=value' ...",
  "language": "bash",
  "explanation": "This curl command exploits the vulnerability by..."
}`, req.Finding.ID, req.Finding.Category, req.Finding.Severity,
		req.Finding.Description, req.Finding.Files, req.Finding.Evidence.Code)

	resp, err := c.chat(ctx, []ChatMessage{
		{Role: "system", Content: pocSystemPrompt()},
		{Role: "user", Content: prompt},
	}, 0.3, 2000)
	if err != nil {
		return nil, fmt.Errorf("PoC generation failed: %w", err)
	}

	var result PoCResponse
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		return &PoCResponse{Type: "text", Command: truncateString(resp.Content, 500), Explanation: "Raw output"}, nil
	}
	return &result, nil
}

// AnalyzeCode performs code analysis using the configured AI provider.
func (c *Client) AnalyzeCode(ctx context.Context, req *CodeAnalysisRequest) (*CodeAnalysisResult, error) {
	startTime := time.Now()
	c.logger.Infof("Starting code analysis for %s in %s", req.FilePath, req.RepositoryName)

	resp, err := c.chat(ctx, []ChatMessage{
		{Role: "system", Content: codeReviewSystemPrompt()},
		{Role: "user", Content: buildAnalysisPrompt(req)},
	}, 0.1, 4000)
	if err != nil {
		return nil, fmt.Errorf("failed to make AI request: %w", err)
	}

	result, err := parseAnalysisResponse(c.logger, resp.Content, req)
	if err != nil {
		return nil, fmt.Errorf("failed to parse analysis response: %w", err)
	}

	result.AnalysisTime = time.Since(startTime)
	result.ModelUsed = resp.Model
	c.logger.Infof("Code analysis completed in %v, found %d issues", result.AnalysisTime, len(result.Issues))
	return result, nil
}

func buildAuditorPrompt(req *AuditorRequest) string {
	var filesSection strings.Builder
	filesSection.WriteString(fmt.Sprintf("Repository: %s\n\n", req.RepositoryName))
	filesSection.WriteString(fmt.Sprintf("Files to analyze (%d):\n\n", len(req.FileContents)))

	maxCharsPerFile := 8000
	for _, file := range req.FileContents {
		content := file.Content
		if len(content) > maxCharsPerFile {
			content = content[:maxCharsPerFile] + "\n... [truncated]"
		}
		lang := file.Language
		if lang == "" {
			lang = "text"
		}
		filesSection.WriteString(fmt.Sprintf("--- FILE: %s (%s) ---\n", file.Path, lang))
		filesSection.WriteString(content)
		filesSection.WriteString("\n\n")
	}

	if len(req.AttackSurface) > 0 {
		filesSection.WriteString("Known attack surface entries:\n")
		limit := len(req.AttackSurface)
		if limit > 10 {
			limit = 10
		}
		for i := 0; i < limit; i++ {
			entry := req.AttackSurface[i]
			filesSection.WriteString(fmt.Sprintf("- %s:%d (%s) %s\n", entry.File, entry.Line, entry.Type, entry.DataFlow))
		}
		filesSection.WriteString("\n")
	}

	return fmt.Sprintf(`You are a %s security auditor. Analyze the following source code for %s vulnerabilities.

%s
TASK:
For each file, identify %s vulnerabilities and provide:
- File and line number
- The vulnerable code snippet
- Call chain from entry point (if available)
- Severity (critical/high/medium/low)
- Confidence (0.0-1.0)

Respond with JSON:
{
  "findings": [
    {
      "file": "src/db.go",
      "line": 42,
      "hypothesis": "SQL injection via string concatenation in user query",
      "code_snippet": "query := \"SELECT * FROM users WHERE id = \" + userID",
      "call_chain": ["handleRequest() → getUser() → db.Query()"],
      "severity": "critical",
      "confidence": 0.92
    }
  ]
}

If no vulnerabilities found, respond with: {"findings": []}`, req.AuditorType, req.VulnerabilityClass, filesSection.String(), req.VulnerabilityClass)
}

func auditorSystemPrompt(auditorType string) string {
	prompts := map[string]string{
		"sql":       `You are a SQL injection security auditor. You specialize in finding SQL injection vulnerabilities including string concatenation, improper escaping, and dynamic SQL construction.`,
		"xss":       `You are an XSS security auditor. You specialize in finding cross-site scripting vulnerabilities including improper input sanitization, missing output encoding, and DOM manipulation risks.`,
		"auth":      `You are an authentication security auditor. You specialize in finding authentication bypasses, session management flaws, and authorization vulnerabilities.`,
		"injection": `You are a command injection auditor. You specialize in finding command injection, SSRF, and LDAP injection vulnerabilities.`,
		"crypto":    `You are a cryptography security auditor. You specialize in finding hardcoded secrets, weak cryptographic algorithms, IV reuse, and improper key management.`,
		"race":      `You are a race condition auditor. You specialize in finding time-of-check-time-of-use (TOCTOU) vulnerabilities and concurrent access bugs.`,
		"memory":    `You are a memory safety auditor. You specialize in finding buffer overflows, use-after-free, and unsafe memory operations in C/C++ code.`,
		"config":    `You are a configuration security auditor. You specialize in finding insecure defaults, debug mode in production, missing security headers, and environment-based misconfigurations.`,
	}
	if prompt, ok := prompts[auditorType]; ok {
		return prompt
	}
	return `You are a security auditor. Analyze code for security vulnerabilities and provide specific, actionable findings with code references.`
}

func attackSurfaceSystemPrompt() string {
	return `You are a security architecture analyst. Identify entry points, attack surfaces, and trust boundaries from repository structure. Always respond with valid JSON.`
}

func debaterSystemPrompt() string {
	return `You are a security expert analyzing vulnerability claims. You must be technically rigorous — do not accept vague claims. Either prove or disprove exploitability with concrete code references and attack paths.`
}

func pocSystemPrompt() string {
	return `You are a security researcher. Generate realistic, executable proof-of-concept exploits. For web vulnerabilities, use curl. For other vulnerabilities, provide working code snippets. Always explain how the PoC works.`
}

func codeReviewSystemPrompt() string {
	return `You are an expert code reviewer and security analyst. Your task is to analyze code for security vulnerabilities, quality issues, performance problems, bugs, and best practice violations. Always respond with valid JSON focused on actionable issues.`
}

func buildAnalysisPrompt(req *CodeAnalysisRequest) string {
	return fmt.Sprintf(`
Please analyze the following code for potential issues, bugs, security vulnerabilities, and quality improvements.

Repository: %s
File: %s
Language: %s
Context: %s
Analysis Type: %s

Code Content:
--- %s ---
%s
--- end ---

Format your response as JSON with issues, suggestions, and overall_score fields.
`, req.RepositoryName, req.FilePath, req.Language, req.Context, req.AnalysisType, req.Language, req.CodeContent)
}

func parseAnalysisResponse(logger *logrus.Logger, content string, req *CodeAnalysisRequest) (*CodeAnalysisResult, error) {
	var jsonResult struct {
		Issues       []CodeIssue      `json:"issues"`
		Suggestions  []CodeSuggestion `json:"suggestions"`
		OverallScore float64          `json:"overall_score"`
	}

	if err := json.Unmarshal([]byte(content), &jsonResult); err == nil {
		return &CodeAnalysisResult{
			Issues:       jsonResult.Issues,
			Suggestions:  jsonResult.Suggestions,
			OverallScore: jsonResult.OverallScore,
		}, nil
	} else {
		logger.Warnf("Failed to parse JSON response, using fallback: %v", err)
	}
	return &CodeAnalysisResult{
		Issues: []CodeIssue{{
			Severity:    "medium",
			Category:    "quality",
			Title:       "Analysis Response Parsing Failed",
			Description: "The AI response could not be parsed as structured data. Manual review recommended.",
			Confidence:  1.0,
		}},
		Suggestions:  []CodeSuggestion{},
		OverallScore: 0.5,
	}, nil
}

func truncateString(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}
