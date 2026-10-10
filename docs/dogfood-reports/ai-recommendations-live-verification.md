# AI Recommendations live verification

**Date:** 2026-06-10  
**Revision:** `rc-e3e19ec`

## Default state (verified live)

| Setting | Live value |
|---------|------------|
| `enabled` | **false** |
| `max_tokens_per_scan` | **0** |
| `provider` | `openclaw` (internal provider name) |
| `feature` API label | `ai_recommendations` |
| `cah_enabled` | true |
| `advisory_only` | true |
| `send_source_snippets` | false |
| `send_full_files` | false |

## UI

| Surface | Wording |
|---------|---------|
| Configure | **AI recommendations** (not "OpenClaw AI review") |
| Scan detail | AI recommendations panel |
| Learning | AI recommendations queue |

Legacy `/api/v1/openclaw/config` alias still available for compatibility.

## Controlled live test

**Not run** this pass — token budget remains 0; no provider call made.

Prior sprint noted OpenClaw returns non-JSON → stored as failed review without auto-changes.

## Acceptance

| Item | Status |
|------|--------|
| Provider-neutral naming live | **PASS** |
| Disabled by default | **PASS** |
| `/api/v1/ai-recommendations/config` | **PASS** |
| Strict JSON on provider | **partial** (known provider-side gap) |

## Post-test state

AI Recommendations remain **disabled**, token budget **0**.
