package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"yourusername/gitea-bugbot/analyzers"
)

// ============================================================================
// NEW CAH PIPELINE METHODS
// ============================================================================

// AttackSurfaceRequest is the input for attack surface analysis
type AttackSurfaceRequest struct {
	RepositoryName string
	Files         string // file tree summary
}

// AttackSurfaceResponse is the output of attack surface analysis
type AttackSurfaceResponse struct {
	EntryPoints     []analyzers.EntryPoint
	AttackSurface   []analyzers.AttackSurfaceEntry
	TrustBoundaries []analyzers.TrustBoundary
}

// AnalyzeAttackSurface identifies entry points, attack surface, and trust boundaries
func (c *OpenWebUIClient) AnalyzeAttackSurface(ctx context.Context, req *AttackSurfaceRequest) (*AttackSurfaceResponse, error) {
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

	openReq := &OpenWebUIRequest{
		Model:       "default",
		Messages:    []OpenWebUIMessage{{Role: "system", Content: getSystemPrompt()}, {Role: "user", Content: prompt}},
		Stream:      false,
		Temperature: 0.1,
		MaxTokens:  4000,
	}

	resp, err := c.makeRequest(ctx, openReq)
	if err != nil {
		return nil, fmt.Errorf("attack surface analysis failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenWebUI")
	}

	// Parse JSON response
	var result AttackSurfaceResponse
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		c.logger.Warnf("Failed to parse attack surface response as JSON: %v", err)
		// Return empty result rather than failing
		return &AttackSurfaceResponse{}, nil
	}

	return &result, nil
}

// AuditorRequest is the input for an auditor agent
type AuditorRequest struct {
	RepositoryName     string
	VulnerabilityClass string
	Files             []interface{} // gitea.RepositoryContent
	AttackSurface    []analyzers.AttackSurfaceEntry
	AuditorType     string
}

// AuditorFinding is a candidate finding from an auditor
type AuditorFinding struct {
	File        string
	Line        int
	Hypothesis  string
	CodeSnippet string
	CallChain   []string
	Severity    string
	Confidence  float64
}

// AuditorResponse is the output of an auditor agent
type AuditorResponse struct {
	Findings []AuditorFinding
}

// RunAuditor runs a specialized auditor agent
func (c *OpenWebUIClient) RunAuditor(ctx context.Context, req *AuditorRequest) (*AuditorResponse, error) {
	prompt := c.buildAuditorPrompt(req)

	openReq := &OpenWebUIRequest{
		Model:       "default",
		Messages:    []OpenWebUIMessage{{Role: "system", Content: getAuditorSystemPrompt(req.AuditorType)}, {Role: "user", Content: prompt}},
		Stream:      false,
		Temperature: 0.1,
		MaxTokens:  4000,
	}

	resp, err := c.makeRequest(ctx, openReq)
	if err != nil {
		return nil, fmt.Errorf("auditor request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return &AuditorResponse{}, nil
	}

	var result AuditorResponse
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		c.logger.Warnf("Failed to parse auditor response: %v", err)
		return &AuditorResponse{}, nil
	}

	return &result, nil
}

func (c *OpenWebUIClient) buildAuditorPrompt(req *AuditorRequest) string {
	return fmt.Sprintf(`You are a %s security auditor. Analyze the following code for %s vulnerabilities.

Repository: %s

Files to analyze: %d files

TASK:
For each file, identify %s vulnerabilities and provide:
- File and line number
- The vulnerable code snippet
- Call chain from entry point (if available)
- Severity (critical/high/medium/low)
- Confidence (0.0-1.0)

Respond with JSON array of findings:
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

If no vulnerabilities found, respond with: {"findings": []}`, req.AuditorType, req.VulnerabilityClass, req.RepositoryName, len(req.Files), req.VulnerabilityClass)
}

func getAuditorSystemPrompt(auditorType string) string {
	prompts := map[string]string{
		"sql":          `You are a SQL injection security auditor. You specialize in finding SQL injection vulnerabilities including string concatenation, improper escaping, and dynamic SQL construction.`,
		"xss":          `You are an XSS security auditor. You specialize in finding cross-site scripting vulnerabilities including improper input sanitization, missing output encoding, and DOM manipulation risks.`,
		"auth":         `You are an authentication security auditor. You specialize in finding authentication bypasses, session management flaws, and authorization vulnerabilities.`,
		"injection":    `You are a command injection auditor. You specialize in finding command injection, SSRF, and LDAP injection vulnerabilities.`,
		"crypto":       `You are a cryptography security auditor. You specialize in finding hardcoded secrets, weak cryptographic algorithms, IV reuse, and improper key management.`,
		"race":        `You are a race condition auditor. You specialize in finding time-of-check-time-of-use (TOCTOU) vulnerabilities and concurrent access bugs.`,
		"memory":      `You are a memory safety auditor. You specialize in finding buffer overflows, use-after-free, and unsafe memory operations in C/C++ code.`,
		"config":      `You are a configuration security auditor. You specialize in finding insecure defaults, debug mode in production, missing security headers, and environment-based misconfigurations.`,
	}
	if p, ok := prompts[auditorType]; ok {
		return p
	}
	return `You are a security auditor. Analyze code for security vulnerabilities and provide specific, actionable findings with code references.`
}

// DebaterRequest is the input for a debater agent
type DebaterRequest struct {
	Finding analyzers.CandidateFinding
	Role   string // "advocate" or "counsel"
}

// DebaterResponse is the output of a debater agent
type DebaterResponse struct {
	Confidence float64
	Arguments string
}

// RunDebater runs a debater agent (advocate or counsel)
func (c *OpenWebUIClient) RunDebater(ctx context.Context, req *DebaterRequest) (*DebaterResponse, error) {
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

	openReq := &OpenWebUIRequest{
		Model:       "default",
		Messages:    []OpenWebUIMessage{{Role: "system", Content: getDebaterSystemPrompt()}, {Role: "user", Content: prompt}},
		Stream:      false,
		Temperature: 0.2,
		MaxTokens:  2000,
	}

	resp, err := c.makeRequest(ctx, openReq)
	if err != nil {
		return nil, fmt.Errorf("debater request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return &DebaterResponse{Confidence: 0.5, Arguments: "No response"}, nil
	}

	var result DebaterResponse
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		return &DebaterResponse{Confidence: 0.5, Arguments: resp.Choices[0].Message.Content[:min(200, len(resp.Choices[0].Message.Content))]}, nil
	}

	return &result, nil
}

func getDebaterSystemPrompt() string {
	return `You are a security expert analyzing vulnerability claims. You must be technically rigorous — do not accept vague claims. Either prove or disprove exploitability with concrete code references and attack paths.`
}

// PoCRequest is the input for PoC generation
type PoCRequest struct {
	Finding analyzers.DedupedFinding
}

// PoCResponse is the output of PoC generation
type PoCResponse struct {
	Type        string
	Command     string
	Language    string
	Explanation string
}

// GeneratePoC generates a proof-of-concept for a finding
func (c *OpenWebUIClient) GeneratePoC(ctx context.Context, req *PoCRequest) (*PoCResponse, error) {
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

	openReq := &OpenWebUIRequest{
		Model:       "default",
		Messages:    []OpenWebUIMessage{{Role: "system", Content: getPoCSystemPrompt()}, {Role: "user", Content: prompt}},
		Stream:      false,
		Temperature: 0.3,
		MaxTokens:  2000,
	}

	resp, err := c.makeRequest(ctx, openReq)
	if err != nil {
		return nil, fmt.Errorf("PoC generation failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return &PoCResponse{Type: "none", Command: "", Explanation: "No PoC generated"}, nil
	}

	var result PoCResponse
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		return &PoCResponse{Type: "text", Command: resp.Choices[0].Message.Content[:min(500, len(resp.Choices[0].Message.Content))], Explanation: "Raw output"}, nil
	}

	return &result, nil
}

func getPoCSystemPrompt() string {
	return `You are a security researcher. Generate realistic, executable proof-of-concept exploits. For web vulnerabilities, use curl. For other vulnerabilities, provide working code snippets. Always explain how the PoC works.`
}

// OpenWebUIClient handles communication with OpenWebUI server
type OpenWebUIClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
	logger     *logrus.Logger
}

// OpenWebUIRequest represents a request to OpenWebUI
type OpenWebUIRequest struct {
	Model       string                 `json:"model"`
	Messages    []OpenWebUIMessage     `json:"messages"`
	Stream      bool                   `json:"stream"`
	Temperature float64                `json:"temperature"`
	MaxTokens  int                    `json:"max_tokens"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// OpenWebUIMessage represents a message in the conversation
type OpenWebUIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenWebUIResponse represents a response from OpenWebUI
type OpenWebUIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// CodeAnalysisRequest represents a request for code analysis
type CodeAnalysisRequest struct {
	RepositoryName string
	FilePath       string
	CodeContent    string
	Language       string
	Context        string
	AnalysisType   string // "security", "quality", "bug", "all"
}

// CodeAnalysisResult represents the result of code analysis
type CodeAnalysisResult struct {
	Issues        []CodeIssue
	Suggestions   []CodeSuggestion
	OverallScore  float64
	AnalysisTime  time.Duration
	ModelUsed     string
}

// CodeIssue represents a detected issue in the code
type CodeIssue struct {
	Severity      string `json:"severity"`       // "low", "medium", "high", "critical"
	Category      string `json:"category"`       // "security", "performance", "maintainability", "bug"
	Title         string `json:"title"`
	Description   string `json:"description"`
	LineNumber    int    `json:"line_number,omitempty"`
	ColumnNumber  int    `json:"column_number,omitempty"`
	CodeSnippet   string `json:"code_snippet,omitempty"`
	Confidence    float64 `json:"confidence"`
}

// CodeSuggestion represents a suggested improvement
type CodeSuggestion struct {
	Type          string `json:"type"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	CodeExample   string `json:"code_example,omitempty"`
	Priority      string `json:"priority"`
}

// NewOpenWebUIClient creates a new OpenWebUI client
func NewOpenWebUIClient(baseURL, token string, logger *logrus.Logger) *OpenWebUIClient {
	return &OpenWebUIClient{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		logger: logger,
	}
}

// AnalyzeCode performs code analysis using OpenWebUI AI
func (c *OpenWebUIClient) AnalyzeCode(ctx context.Context, req *CodeAnalysisRequest) (*CodeAnalysisResult, error) {
	startTime := time.Now()
	c.logger.Infof("Starting code analysis for %s in %s", req.FilePath, req.RepositoryName)

	// Create the analysis prompt
	prompt := c.createAnalysisPrompt(req)
	
	// Prepare the OpenWebUI request
	openWebUIReq := &OpenWebUIRequest{
		Model:       "default", // Use default model, can be configured
		Messages: []OpenWebUIMessage{
			{
				Role:    "system",
				Content: c.getSystemPrompt(),
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream:      false,
		Temperature: 0.1, // Low temperature for consistent analysis
		MaxTokens:  4000,
	}

	// Make the request to OpenWebUI
	resp, err := c.makeRequest(ctx, openWebUIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make OpenWebUI request: %w", err)
	}

	// Parse the response
	result, err := c.parseAnalysisResponse(resp, req)
	if err != nil {
		return nil, fmt.Errorf("failed to parse analysis response: %w", err)
	}

	result.AnalysisTime = time.Since(startTime)
	result.ModelUsed = resp.Model

	c.logger.Infof("Code analysis completed in %v, found %d issues", 
		result.AnalysisTime, len(result.Issues))

	return result, nil
}

// createAnalysisPrompt creates a comprehensive prompt for code analysis
func (c *OpenWebUIClient) createAnalysisPrompt(req *CodeAnalysisRequest) string {
	prompt := fmt.Sprintf(`
Please analyze the following code for potential issues, bugs, security vulnerabilities, and quality improvements.

Repository: %s
File: %s
Language: %s
Context: %s
Analysis Type: %s

Code Content:
```%s
%s
```

Please provide a comprehensive analysis including:
1. Security vulnerabilities (if any)
2. Code quality issues
3. Potential bugs
4. Performance improvements
5. Best practice violations
6. Specific suggestions for fixes

For each issue found, please provide:
- Severity level (low/medium/high/critical)
- Category (security/performance/maintainability/bug)
- Clear description of the problem
- Line number if applicable
- Code snippet showing the issue
- Confidence level (0.0-1.0)
- Suggested fix or improvement

Format your response as JSON with the following structure:
{
  "issues": [
    {
      "severity": "high",
      "category": "security",
      "title": "SQL Injection Vulnerability",
      "description": "User input is directly concatenated into SQL query",
      "line_number": 42,
      "code_snippet": "query := \"SELECT * FROM users WHERE id = \" + userInput",
      "confidence": 0.95
    }
  ],
  "suggestions": [
    {
      "type": "improvement",
      "title": "Use Parameterized Queries",
      "description": "Replace string concatenation with parameterized queries",
      "code_example": "query := \"SELECT * FROM users WHERE id = ?\"",
      "priority": "high"
    }
  ],
  "overall_score": 0.75
}

Please ensure the response is valid JSON and focuses on actionable, specific issues.
`, req.RepositoryName, req.FilePath, req.Language, req.Context, req.AnalysisType, req.Language, req.CodeContent)

	return prompt
}

// getSystemPrompt returns the system prompt for the AI
func (c *OpenWebUIClient) getSystemPrompt() string {
	return `You are an expert code reviewer and security analyst. Your task is to analyze code for:

1. Security vulnerabilities (SQL injection, XSS, CSRF, etc.)
2. Code quality issues (complexity, maintainability, readability)
3. Performance problems (inefficient algorithms, memory leaks, etc.)
4. Potential bugs and edge cases
5. Best practice violations
6. Code style and formatting issues

You must:
- Be thorough but focused on actionable issues
- Provide specific, concrete examples
- Rate severity accurately (low/medium/high/critical)
- Give confidence levels based on certainty
- Suggest specific fixes when possible
- Always respond with valid JSON
- Focus on the most important issues first

Your analysis should help developers improve their code quality and security.`
}

// makeRequest makes an HTTP request to OpenWebUI
func (c *OpenWebUIClient) makeRequest(ctx context.Context, req *OpenWebUIRequest) (*OpenWebUIResponse, error) {
	// Convert request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	// Make the request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenWebUI returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var openWebUIResp OpenWebUIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openWebUIResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &openWebUIResp, nil
}

// parseAnalysisResponse parses the AI response into structured analysis results
func (c *OpenWebUIClient) parseAnalysisResponse(resp *OpenWebUIResponse, req *CodeAnalysisRequest) (*CodeAnalysisResult, error) {
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in OpenWebUI response")
	}

	content := resp.Choices[0].Message.Content
	c.logger.Debugf("Raw AI response: %s", content)

	// Try to parse as JSON first
	var jsonResult struct {
		Issues       []CodeIssue       `json:"issues"`
		Suggestions  []CodeSuggestion  `json:"suggestions"`
		OverallScore float64           `json:"overall_score"`
	}

	if err := json.Unmarshal([]byte(content), &jsonResult); err == nil {
		return &CodeAnalysisResult{
			Issues:       jsonResult.Issues,
			Suggestions:  jsonResult.Suggestions,
			OverallScore: jsonResult.OverallScore,
		}, nil
	}

	// If JSON parsing fails, try to extract information from text
	c.logger.Warnf("Failed to parse JSON response, attempting text extraction: %v", err)
	return c.extractFromText(content, req)
}

// extractFromText attempts to extract analysis results from text response
func (c *OpenWebUIClient) extractFromText(content string, req *CodeAnalysisRequest) (*CodeAnalysisResult, error) {
	// This is a fallback method when JSON parsing fails
	// In a production environment, you might want to retry with a different prompt
	// or implement more sophisticated text parsing

	result := &CodeAnalysisResult{
		Issues:       []CodeIssue{},
		Suggestions:  []CodeSuggestion{},
		OverallScore: 0.5, // Default score
	}

	// Add a generic issue indicating parsing failure
	result.Issues = append(result.Issues, CodeIssue{
		Severity:    "medium",
		Category:    "quality",
		Title:       "Analysis Response Parsing Failed",
		Description: "The AI response could not be parsed as structured data. Manual review recommended.",
		Confidence:  1.0,
	})

	return result, nil
}

// TestConnection tests the connection to OpenWebUI
func (c *OpenWebUIClient) TestConnection(ctx context.Context) error {
	req := &OpenWebUIRequest{
		Model: "default",
		Messages: []OpenWebUIMessage{
			{
				Role:    "user",
				Content: "Hello, this is a connection test.",
			},
		},
		Stream:      false,
		Temperature: 0.0,
		MaxTokens:  10,
	}

	_, err := c.makeRequest(ctx, req)
	return err
}
