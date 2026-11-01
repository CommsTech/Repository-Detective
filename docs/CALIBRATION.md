# Calibration (per-repository learning)

Repository Detective calibrates noisy findings **per repository** by default.

## Scoping rules

| Scope | Applies to | Expires |
|-------|------------|---------|
| Repository suppression | Single repo fingerprint/rule | Operator review or disable |
| Global suppression | Proven universal false positives only | Requires explicit approval |
| Graph homelab profile | Infra repos via `homelab_infra` scan profile | Profile-bound, not global |

## Evidence used

- Duplicate rate across scans
- Resolved-verified vs false-positive disposition
- Scanner stability (grype DB, parse vs unavailable)
- Rule actionability and developer feedback

## Safety

- High-confidence security findings are **not** downgraded by repo-local graph noise rules alone.
- Cross-repo learning may **suggest** calibration; auto-apply remains off (`calibration_auto_apply: false`).
- Secrets / High / Critical remain protected from automatic downgrade/suppression (unchanged authority — RD-025).

## Transparency (RD-025)

Operators can inspect every calibration change:

| Question | Where |
|----------|--------|
| What rule changed? | `source` + `rule_id` on recommendation / history |
| Why? | `reason` + confidence |
| Who/what proposed it? | `deterministic_calibration` (actor_source) |
| Automatically applied? | Always **false** today (`calibration_auto_apply` remains off) |
| Scope? | `repo` (or global tile that expands to per-repo rules) |
| Previous vs new behavior? | `current_action` → `recommended_action` |
| Revert? | `POST /api/v1/calibration/recommendations/:id/revert` (accepted only) |

API:

- `GET /api/v1/calibration/recommendations?status=proposed|accepted|rejected|reverted`
- `GET /api/v1/calibration/history` — all statuses with previous/proposed behavior fields
- `POST …/accept` | `…/reject` | `…/revert`

Revert **expires** linked `repo_calibration_rules` (via `recommendation_id`). It does not rewrite forge issues or invent a full transactional undo of historical findings.

## API / storage

- Suppressions: `finding_suppressions` + calibration recommendations tables
- Matcher cache: per-repository via `calibration.Matcher.LoadRepository`
- Accept installs repo-scoped `repo_calibration_rules` only (findings stay visible)

See also: `docs/beta/CALIBRATION_BETA_POLICY.md`
