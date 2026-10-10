# Repository Detective — Agent quickstart

**Audience:** AI agents (OpenClaw, Cursor, Claude Desktop, custom automations) that need to drive Repository Detective over HTTP or MCP.

**Product:** Repository Detective — Inspect. Analyze. Improve.  
**Repo:** https://git.commsnet.org/commstech/Repository-Detective

## Two integration modes

| Mode | When to use | Entry point |
|------|-------------|-------------|
| **REST API** | Any HTTP-capable agent | `http://<host>:8081/api/v1/*` |
| **MCP (stdio)** | OpenClaw / Cursor / Claude Desktop tool hosts | `repository-detective-mcp` binary |

Repository Detective also **calls out** to OpenClaw (optional) for advisory AI reviews. That is the reverse direction — see [OPENCLAW_INTEGRATION.md](OPENCLAW_INTEGRATION.md).

## Base URLs

| Environment | URL | Notes |
|-------------|-----|-------|
| Default compose / homelab | `http://127.0.0.1:8081` | `docker-compose.yml` |
| Minimal local trial | `http://127.0.0.1:8080` | `docker-compose.minimal.yml` only |

Use `REPOSITORY_DETECTIVE_PUBLIC_URL` when agents run outside the host network.

## Authentication

Operator JSON API requires an API key (never commit real keys):

```http
X-Repository-Detective-API-Key: <REPOSITORY_DETECTIVE_API_KEY>
```

Also accepted:

```http
Authorization: Bearer <REPOSITORY_DETECTIVE_API_KEY>
```

**Not accepted:** `X-Bugbot-API-Key` (legacy brand — rejected).

Public (no API key): `GET /health`, `GET /onboard`, `/ui/static/*`.

Machine-readable catalog: [openapi.yaml](openapi.yaml) · live `GET /api/v1/openapi.yaml` (API key) · human [API_ROUTES.md](API_ROUTES.md) · MCP [MCP.md](MCP.md).

## Minimal agent loop (curl)

Replace `BASE` and `KEY`.

```bash
BASE=http://127.0.0.1:8081
KEY="$REPOSITORY_DETECTIVE_API_KEY"
H=(-H "X-Repository-Detective-API-Key: $KEY" -H "Content-Type: application/json")

# 1) Liveness (no auth)
curl -sS "$BASE/health"

# 2) Discover product + agent hints
curl -sS "${H[@]}" "$BASE/api/v1/about" | jq .

# 3) Runtime feature flags (no secrets)
curl -sS "${H[@]}" "$BASE/api/v1/status" | jq .

# 4) Trigger a dry-run scan (safe default for agents)
curl -sS "${H[@]}" -d '{
  "owner": "my-org",
  "repository": "my-repo",
  "ref": "main",
  "report_only_dry_run": true
}' "$BASE/api/v1/analyze" | jq .

# 5) Poll scan (use scan id from analyze response)
SCAN_ID=...
curl -sS "${H[@]}" "$BASE/api/v1/scans/$SCAN_ID" | jq .

# 6) List open high-signal findings
curl -sS "${H[@]}" "$BASE/api/v1/findings?status=open&severity=high&limit=50" | jq .

# 7) Optional advisory AI review (only if configured; max_tokens > 0)
curl -sS "${H[@]}" -X POST "$BASE/api/v1/scans/$SCAN_ID/ai-recommendations" | jq .
```

### Safe defaults for agents

- Prefer `report_only_dry_run: true` until the operator enables issue filing.
- Do not enable remediation PRs unless `remediation_pr_enabled` is on and the operator approved.
- Never log or echo API keys, forge tokens, or finding snippets that may contain secrets into public channels.
- High/critical findings are never auto-downgraded by calibration accepts.

## Discovery endpoint (`GET /api/v1/about`)

Useful fields for agents:

| Field | Meaning |
|-------|---------|
| `product_name` / `version` | Identity |
| `api_base_path` | Always `/api/v1` |
| `openapi_url` | Relative path to OpenAPI document |
| `agent_docs_url` | Gitea URL for this guide |
| `mcp_docs_url` | Gitea URL for MCP setup |
| `auth_headers` | Preferred header names |
| `safe_loop` | Remediation loop summary |

## MCP in one minute

```bash
go build -o repository-detective-mcp ./cmd/repository-detective-mcp
export REPOSITORY_DETECTIVE_BASE_URL=http://127.0.0.1:8081
export REPOSITORY_DETECTIVE_API_KEY=...
./repository-detective-mcp
```

Wire that binary into OpenClaw / Cursor as an MCP stdio server. Tool names and schemas: [MCP.md](MCP.md).

## Suggested tool mapping (OpenClaw → REST)

| Agent intent | REST | MCP tool |
|--------------|------|----------|
| Am I reachable? | `GET /health` | `rd_health` |
| What can I do? | `GET /api/v1/about`, `GET /api/v1/status` | `rd_about`, `rd_status` |
| List repos | `GET /api/v1/repos` | `rd_list_repos` |
| Start scan | `POST /api/v1/analyze` | `rd_analyze` |
| List scans for a repo | `GET /api/v1/repos/:id/scans` | (use REST; no fleet `/scans` list) |
| Inspect scan | `GET /api/v1/scans/:id` | `rd_get_scan` |
| Triage findings | `GET /api/v1/findings?repo_id=` | `rd_list_findings` |
| Dashboard rollup | `GET /api/v1/dashboard/summary` | `rd_dashboard_summary` |
| AI advisory | `POST /api/v1/scans/:id/ai-recommendations` | `rd_run_ai_review` |
| Calibration / learning | `/api/v1/calibration/*` (UI: `/ui/learning`) | `rd_calibration_*` |

## Related docs

- [OPENCLAW_INTEGRATION.md](OPENCLAW_INTEGRATION.md) — RD↔OpenClaw both directions  
- [MCP.md](MCP.md) — MCP binary, tools, config snippets  
- [AI_RECOMMENDATIONS.md](AI_RECOMMENDATIONS.md) — advisory reviews  
- [AI_PROVIDERS.md](AI_PROVIDERS.md) — OpenClaw / OpenAI-compatible providers  
- [CONFIGURATION.md](CONFIGURATION.md) — env vars  
- [API_ROUTES.md](API_ROUTES.md) — full route table  
