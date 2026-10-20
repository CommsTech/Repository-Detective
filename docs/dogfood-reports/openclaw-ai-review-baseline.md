# OpenClaw AI review baseline

Recorded: 2026-06-10  
Latest git commit: `effee25`

## Product dogfood

| Metric | Value |
|---|---:|
| Active-present | 0 |
| High/critical | 0 |
| Live revision | `9e10a40` |

## OpenClaw / AI config (homelab)

| Item | Value |
|---|---|
| `ai_provider` | `openclaw` |
| `ai_base_url` | configured (`https://ai.example.local:18789/v1`) |
| `ai_model` | `openclaw/software-engineer` |
| `enable_llm_auditors` | false (beta_standard / depth 2) |
| `llm_sanity_gate_enabled` | false |
| `openclaw_ai_review_enabled` | **not implemented yet** (this sprint) |

## Existing AI integration

| Path | Status |
|---|---|
| LLM auditors (depth ≥ 3) | off by default |
| LLM sanity gate | stub only; disabled |
| Attack surface / debater / PoC | requires `enable_llm_auditors` |
| `/api/v1/ai/status` | exists |
| OpenClaw advisory review | **missing** — no second-opinion packet path |

## Redaction today

- `redact.SecretEvidence` for snippets/logs
- `secret_scan_redact: true`
- Runner/container log redaction
- Finding persistence uses `redactSnippet` on evidence

## Scan policy

- `scan_profile: standard_deterministic`
- Deterministic scanners are source of truth
- Container scanning disabled post-rollback
- Runner delegation off

## Marketing blockers (unchanged)

1. Gitea wiki HTTP 500
2. Screenshots / external install
3. Pre-install 5-repo scale test
4. 2+ non-product beta scans
5. OpenClaw advisory review proof (this sprint)
