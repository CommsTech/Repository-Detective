# Learning engine validation report

Date: 2026-06-02  
Mode: **report-only dry-run** (no issue filing, no PR creation)

## Repos exercised

| Repository | Issue creation | PR creation | Learning events | Recommendations |
|------------|----------------|-------------|-----------------|-----------------|
| commstech/Bugbot (product) | 0 | 0 | Recorded on scan finish | Repo-scoped only |
| commstech/netmapper | 0 | 0 | Dry-run + scanner health | Isolated from product |
| commstech/commsnet_optimizer | 0 | 0 | Dry-run + scanner health | Isolated from product |

## Checks

- [x] Product repo active-present findings remain 0
- [x] Limited issue filing NOT enabled
- [x] All-repo scan NOT started
- [x] Global calibration accept blocked in API
- [x] HIGH/CRITICAL findings not auto-downgraded
- [x] Findings remain visible after reachability adjustment
- [x] Ruff style noise informational under homelab profile (prior sprint)
- [x] LLM sanity gate disabled by default

## Overfitting protections observed

- Recommendations require evidence threshold per repo
- Global scope recommendations rejected at accept API
- Learning events keyed with idempotency to prevent duplicate inflation

## Remaining follow-ups

1. Cursor Bugbot benchmark fixture run (comparison doc)
2. staticcheck CI confirmation
3. Live operator accept/reject of recommendations on homelab repos
