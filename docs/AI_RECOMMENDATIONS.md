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
| `ai_first_run_rule_tuning_enabled` | `true` (no-op until AI recommendations can invoke) |
| `ai_recommendations_auto_after_scan` | `false` |

Legacy `openclaw_ai_*` keys remain honored when preferred keys are unset.

## First-run AI repo rule tuning

When `ai_first_run_rule_tuning_enabled` is on and AI recommendations can invoke, a newly connected repository’s **first completed scan** runs one CAH advisory review. Repo-scoped `report_only` calibrations are applied **only** when the AI classification is an explicit false-positive confirmation (`possible_false_positive`) and the suggested action is `calibrate_repo_scope`. Protected SEC/CVE/secret rules are never tuned this way. The pass is recorded under operator evidence `ai.first_run_rule_tune.<repo_id>` so it runs once.

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

AI recommendations **must not** auto-file issues, auto-close findings, create PRs, or mute protected security rules.

Calibration apply paths:

- **Operator accept** (API / Learning) — explicit human action.
- **Optional first-run AI rule tuning** — operator-enabled; applies `report_only` only after AI confirms a false positive on non-protected rules (once per repo).
- **Confidence-only `calibration_auto_apply`** — separate deterministic learning path; skipped during the first-run AI tune recompute so soft-FP stats cannot race ahead of AI confirmation.
