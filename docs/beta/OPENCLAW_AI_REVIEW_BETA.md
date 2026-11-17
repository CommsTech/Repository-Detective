# OpenClaw AI review — beta notes

## Beta default

**Off.** Set `openclaw_ai_max_tokens_per_scan > 0` and `openclaw_ai_review_enabled: true` only for controlled homelab tests.

## Safe demo script

1. Confirm `/api/v1/openclaw/config` shows `enabled: false`.
2. Run a product scan (deterministic only).
3. Enable review temporarily with a small token budget (≤2000) and ≤5 findings.
4. `POST /api/v1/scans/{scan_id}/ai-review`
5. Verify recommendations stored; findings unchanged; no issues/PRs created.
6. Disable review again.

## Marketing

Not required for private beta. When enabled in demos, label clearly:

> Advisory only — deterministic scanners remain source of truth.
