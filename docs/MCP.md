# MCP server for Repository Detective

Agents that speak the [Model Context Protocol](https://modelcontextprotocol.io/) can drive Repository Detective without hand-written HTTP.

The MCP adapter is a **stdio JSON-RPC bridge** to the existing REST API. It does not embed the database; the main `repository-detective` process must already be running and reachable.

## Build

```bash
cd Repository-Detective
go build -o repository-detective-mcp ./cmd/repository-detective-mcp
```

## Environment

| Variable | Required | Default | Purpose |
|----------|----------|---------|---------|
| `REPOSITORY_DETECTIVE_BASE_URL` | yes* | `http://127.0.0.1:8081` | API origin (no trailing slash) |
| `REPOSITORY_DETECTIVE_API_KEY` | yes | — | Operator API key |
| `REPOSITORY_DETECTIVE_PUBLIC_BASE_URL` | no | — | Alias for base URL |
| `RD_BASE_URL` | no | — | Short alias for base URL |

\*Default is fine for local Docker on port **8081**.

## OpenClaw / Cursor config snippet

```json
{
  "mcpServers": {
    "repository-detective": {
      "command": "/absolute/path/to/repository-detective-mcp",
      "env": {
        "REPOSITORY_DETECTIVE_BASE_URL": "http://127.0.0.1:8081",
        "REPOSITORY_DETECTIVE_API_KEY": "<your-key>"
      }
    }
  }
}
```

Place the equivalent block in your OpenClaw gateway / agent MCP settings (exact file depends on OpenClaw version — typically under `~/.openclaw/`).

**Important:** MCP logs go to **stderr** only. Never write debug logs to stdout (that breaks JSON-RPC).

## Protocol

- Transport: stdio, newline-delimited JSON-RPC 2.0  
- Protocol version: `2024-11-05`  
- Methods: `initialize`, `notifications/initialized`, `tools/list`, `tools/call`, `ping`, `resources/list`, `resources/read`

## Tools

| Tool | Maps to | Notes |
|------|---------|-------|
| `rd_health` | `GET /health` | No API key required on server; bridge still may send one |
| `rd_about` | `GET /api/v1/about` | Agent discovery metadata |
| `rd_status` | `GET /api/v1/status` | Feature flags |
| `rd_openapi` | `GET /api/v1/openapi.yaml` | Machine-readable API |
| `rd_list_repos` | `GET /api/v1/repos` | Optional `limit` |
| `rd_dashboard_summary` | `GET /api/v1/dashboard/summary` | Fleet rollup |
| `rd_list_findings` | `GET /api/v1/findings` | `status`, `severity`, `repository_id`, `limit` |
| `rd_get_finding` | `GET /api/v1/findings/{id}` | |
| `rd_get_scan` | `GET /api/v1/scans/{scan_id}` | |
| `rd_analyze` | `POST /api/v1/analyze` | Prefer `report_only_dry_run=true` |
| `rd_ai_config` | `GET /api/v1/ai-recommendations/config` | |
| `rd_run_ai_review` | `POST /api/v1/scans/{scan_id}/ai-recommendations` | Requires AI recommendations enabled |
| `rd_list_pending_ai` | `GET /api/v1/ai-recommendations/pending` | |
| `rd_calibration_summary` | `GET /api/v1/calibration/summary` | |
| `rd_list_calibration` | `GET /api/v1/calibration/recommendations` | |
| `rd_accept_calibration` | `POST /api/v1/calibration/recommendations/{id}/accept` | Repo-scoped expand for globals |
| `rd_reject_calibration` | `POST /api/v1/calibration/recommendations/{id}/reject` | |

## Resources

| URI | Description |
|-----|-------------|
| `repository-detective://openapi` | OpenAPI YAML body |
| `repository-detective://docs/agent-quickstart` | Pointer + summary for agents |

## Manual smoke test

```bash
export REPOSITORY_DETECTIVE_BASE_URL=http://127.0.0.1:8081
export REPOSITORY_DETECTIVE_API_KEY=...
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"smoke","version":"0"}}}' \
  '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' \
  '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"rd_health","arguments":{}}}' \
  | ./repository-detective-mcp
```

## Related

- [AGENT_QUICKSTART.md](AGENT_QUICKSTART.md)  
- [OPENCLAW_INTEGRATION.md](OPENCLAW_INTEGRATION.md)  
- [openapi.yaml](openapi.yaml)  
- [API_ROUTES.md](API_ROUTES.md)  
