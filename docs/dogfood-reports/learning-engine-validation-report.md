# Learning engine validation report

Date: 2026-06-02 (post-calibration review)  
Mode: **report-only dry-run** (no issue filing, no PR creation)

## Repos exercised

| Repository | Issue creation | PR creation | Learning events | Calibration |
|------------|----------------|-------------|-----------------|-------------|
| commstech/Repository-Detective (product) | 0 | 0 | Schema v20 + prior scans | No repo rules accepted |
| commstech/netmapper | 0 | 0 | 16 events seeded + dry-run history | 2 rules accepted (graph orphan) |
| commstech/commsnet_optimizer | 0 | 0 | Events seeded | 1 rule accepted, 1 rejected |

## Checks

- [x] Product repo active-present findings remain 0
- [x] Limited issue filing NOT enabled
- [x] All-repo scan NOT started
- [x] Global calibration accept blocked in API
- [x] HIGH/CRITICAL findings not auto-downgraded
- [x] Findings remain visible after calibration
- [x] LLM sanity gate disabled by default
- [x] Structural dedup verified on benchmark fixture
- [x] Per-repo isolation — netmapper rules do not apply to commstech/Repository-Detective

## Learning health (post-migration)

| Metric | Value |
|--------|-------|
| Learning events | 16 |
| Pending recommendations | 0 (after operator review) |
| Active repo calibration rules | 3 |
| Token burn | 0 (LLM gate off) |

## Overfitting protections observed

- Recommendations require evidence threshold per repo
- Global scope recommendations rejected at accept API
- Operator rejected nextcloud_scripts pending more evidence
- Accepted rules expire in 90 days

## Deployment note

Production SQLite under `data/` is owned by container user (`unms`). Run migrations and operator DB updates via Docker root mount or restart container after image rebuild.

## Remaining follow-ups

1. Rebuild `repository-detective` container to serve learning health API on live instance
2. Live report-only dry-run scan trigger on each repo (optional repeat)
