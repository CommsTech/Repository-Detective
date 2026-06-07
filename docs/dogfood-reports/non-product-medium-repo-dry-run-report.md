# Medium repo dry-run report — `commstech/netmapper`

Generated: 2026-06-07  
Mode: **report-only dry run**

## Scan summary

| Metric | Value |
|--------|-------|
| Scan ID | `913bfac39361b4df` |
| Duration | ~16 s (16319 ms analysis) |
| Started | 2026-06-07T12:35:19Z |
| Finished | 2026-06-07T12:35:35Z |
| Files analyzed | 65 |
| Candidates → unique | 113 → **87** |
| Instances persisted | 87 |
| Persistence status | `complete` |
| Issue sync status | `skipped` |
| `dry_run_report_only` | **true** |
| Gitea issues before | 1 |
| Gitea issues after | 1 |
| Issues created | **0** |
| PRs created | **0** |

## Effective policy (scan snapshot)

| Setting | Value |
|---------|-------|
| `issue_policy` | `off` |
| `policy_level` | `monitor_only` |
| `ai_policy` | `disabled` |
| `remediation_policy` | `off` |
| `scan_profile` | `standard_deterministic` |

## Findings by severity

| Severity | Count |
|----------|------:|
| critical | 1 |
| medium | 29 |
| low | 57 |
| **Total** | **87** |

## Top rules (actionable vs noise)

| Rule | Source | Count | Notes |
|------|--------|------:|-------|
| GRAPH-ORPHAN-FILE | graph | 36 | Likely noisy for small Python utilities |
| QUAL-DEBUG | static | 18 | Debug print/logging — review for prod |
| GRAPH-SUSPICIOUS-ISLAND | graph | 12 | Graph heuristic noise candidate |
| REL-INTERNAL-INFRA-REF | static | 12 | Internal host/IP references |
| HEALTH-PY-NO-TEST | test_gap | 5 | Actionable for maintainability |
| OPT-NESTED-LOOP | static | 3 | Performance hint |
| SEC-EVAL | static | 1 | **Critical** — `eval()` usage; highest priority |

## Scanner status

| Scanner | Status | Findings |
|---------|--------|----------|
| static | found | 0 (pipeline stage) |
| health | found | 0 |
| graph | found | 0 |
| trivy | clean | 0 |
| grype | **parse_failed** | 0 |
| gitleaks | clean | 0 |
| semgrep | clean | 0 |
| govulncheck | clean | 0 |
| gosec | clean | 0 |
| staticcheck | clean | 0 |
| hadolint | clean | 0 |
| checkov | clean | 0 |
| ruff | **binary_missing** | 0 |

## Graph

| Metric | Value |
|--------|-------|
| Nodes | 164 |
| Edges | 258 |
| Build time | within scan budget (~16 s total) |

## Performance

- 65 files / ~249 KB workspace — completed in **16 s** (acceptable for medium homelab repo).
- Dedup reduced 113 candidates → 87 unique (26% reduction).
- No OOM or timeout.

## Observations

- **Report-only worked under load:** 87 findings persisted, zero forge activity.
- **Pre-existing external issue mapping:** 1 linked row from prior scan history — not created this run.
- **Scanner gaps:** grype parse failure (container/env); ruff not installed — Python lint coverage incomplete.
- **Noise:** graph orphan/island rules dominate (48/87 = 55%); filing without calibration would flood repo.

## Acceptance

- [x] Scan completed
- [x] Findings persisted (87)
- [x] Issue creation count = 0
- [x] No PRs created
- [x] No duplicate issue activity
- [x] Report generated
