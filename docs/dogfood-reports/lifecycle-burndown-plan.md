# Lifecycle burndown plan — commstech/Repository-Detective

**Generated:** 2026-06-06  
**Latest scan:** `852f2fb850b2b56d` (1080/1080 persisted, reconcilable)  
**Open Gitea issues:** 275

## Buckets (post backfill + closure)

| Classification | Count | Planned action |
|----------------|------:|----------------|
| active_present_in_latest_scan | 48 | Code-fix batch 3 (bounded) |
| resolved_absent_from_latest_scan | 129 | Evidence closure (done via API) |
| duplicate_existing_fingerprint | 68 | Reconcile label/link (applied) |
| out_of_scope_for_current_batch | 28 | Summary/ops — defer |
| needs_human_review | 2 | Manual triage (#48, #49) |

## Safe evidence rules

| Action | Requires |
|--------|----------|
| evidence_closure | Latest reconcilable scan; fingerprint absent; responsible scanner ran |
| duplicate_link | Canonical issue identified; no delete |
| mapping_backfill | Valid RD fingerprint marker; no new Gitea issue |
| code_fix | Active in latest scan; test + rescan after push |

## Priority order

1. Push `2ff94d6` + pending backfill classify fixes
2. Evidence closure verified (95 API calls — complete)
3. Duplicate reconcile (200 items applied ×2 passes)
4. SQL backfill 39 duplicate issue-number mappings
5. Real active backlog code fixes (48 findings)
6. Rescan after push (Phase 8)
