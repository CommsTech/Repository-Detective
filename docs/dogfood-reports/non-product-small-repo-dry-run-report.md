# Small repo dry-run report — `commstech/nextcloud_scripts`

Generated: 2026-06-07  
Mode: **report-only dry run**

## Scan summary

| Metric | Value |
|--------|-------|
| Scan ID | `de67d8671c92d720` |
| Duration | ~10 s (9863 ms analysis) |
| Started | 2026-06-07T12:35:01Z |
| Finished | 2026-06-07T12:35:11Z |
| Files analyzed | 1 |
| Findings (unique) | **0** |
| Instances persisted | 0 |
| Persistence status | `complete` |
| Issue sync status | `skipped` |
| `dry_run_report_only` | **true** |
| Gitea issues before | 0 |
| Gitea issues after | 0 |
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

## Scanner status

| Scanner | Status | Findings |
|---------|--------|----------|
| static | clean | 0 |
| health | clean | 0 |
| graph | clean | 0 |
| trivy | clean | 0 |
| grype | **parse_failed** | 0 |
| gitleaks | clean | 0 |
| semgrep | clean | 0 |
| govulncheck | clean | 0 |
| gosec | clean | 0 |
| staticcheck | clean | 0 |
| hadolint | clean | 0 |
| checkov | clean | 0 |
| shellcheck | **binary_missing** | 0 |

## Graph

| Metric | Value |
|--------|-------|
| Nodes | 2 |
| Edges | 0 |

## Observations

- **Sanity pass:** single shell script repo scanned cleanly with no findings.
- **Scanner failures (non-blocking):** grype JSON parse error; shellcheck binary absent in container image.
- **False positives:** none (no findings).
- **Report-only enforcement:** log confirmed `Forge issue creation skipped (policy_level=monitor_only issue_policy=off)`.

## Acceptance

- [x] Scan completed
- [x] Findings persisted (0)
- [x] Issue creation count = 0
- [x] No PRs created
- [x] No duplicate issue activity
- [x] Report generated

**Note:** An earlier pre-fix scan (`ec62550c74881916`) ran before context propagation fix; superseded by `de67d8671c92d720`.
