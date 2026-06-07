# Batch 4b verification report

Generated: 2026-06-07 01:45 UTC

## Scans

| | Scan ID | Instances | Notes |
|--|---------|----------:|-------|
| Before (baseline) | `db2d7061eaac8eb0` | 1093 | 11 active, 14 resolved-absent open |
| After batch 4b deploy | `14f53450f4ad2b0a` | 1057 | All 11 batch fingerprints absent; 1 new high SEC-SQL-CONCAT (#346) from G201 refactor |
| After SQL FP fix | `cb2387b18150a561` | 1060 | #346 still present (multi-line query line) |
| Final | `68cab1ba3dc0591d` | 1088 | 0 active-present; persistence complete |

## Issue counts

| Metric | Before | After |
|--------|-------:|------:|
| Gitea open | 57 | **43** |
| Real active (present in latest scan) | 11 | **0** |
| Resolved-absent closed (batch) | — | 14 |
| Post-rescan evidence closes | — | #258, #346 |

## Batch 4b active findings

| Target | Result |
|--------|--------|
| Fingerprints targeted | 11 |
| Fixed in code | 11 |
| Absent in final scan | 11/11 |
| Skipped | 0 |

## Backlog control

- Mode: **active**
- New low/medium issues during final rescan: **0**
- New high issue during interim rescan: **1** (#346 SEC-SQL-CONCAT FP — closed with evidence after static fix)

## Duplicates

- Duplicate issues created during sprint: **0**

## Infrastructure

| Check | Status |
|-------|--------|
| Docker `core` build | **pass** (local `docker build --target core`) |
| Docker full verify (`all-in-one`) | partial — core/runner verified; full script slow on scanner-tools stage |
| CI run #119 (`73c4a0f`) | in progress at report time |
| Tests (`go test ./...`) | pass |
| Staticcheck | pass |
| Gosec | not run locally (scanner timeout in archive scans) |
| Secrets committed | **no** |

## Prior batches

Batch 2, 3a, 3b, 3c, 3d, 4a — remain verified.

## Remaining blockers

- 43 open Gitea issues (30 out-of-scope summaries, 2 needs-human-review, remainder non-active/mapped)
- CI run #119 must complete green
- Full `docker-build-verify.sh` all-in-one smoke on homelab (optional hardening)

## Sprint outcome

**Successful:** open count 57 → 43; real active 11 → 0; Docker core rebuild deterministic; batch 4b code fixes verified absent in final scan.

## Next recommended batch

Batch 5: triage remaining 43 open (summary/ops/human-review buckets) before all-repo dry-run planning.
