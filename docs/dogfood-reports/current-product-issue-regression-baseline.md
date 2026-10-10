# Product issue regression baseline

Recorded: 2026-06-08 after issue filing policy restore (`ddb79d6`).

## Repository state

| Metric | Value |
|--------|-------|
| Product repo | `commstech/Repository-Detective` (ID 1) |
| Latest scan ID | `8579510667b7de08` |
| Live container revision | `ddb79d6` |
| Gitea open issues | **6** (#347–351, #48) |
| Reconciliation `forge_open_issues` (DB) | 132 (stale mappings — many closed in Gitea) |
| Reconciliation `mapped_open_issues` | 132 |
| Reconciliation `unmapped_open_issues` | 127 |
| Active-present open findings | **1198** |
| Open findings total (DB) | 2289 |
| Findings without linked forge issue | 2258 |
| Findings with open forge issue | 31 |
| Issue filing enabled | yes |
| Dry-run on latest scan | no |
| Backlog-control | active (`dogfood_backlog_control_enabled: true`) |

## Open Gitea issues (snapshot)

| # | Title | Fingerprint | Path |
|---|-------|-------------|------|
| 351 | Possible command execution with dynamic input | rd-2f535117ad969767 | `sbom/sbom.go:118` |
| 350 | Possible hardcoded secret | rd-d6945f56a53d18dd | `benchmark/fixture/secret_hardcoded.go.src` |
| 349 | Possible hardcoded secret | rd-0be6c994528c6e24 | `benchmark/fixture/mock_secret_test.go.src` |
| 348 | Dynamic code execution | rd-68f285ebd87f8377 | `benchmark/fixture/dup_pattern_b.go.src` |
| 347 | Dynamic code execution | rd-47c97d7a4ecd8dd7 | `benchmark/fixture/dup_pattern_a.go.src` |
| 48 | Ops: homelab AI/Qdrant connectivity | (operator) | — |

Note: #205 (Base64 entropy / config template) is **closed** — mapped fingerprint `rd-a7fb8b9ed08e7f8f` remains a reconciliation example of finding↔issue linkage.

## Pre-install audit

| Setting | Live homelab |
|---------|----------------|
| `preinstall_audit_enabled` | **false** in `config/config.yaml` (overrides code default `true`) |
| UI | Shows disabled banner |

## Suspected cause of regression

1. **Issue filing restored** — manual/scheduled scans can create forge issues when policy allows.
2. **Benchmark fixtures scanned** — `benchmark/fixture/*.go.src` intentionally contain TP patterns but must not drive product-repo issues.
3. **Static rule SEC-CMD-EXEC** matched `sbom.go` grype invocation (string concat on same line as `exec.CommandContext`).
4. **Stale `external_issues` rows** — DB counts 132 open mappings vs 6 actual Gitea open issues.

## Top finding categories (latest scan)

Most active-present findings are graph/quality signals gated below issue-filing thresholds — expected under backlog-control and `reporting.max_issues_per_scan`. The urgent product-facing problems are the **6 open Gitea issues**, four of which are benchmark fixture false positives.

## Policy context

- Normal connected repos: file/update when `auto_create_issues: true` and dry-run not selected
- Pre-install: always report-only
- Product repo: must not accumulate unresolved forge issues from fixtures or self-match rules
