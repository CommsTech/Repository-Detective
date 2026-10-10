# AI recommendations

Repository Detective uses **deterministic scanners first**. AI recommendations are an optional **second-opinion layer** for false-positive reduction, explanations, and calibration suggestions.

## Flow

```
Deterministic scanners → CAH harness (uncertainty scoring) → redacted packet → AI provider → advisory JSON → operator accept/reject
```

## Configuration (preferred keys)

| Key | Default |
|-----|---------|
| `ai_recommendations_enabled` | `false` |
| `ai_recommendations_provider` | `openclaw` |
| `ai_recommendations_max_tokens_per_scan` | `0` (no API calls) |
| `ai_recommendations_send_source_snippets` | `false` |
| `ai_recommendations_send_full_files` | `false` |
| `ai_recommendations_advisory_only` | `true` |
| `ai_recommendations_require_operator_approval` | `true` |
| `ai_recommendations_use_cah_harness` | `true` |

Legacy `openclaw_ai_*` keys remain honored when preferred keys are unset.

## CAH gating

Only findings where AI review may change operator confidence are sent:

- Low/medium confidence
- Possible false positives
- High duplicate/stale patterns
- Never broad full-scan default

High-confidence critical findings are not auto-downgraded.

## API

- `GET /api/v1/ai-recommendations/config`
- `POST /api/v1/scans/:id/ai-recommendations`
- `GET /api/v1/ai-recommendations/pending`

Legacy `/openclaw/*` and `/ai-review/*` routes remain for compatibility.

## Policy

AI recommendations **must not** auto-file issues, auto-close, auto-suppress, create PRs, or change calibration without operator action.
