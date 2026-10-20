# Learning health metrics

Operator-facing metrics for the continuous learning engine. Available on **Dashboard → Learning health** and **Learning & Calibration** (`/ui/learning`).

## Metrics

| Metric | Source | Meaning |
|--------|--------|---------|
| Learning events recorded | `learning_events` | Lifecycle outcomes captured (FP marks, verified closure, scanner failures, dry-runs, duplicates) |
| Pending recommendations | `calibration_recommendations` | Repo-scoped proposals awaiting operator accept/reject |
| Active repo calibration rules | `repo_calibration_rules` | Approved, non-expired per-repo adjustments |
| Structurally grouped findings | `findings.canonical_finding_id` | Repeated patterns grouped under one canonical finding |
| Avg false-positive rate | `rule_reliability_stats` | Per-rule FP rate aggregated for dashboard |
| Scanner failure rate | `scanner_health_history` | Failed/timed-out/parse-failed runs vs total |

## Per-repo isolation

All learning metrics filter by repository unless explicitly marked global. One repo’s ruff noise does not affect another repo’s recommendations.

## Safety

- Findings are never deleted or hidden from reports.
- HIGH/CRITICAL security categories are protected from automatic downgrade.
- LLM sanity gate is off by default and not required for scan correctness.

## API

- `GET /api/v1/calibration/summary` includes `learning_health`
- `POST /api/v1/calibration/recommendations/:id/accept` — repo-scoped only in beta
- `POST /api/v1/calibration/recommendations/:id/reject`
