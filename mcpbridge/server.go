// Package mcpbridge implements a stdio MCP server that proxies to the Repository Detective REST API.
package mcpbridge

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const protocolVersion = "2024-11-05"

// Config holds MCP → REST connection settings.
type Config struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

// ConfigFromEnv loads BASE URL + API key from environment.
func ConfigFromEnv() Config {
	base := firstNonEmpty(
		os.Getenv("REPOSITORY_DETECTIVE_BASE_URL"),
		os.Getenv("REPOSITORY_DETECTIVE_PUBLIC_BASE_URL"),
		os.Getenv("RD_BASE_URL"),
		"http://127.0.0.1:8081",
	)
	return Config{
		BaseURL: strings.TrimRight(strings.TrimSpace(base), "/"),
		APIKey:  strings.TrimSpace(os.Getenv("REPOSITORY_DETECTIVE_API_KEY")),
		Client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      any         `json:"id,omitempty"`
	Result  any         `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// Serve runs the MCP JSON-RPC loop on in/out (typically stdin/stdout).
func Serve(ctx context.Context, cfg Config, in io.Reader, out io.Writer) error {
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 60 * time.Second}
	}
	if cfg.APIKey == "" {
		fmt.Fprintln(os.Stderr, "warning: REPOSITORY_DETECTIVE_API_KEY is empty; authenticated tools will fail")
	}
	enc := json.NewEncoder(out)
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return err
			}
			return nil
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			_ = enc.Encode(rpcResponse{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: "parse error"}})
			continue
		}
		if req.Method == "" {
			continue
		}
		// Notifications have no id / no response.
		if req.ID == nil || string(req.ID) == "null" {
			if req.Method == "notifications/initialized" || req.Method == "initialized" {
				continue
			}
			continue
		}
		var id any
		_ = json.Unmarshal(req.ID, &id)

		result, err := dispatch(ctx, cfg, req.Method, req.Params)
		if err != nil {
			_ = enc.Encode(rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: -32000, Message: err.Error()}})
			continue
		}
		_ = enc.Encode(rpcResponse{JSONRPC: "2.0", ID: id, Result: result})
	}
}

func dispatch(ctx context.Context, cfg Config, method string, params json.RawMessage) (any, error) {
	switch method {
	case "initialize":
		return map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]any{
				"tools":     map[string]any{},
				"resources": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "repository-detective",
				"version": "1.0.0",
			},
			"instructions": "Repository Detective MCP bridge. Prefer dry-run scans (report_only_dry_run=true). See docs/AGENT_QUICKSTART.md and docs/MCP.md.",
		}, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": toolCatalog()}, nil
	case "tools/call":
		return callTool(ctx, cfg, params)
	case "resources/list":
		return map[string]any{
			"resources": []map[string]any{
				{"uri": "repository-detective://openapi", "name": "OpenAPI", "mimeType": "application/yaml", "description": "REST OpenAPI document"},
				{"uri": "repository-detective://docs/agent-quickstart", "name": "Agent quickstart", "mimeType": "text/plain", "description": "Pointer to agent docs"},
			},
		}, nil
	case "resources/read":
		return readResource(ctx, cfg, params)
	default:
		return nil, fmt.Errorf("method not found: %s", method)
	}
}

func toolCatalog() []toolDef {
	obj := func(props map[string]any, required ...string) map[string]any {
		schema := map[string]any{"type": "object", "properties": props}
		if len(required) > 0 {
			schema["required"] = required
		}
		return schema
	}
	return []toolDef{
		{Name: "rd_health", Description: "GET /health — process liveness", InputSchema: obj(map[string]any{})},
		{Name: "rd_about", Description: "GET /api/v1/about — product + agent discovery", InputSchema: obj(map[string]any{})},
		{Name: "rd_status", Description: "GET /api/v1/status — feature flags", InputSchema: obj(map[string]any{})},
		{Name: "rd_openapi", Description: "GET /api/v1/openapi.yaml — OpenAPI document", InputSchema: obj(map[string]any{})},
		{Name: "rd_list_repos", Description: "GET /api/v1/repos", InputSchema: obj(map[string]any{
			"limit": map[string]any{"type": "integer"},
		})},
		{Name: "rd_dashboard_summary", Description: "GET /api/v1/dashboard/summary", InputSchema: obj(map[string]any{})},
		{Name: "rd_list_findings", Description: "GET /api/v1/findings", InputSchema: obj(map[string]any{
			"status":        map[string]any{"type": "string"},
			"severity":      map[string]any{"type": "string"},
			"repository_id": map[string]any{"type": "integer"},
			"limit":         map[string]any{"type": "integer"},
		})},
		{Name: "rd_get_finding", Description: "GET /api/v1/findings/{id}", InputSchema: obj(map[string]any{
			"id": map[string]any{"type": "integer"},
		}, "id")},
		{Name: "rd_get_scan", Description: "GET /api/v1/scans/{scan_id}", InputSchema: obj(map[string]any{
			"scan_id": map[string]any{"type": "string"},
		}, "scan_id")},
		{Name: "rd_analyze", Description: "POST /api/v1/analyze — prefer report_only_dry_run=true", InputSchema: obj(map[string]any{
			"owner":               map[string]any{"type": "string"},
			"repository":          map[string]any{"type": "string"},
			"ref":                 map[string]any{"type": "string"},
			"forge":               map[string]any{"type": "string"},
			"report_only_dry_run": map[string]any{"type": "boolean"},
		}, "owner", "repository")},
		{Name: "rd_ai_config", Description: "GET /api/v1/ai-recommendations/config", InputSchema: obj(map[string]any{})},
		{Name: "rd_run_ai_review", Description: "POST /api/v1/scans/{scan_id}/ai-recommendations", InputSchema: obj(map[string]any{
			"scan_id": map[string]any{"type": "string"},
		}, "scan_id")},
		{Name: "rd_list_pending_ai", Description: "GET /api/v1/ai-recommendations/pending", InputSchema: obj(map[string]any{})},
		{Name: "rd_calibration_summary", Description: "GET /api/v1/calibration/summary", InputSchema: obj(map[string]any{})},
		{Name: "rd_list_calibration", Description: "GET /api/v1/calibration/recommendations", InputSchema: obj(map[string]any{
			"status": map[string]any{"type": "string"},
		})},
		{Name: "rd_accept_calibration", Description: "POST /api/v1/calibration/recommendations/{id}/accept", InputSchema: obj(map[string]any{
			"id": map[string]any{"type": "integer"},
		}, "id")},
		{Name: "rd_reject_calibration", Description: "POST /api/v1/calibration/recommendations/{id}/reject", InputSchema: obj(map[string]any{
			"id": map[string]any{"type": "integer"},
		}, "id")},
	}
}

func callTool(ctx context.Context, cfg Config, params json.RawMessage) (any, error) {
	var p struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid tools/call params: %w", err)
	}
	if p.Arguments == nil {
		p.Arguments = map[string]any{}
	}
	body, status, err := invoke(ctx, cfg, p.Name, p.Arguments)
	if err != nil {
		return toolError(err.Error()), nil
	}
	text := string(body)
	if status >= 400 {
		return toolError(fmt.Sprintf("HTTP %d: %s", status, text)), nil
	}
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
		"isError": false,
	}, nil
}

func toolError(msg string) map[string]any {
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": msg}},
		"isError": true,
	}
}

func invoke(ctx context.Context, cfg Config, name string, args map[string]any) ([]byte, int, error) {
	switch name {
	case "rd_health":
		return do(ctx, cfg, http.MethodGet, "/health", nil, false)
	case "rd_about":
		return do(ctx, cfg, http.MethodGet, "/api/v1/about", nil, true)
	case "rd_status":
		return do(ctx, cfg, http.MethodGet, "/api/v1/status", nil, true)
	case "rd_openapi":
		return do(ctx, cfg, http.MethodGet, "/api/v1/openapi.yaml", nil, true)
	case "rd_list_repos":
		q := url.Values{}
		if v, ok := intArg(args, "limit"); ok {
			q.Set("limit", strconv.Itoa(v))
		}
		path := "/api/v1/repos"
		if enc := q.Encode(); enc != "" {
			path += "?" + enc
		}
		return do(ctx, cfg, http.MethodGet, path, nil, true)
	case "rd_dashboard_summary":
		return do(ctx, cfg, http.MethodGet, "/api/v1/dashboard/summary", nil, true)
	case "rd_list_findings":
		q := url.Values{}
		for _, k := range []string{"status", "severity"} {
			if s, ok := strArg(args, k); ok {
				q.Set(k, s)
			}
		}
		if v, ok := intArg(args, "repository_id"); ok {
			q.Set("repository_id", strconv.Itoa(v))
		}
		if v, ok := intArg(args, "limit"); ok {
			q.Set("limit", strconv.Itoa(v))
		}
		path := "/api/v1/findings"
		if enc := q.Encode(); enc != "" {
			path += "?" + enc
		}
		return do(ctx, cfg, http.MethodGet, path, nil, true)
	case "rd_get_finding":
		id, ok := intArg(args, "id")
		if !ok {
			return nil, 0, fmt.Errorf("id required")
		}
		return do(ctx, cfg, http.MethodGet, fmt.Sprintf("/api/v1/findings/%d", id), nil, true)
	case "rd_get_scan":
		sid, ok := strArg(args, "scan_id")
		if !ok {
			return nil, 0, fmt.Errorf("scan_id required")
		}
		return do(ctx, cfg, http.MethodGet, "/api/v1/scans/"+url.PathEscape(sid), nil, true)
	case "rd_analyze":
		payload := map[string]any{
			"owner":      args["owner"],
			"repository": args["repository"],
		}
		if s, ok := strArg(args, "ref"); ok {
			payload["ref"] = s
		}
		if s, ok := strArg(args, "forge"); ok {
			payload["forge"] = s
		}
		if b, ok := boolArg(args, "report_only_dry_run"); ok {
			payload["report_only_dry_run"] = b
		} else {
			payload["report_only_dry_run"] = true
		}
		return do(ctx, cfg, http.MethodPost, "/api/v1/analyze", payload, true)
	case "rd_ai_config":
		return do(ctx, cfg, http.MethodGet, "/api/v1/ai-recommendations/config", nil, true)
	case "rd_run_ai_review":
		sid, ok := strArg(args, "scan_id")
		if !ok {
			return nil, 0, fmt.Errorf("scan_id required")
		}
		return do(ctx, cfg, http.MethodPost, "/api/v1/scans/"+url.PathEscape(sid)+"/ai-recommendations", map[string]any{}, true)
	case "rd_list_pending_ai":
		return do(ctx, cfg, http.MethodGet, "/api/v1/ai-recommendations/pending", nil, true)
	case "rd_calibration_summary":
		return do(ctx, cfg, http.MethodGet, "/api/v1/calibration/summary", nil, true)
	case "rd_list_calibration":
		path := "/api/v1/calibration/recommendations"
		if s, ok := strArg(args, "status"); ok {
			path += "?status=" + url.QueryEscape(s)
		}
		return do(ctx, cfg, http.MethodGet, path, nil, true)
	case "rd_accept_calibration":
		id, ok := intArg(args, "id")
		if !ok {
			return nil, 0, fmt.Errorf("id required")
		}
		return do(ctx, cfg, http.MethodPost, fmt.Sprintf("/api/v1/calibration/recommendations/%d/accept", id), map[string]any{}, true)
	case "rd_reject_calibration":
		id, ok := intArg(args, "id")
		if !ok {
			return nil, 0, fmt.Errorf("id required")
		}
		return do(ctx, cfg, http.MethodPost, fmt.Sprintf("/api/v1/calibration/recommendations/%d/reject", id), map[string]any{}, true)
	default:
		return nil, 0, fmt.Errorf("unknown tool: %s", name)
	}
}

func readResource(ctx context.Context, cfg Config, params json.RawMessage) (any, error) {
	var p struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	switch p.URI {
	case "repository-detective://openapi":
		body, status, err := do(ctx, cfg, http.MethodGet, "/api/v1/openapi.yaml", nil, true)
		if err != nil {
			return nil, err
		}
		if status >= 400 {
			return nil, fmt.Errorf("HTTP %d: %s", status, string(body))
		}
		return map[string]any{
			"contents": []map[string]any{{
				"uri": p.URI, "mimeType": "application/yaml", "text": string(body),
			}},
		}, nil
	case "repository-detective://docs/agent-quickstart":
		text := "See https://git.commsnet.org/commstech/Repository-Detective/src/branch/main/docs/AGENT_QUICKSTART.md and docs/MCP.md / docs/OPENCLAW_INTEGRATION.md"
		return map[string]any{
			"contents": []map[string]any{{
				"uri": p.URI, "mimeType": "text/plain", "text": text,
			}},
		}, nil
	default:
		return nil, fmt.Errorf("unknown resource: %s", p.URI)
	}
}

func do(ctx context.Context, cfg Config, method, path string, jsonBody any, auth bool) ([]byte, int, error) {
	var bodyReader io.Reader
	if jsonBody != nil {
		b, err := json.Marshal(jsonBody)
		if err != nil {
			return nil, 0, err
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, cfg.BaseURL+path, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	if jsonBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth && cfg.APIKey != "" {
		req.Header.Set("X-Repository-Detective-API-Key", cfg.APIKey)
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	resp, err := cfg.Client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return b, resp.StatusCode, nil
}

func strArg(args map[string]any, key string) (string, bool) {
	v, ok := args[key]
	if !ok || v == nil {
		return "", false
	}
	switch t := v.(type) {
	case string:
		if strings.TrimSpace(t) == "" {
			return "", false
		}
		return t, true
	case float64:
		return strconv.FormatInt(int64(t), 10), true
	case json.Number:
		return t.String(), true
	default:
		return fmt.Sprint(t), true
	}
}

func intArg(args map[string]any, key string) (int, bool) {
	v, ok := args[key]
	if !ok || v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case float64:
		return int(t), true
	case int:
		return t, true
	case int64:
		return int(t), true
	case json.Number:
		i, err := t.Int64()
		return int(i), err == nil
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(t))
		return i, err == nil
	default:
		return 0, false
	}
}

func boolArg(args map[string]any, key string) (bool, bool) {
	v, ok := args[key]
	if !ok || v == nil {
		return false, false
	}
	switch t := v.(type) {
	case bool:
		return t, true
	case string:
		b, err := strconv.ParseBool(t)
		return b, err == nil
	default:
		return false, false
	}
}
