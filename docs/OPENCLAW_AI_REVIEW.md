# OpenClaw advisory AI review

Repository Detective uses **deterministic scanners first**. OpenClaw is an optional **second-opinion layer** for false-positive reduction, explanations, and calibration suggestions.

## Architecture

```text
Deterministic scan
  → normalized findings (fingerprints, severity, confidence, evidence)
  → redaction / sanitization
  → optional OpenClaw review packet (≤N findings)
  → advisory recommendations stored
  → operator accepts or rejects (no auto-apply)
```

## Default posture

| Setting | Default |
|---|---|
| `openclaw_ai_review_enabled` | `false` |
| `openclaw_ai_max_tokens_per_scan` | `0` (no API calls) |
| `openclaw_ai_send_source_snippets` | `false` |
| `openclaw_ai_send_full_files` | `false` |
| `openclaw_ai_allow_preinstall` | `false` |
| `openclaw_ai_advisory_only` | `true` |
| `openclaw_ai_require_operator_approval` | `true` |

## Allowed inputs (default)

- Finding title, rule ID, severity, confidence, scanner/source
- Redacted path, description, evidence
- Scanner coverage summary
- Repository language/framework summary (when available)
- Issue lifecycle hints (seen before, closed as false positive)

## Disallowed by default

- Raw secrets, tokens, private keys, `.env` contents
- Full source files
- Registry credentials, runner secrets, Gitea tokens
- PHI/PII (redacted when `openclaw_ai_redact_pii: true`)

## OpenClaw output (advisory)

- `possible_false_positive` / `likely_true_positive` / `needs_human_review`
- Suggested severity/confidence adjustments
- Suggested repo-scoped calibration
- Remediation summary and evidence gaps

OpenClaw **must not** auto-file issues, auto-close, auto-suppress, create PRs, or change calibration.

## API

- `GET /api/v1/openclaw/config`
- `GET /api/v1/scans/:scan_id/ai-review`
- `POST /api/v1/scans/:scan_id/ai-review` (manual trigger; does not modify findings)
- `POST /api/v1/ai-review/recommendations/:id/accept` (marks accepted; calibration draft only)
- `POST /api/v1/ai-review/recommendations/:id/reject`

## UI

- Scan detail: **AI Advisory Review** panel
- Learning: OpenClaw recommendations queue
- Configure: OpenClaw settings section

See also `docs/beta/OPENCLAW_AI_REVIEW_BETA.md`.
