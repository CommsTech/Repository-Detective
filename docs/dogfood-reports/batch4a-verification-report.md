# Batch 4a verification report

Generated: 2026-06-06

## Scans

| | Scan ID | Instances |
|--|---------|----------:|
| Before Batch 4a | `cd4cb8d70d357f26` | 1111 |
| After Batch 4a | `db2d7061eaac8eb0` | 1093 |

## Issue counts

| Metric | Before sprint | After closeout | After Batch 4a rescan |
|--------|-------------:|---------------:|----------------------:|
| Open Gitea issues | 138 | 56 | **57** |
| Real active backlog | 24 | 24 | **11** |
| Resolved absent (open) | 69 | 0 | 14 |

## Closures this sprint (from 138 baseline)

| Action | Count |
|--------|------:|
| Resolved-absent closed (script + reconcile) | 82 |
| Reconcile apply actions | 115 |

## Batch 4a

| Metric | Value |
|--------|------:|
| Issues targeted | 12 |
| Issues fixed | 12 |
| Fingerprints absent post-rescan | 12/12 |

## Post-rescan filing

| Metric | Value |
|--------|------:|
| New issues created | 0 |
| Duplicate issues created | 0 |
| Backlog-control active | Yes |

## Prior batches still verified

Batch 2, 3a, 3b, 3c, 3d: sample fingerprints absent in `db2d7061eaac8eb0`.

## Remaining

- **Open Gitea:** 57 (30 out-of-scope summary/rollup, 2 needs-human-review, 14 new resolved-absent to close on next pass)
- **Active code backlog:** 11

## Next recommended batch

Batch 4b: close 14 new resolved-absent from rescan; fix remaining 11 active (gosec G304/G203, CKV template review, archive path).
