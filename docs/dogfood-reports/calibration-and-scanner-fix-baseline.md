# Calibration and scanner fix baseline

Generated: 2026-06-07  
Mission: improve dry-run accuracy before any limited issue filing

## Product repo status

| Metric | Value |
|--------|-------|
| Latest commit (start) | `d2437c3` |
| Open Gitea issues | **1** (#48 operator task) |
| Active-present findings | **0** |
| Report-only dry-run | enforced (`report_only_dry_run: true`) |
| Limited issue filing | **NOT approved** |
| Backlog-control | active |

## Dry-run round 1 results

| Repo | Scan ID | Findings | Issues created |
|------|---------|----------|----------------|
| `commstech/nextcloud_scripts` | `de67d8671c92d720` | 0 | 0 |
| `commstech/netmapper` | `913bfac39361b4df` | 87 | 0 |

## Graph noise (netmapper)

| Rule | Count | % of findings |
|------|------:|--------------:|
| GRAPH-ORPHAN-FILE | 36 | 41% |
| GRAPH-SUSPICIOUS-ISLAND | 12 | 14% |
| **Graph subtotal** | **48** | **~55%** |

## Scanner failures (round 1)

| Scanner | Status | Notes |
|---------|--------|-------|
| grype | parse_failed | Non-JSON vuln DB error on stdout |
| shellcheck | binary_missing | Not in container image |
| ruff | binary_missing | Not in container image |

## Gate decision (pre-calibration)

`ready_for_more_dry_runs` — safety controls proven, signal quality insufficient.

## Why limited issue filing remains blocked

1. Graph findings dominated netmapper report (~55% noise).
2. grype misclassified infrastructure failures as parse_failed.
3. Python/shell linter gaps (ruff, shellcheck) reduced confidence.
4. Homelab internal refs flagged at medium severity without context.
5. Operator has not approved limited issue filing.

## Calibration targets

- Graph: downgrade/skip low-signal orphans in small/homelab repos; detect Makefile/compose/README entrypoints.
- grype: classify DB errors vs no-manifest vs valid JSON.
- Docker: install shellcheck + ruff; refresh grype DB on image build.
- Profile: auto-detect homelab/infra repos; downgrade REL-INTERNAL-INFRA-REF when appropriate.
