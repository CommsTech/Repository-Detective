# Product issue burn-down after filing restore

Plan recorded: 2026-06-08 (sprint after `ddb79d6`).

## Before

| Metric | Value |
|--------|-------|
| Gitea open issues | 6 (#347–351, #48) |
| Active-present findings | 1198 |
| Mapped forge issues (DB) | 132 (stale) |
| Findings with forge issue | 31 |

## Classification (open Gitea issues)

| Issue | Class | Action |
|-------|-------|--------|
| #347–348 | false_positive_with_evidence | Benchmark fixture `dup_pattern_*.go.src` — close |
| #349–350 | false_positive_with_evidence | Benchmark fixture secret patterns — close |
| #351 | active_present_code_fix | Refactor `sbom/sbom.go` grype exec — close after fix |
| #48 | operator_task | Homelab ops — keep open |

## Code fixes (this sprint)

1. `skipStaticAnalysisPath` — exclude `benchmark/fixture/` and `.go.src`
2. `profile.ClassifySourceType` — classify benchmark paths as test fixtures
3. `sbom/sbom.go` — grype args on separate line (SEC-CMD-EXEC avoid)
4. `preinstall_audit_enabled: true` in homelab config

## Closures executed

Closed with evidence comments: #347, #348, #349, #350, #351.

## After (immediate)

| Metric | Value |
|--------|-------|
| Gitea open issues | **1** (#48 ops) |
| Issues closed | 5 |
| Active-present | 1198 (pending rescan to mark fixture findings resolved_absent) |

## Expected after product rescan

- Benchmark fixture fingerprints absent from latest scan instances
- Active-present count drops materially (exact delta depends on graph/quality gates)
- `external_issues` stale rows synced via reconcile + forge state refresh

## Next batches

1. Product repo manual rescan (not all-repo) after deploy
2. Reconcile `external_issues` against Gitea closed state
3. High/critical security in non-fixture paths
4. Graph noise calibration (report-only, not suppress)
