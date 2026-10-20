# OpenClaw advisory AI review verification

Recorded: 2026-06-10

## Config states tested

| State | Result |
|---|---|
| Default (disabled) | `enabled: false`, `max_tokens_per_scan: 0` |
| Test window (env override) | `enabled: true`, `max_tokens_per_scan: 2000`, `max_findings: 5` |

## Endpoint

| Check | Result |
|---|---|
| Endpoint configured | yes (`ai_base_url` homelab OpenClaw) |
| `POST /api/v1/ai/test-connection` | **reachable** (`last_test_ok: true`) |
| Model | `openclaw/software-engineer` |

## Controlled review (`cfcb1e05419859d6`)

| Field | Value |
|---|---|
| Review ID | `air-4f97d8c7b817802e` |
| Findings sent | 1 (low/static, redacted packet) |
| Redaction count | applied (no secrets in stored packet) |
| Response status | `failed` (OpenClaw returned non-JSON body) |
| Recommendations | 0 |
| Automatic finding changes | **0** |
| Issues created | **0** |
| PRs created | **0** |

Packet excerpt (sanitized): one `low` finding `OPT-HTTP-CLIENT-PER-CALL` with path `health/reliability.go` — no credentials in JSON.

## Clean product scan (`95a5551881e866d4`)

| Field | Value |
|---|---|
| Status | `skipped` |
| Reason | `no findings to review` |

## Rollback

- Removed test env overrides (`openclaw_ai_review_enabled`, token budget).
- Live `/api/v1/openclaw/config`: `enabled: false`, `max_tokens_per_scan: 0`.

## Notes

- OpenClaw connection is healthy for metadata test; chat completion may return non-JSON (operator should align OpenClaw prompt/response format).
- Malformed responses do not fail deterministic scans or modify findings.
