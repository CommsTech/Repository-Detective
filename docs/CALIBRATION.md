# Deterministic calibration

Repository Detective learns **locally** from suppressions, false positives, verified fixes, and issue lifecycle — **no LLM required**.

## Inputs (local DB only)

- Findings, issues created, suppressions, false-positive marks
- Verified closures, scanner outcomes
- Rule IDs, sources, categories, severities

**Not used:** raw code, embeddings, issue bodies, private repo metadata.

## Outputs

- `rule_false_positive_rate`, `rule_actionable_rate`
- `scanner_reliability_score` (via scan-quality metrics)
- `recommended_default_action` per rule
- Proposed suppressions and report-only changes

## Tables (migration 15)

- `calibration_rule_stats`
- `calibration_recommendations`

## API

```text
GET  /api/v1/calibration/summary
GET  /api/v1/calibration/recommendations
POST /api/v1/calibration/recommendations/:id/accept
POST /api/v1/calibration/recommendations/:id/reject
POST /api/v1/calibration/recompute
```

## Background job

```yaml
calibration_enabled: true
calibration_interval_hours: 24
calibration_min_findings_for_recommendation: 20
calibration_auto_apply: false   # never auto-apply by default
```

Accepting a recommendation may create a global suppression; rejecting persists the decision.

## Community boundary

See [PRIVACY.md](PRIVACY.md) — no cross-instance sharing in this phase.
