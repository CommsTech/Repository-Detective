# OpenClaw ↔ Repository Detective

OpenClaw and Repository Detective integrate in **two directions**. Confusing them is the most common setup mistake.

```text
┌────────────────────┐         advisory review (RD calls OpenClaw)
│ Repository         │ ──────► OpenAI-compatible /v1/chat/completions
│ Detective          │         (optional AI recommendations)
│  :8081 /api/v1     │
└─────────▲──────────┘
          │
          │ REST or MCP (OpenClaw / agent calls RD)
          │
┌─────────┴──────────┐
│ OpenClaw agent     │  stdio MCP: repository-detective-mcp
│ or any AI agent    │  or plain HTTP + API key
└────────────────────┘
```

## Direction A — Agent uses Repository Detective (you want this for automation)

1. Run Repository Detective (`http://127.0.0.1:8081`).
2. Give the agent `REPOSITORY_DETECTIVE_API_KEY`.
3. Prefer **MCP** ([MCP.md](MCP.md)) or follow [AGENT_QUICKSTART.md](AGENT_QUICKSTART.md) REST loop.

Typical agent jobs: trigger dry-run scans, list findings, read dashboard summary, accept/reject calibration recommendations, optionally request AI advisory reviews.

## Direction B — Repository Detective uses OpenClaw (advisory reviews)

RD sends a **redacted** finding packet to an OpenAI-compatible chat endpoint and expects **strict JSON** back.

### Enable (operator)

```bash
# .env
REPOSITORY_DETECTIVE_AI_RECOMMENDATIONS_ENABLED=true
REPOSITORY_DETECTIVE_AI_RECOMMENDATIONS_PROVIDER=openclaw
REPOSITORY_DETECTIVE_AI_RECOMMENDATIONS_ENDPOINT=http://127.0.0.1:18789/v1
# Required: reviews are no-ops while this is 0
REPOSITORY_DETECTIVE_AI_RECOMMENDATIONS_MAX_TOKENS_PER_SCAN=4000
REPOSITORY_DETECTIVE_AI_API_KEY=<gateway-token-if-needed>
```

Also see [AI_RECOMMENDATIONS.md](AI_RECOMMENDATIONS.md) and [AI_PROVIDERS.md](AI_PROVIDERS.md).

### OpenClaw gateway

Default chat base: `http://127.0.0.1:18789/v1` (`/chat/completions`).

From Docker, use a reachable host IP or compose network DNS — not `127.0.0.1` inside the container unless OpenClaw shares the network namespace. Self-signed TLS: mount CA under `certs/` (see [DEPLOYMENT_ISSUES.md](DEPLOYMENT_ISSUES.md)).

### Response contract (must be JSON)

```json
{
  "review_id": "string",
  "overall_assessment": "string",
  "recommendations": [
    {
      "finding_fingerprint": "string",
      "classification": "string",
      "suggested_action": "string",
      "suggested_severity": "string",
      "suggested_confidence": "string",
      "reason": "string",
      "evidence_gaps": []
    }
  ]
}
```

Non-JSON OpenClaw replies mark the review `failed` (findings are unchanged — advisory only).

### Operator API (after a scan)

```http
GET  /api/v1/ai-recommendations/config
POST /api/v1/scans/{scan_id}/ai-recommendations
GET  /api/v1/ai-recommendations/pending
POST /api/v1/ai-recommendations/{id}/accept
POST /api/v1/ai-recommendations/{id}/reject
```

Legacy aliases `/openclaw/*` and `/ai-review/*` still work.

Accept/reject creates **calibration drafts only** — findings are never auto-mutated by AI recommendations.

## Safety

- Advisory layer is off by default (`enabled=false`, `max_tokens_per_scan=0`).
- Source snippets / full files are off by default; redaction defaults on.
- Secrets/security categories cannot be bulk-accepted via calibration.
- Prefer dry-run scans from agents until filing policy is intentional.

## Related

- [AGENT_QUICKSTART.md](AGENT_QUICKSTART.md)  
- [MCP.md](MCP.md)  
- [AI_RECOMMENDATIONS.md](AI_RECOMMENDATIONS.md)  
- [AI_PROVIDERS.md](AI_PROVIDERS.md)  
- [openapi.yaml](openapi.yaml)  
