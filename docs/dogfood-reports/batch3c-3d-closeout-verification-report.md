# Batch 3c/3d closeout verification report

Generated: 2026-06-06

## Scans

| | Scan ID | Finding instances |
|--|---------|------------------:|
| Before sprint | `a8bb4cddd72ab80c` | 1101 |
| After sprint | `cd4cb8d70d357f26` | 1111 |

## Gitea issue count

| Metric | Before | After | Delta |
|--------|-------:|------:|------:|
| Open Gitea issues | 294 | 138 | **-156** |
| Real active backlog | 51 | 24 | **-27** |
| Resolved absent (open) | 143 | 82 | -61 |
| Duplicates (open) | 68 | 0 | -68 |

## Closures (evidence-backed)

| Action | Count |
|--------|------:|
| Verified-resolved closed (closeout script) | 77 |
| Duplicate closed (closeout script) | 67 |
| Additional reconcile closures (post-deploy) | 12 |
| Manual/no-evidence closures | 0 |

## Code fixes

| Batch | Issues targeted | Fingerprints absent post-rescan |
|-------|----------------:|--------------------------------|
| Batch 3c (gosec medium) | 15 (#301–#313, #318–#319) | 14/15 on first rescan; #312 fixed in `26eae14` |
| Batch 3d (HEALTH-IGNORED-ERROR) | 11 (#329, #334–#343) | All absent |

## Post-rescan issue filing

- **New issues created during rescan:** 0 (`issues_created: 0` in scan summary)
- **Duplicate burst:** none observed
- **Backlog-control mode:** active — low/medium filing paused; findings still persisted (1111 instances)

## Batch verification carry-forward

| Batch | Status |
|-------|--------|
| Batch 2 (store reliability) | Still verified |
| Batch 3a (#316, #323) | Still verified |
| Batch 3b (#263–#268, #276, #292) | Still verified |

## Remaining

- **Open Gitea issues:** 138 (82 resolved-absent candidates for next reconcile pass)
- **Real active backlog:** 24 code-fix candidates
- **CI:** Run #1864 in progress on `a4a565c`; prior runs failed on unrelated checks

## Sprint outcome

**Successful:** visible Gitea open count decreased from ~294 to **138** with evidence-backed closures, duplicate removal, backlog-control filing pause, and active code fixes.
