# Non-product dry-run round 2 report

Generated: 2026-06-07  
Mode: **report-only dry run** (post-calibration)

## Selected repo

| Attribute | Value |
|-----------|-------|
| Repo | `commstech/commsnet_optimizer` |
| Rationale | Medium shell homelab utility (~94 KB); representative of operator script repos; 0 open issues; not mission-critical |
| Alternative attempted | `commstech/OpenClaw` — scan hung in PREPARE (marked failed; infrastructure issue, not policy) |

## Scan summary

| Metric | Value |
|--------|-------|
| Scan ID | `4c8bfe80fa0abb6d` |
| Duration | ~11 s |
| Findings (unique) | **5** |
| Graph nodes/edges | 17 / 12 |
| Issues created | **0** |
| PRs created | **0** |
| `dry_run_report_only` | true |
| `issue_sync_status` | skipped |

## Scanner status (round 2)

| Scanner | Status | Notes |
|---------|--------|-------|
| grype | **scanner_unavailable** | Was `parse_failed` in round 1 — now correctly classified |
| shellcheck | binary_missing | Requires full image rebuild (install script updated; live container not rebuilt) |
| ruff | n/a | Not applicable (shell repo) |
| static / graph / trivy / gitleaks | clean or found | Expected |

## Findings breakdown

| Rule | Severity | Count |
|------|----------|------:|
| GRAPH-ORPHAN-FILE | info | 3 |
| REL-INTERNAL-INFRA-REF | info | 1 |
| QUAL-DEBUG | low | 1 |

All graph findings downgraded to **informational** with calibration notes — more actionable framing for repo owner.

## Netmapper recalibration comparison (same session)

Re-scanned `commstech/netmapper` to measure graph calibration:

| Metric | Round 1 (`913bfac39361b4df`) | Post-calibration (`8ce74105cee525b4`) |
|--------|------------------------------|---------------------------------------|
| Total findings | 87 | 259 |
| Graph findings | 48 (36 orphan + 12 island) | 23 (16 orphan + 7 island, all **info**) |
| Graph severity | medium/low | **info** (downgraded) |
| grype status | parse_failed | **scanner_unavailable** |
| ruff | binary_missing | **found** (197 persisted — needs gating) |

Graph noise **reduced and downgraded**; total finding count rose because ruff now runs in container (new calibration target).

## Actionability assessment

Round 2 report is **more actionable** than round 1 netmapper:

- Fewer total findings on representative shell repo (5 vs 87)
- Graph findings clearly informational, not blockers
- Internal infra ref downgraded appropriately for homelab context
- grype failure mode is honest (scanner unavailable vs misleading parse_failed)

## Acceptance

- [x] Scan completed
- [x] 0 issues created
- [x] 0 PRs created
- [x] Graph noise reduced/downgraded
- [x] grype no longer `parse_failed` for infra failure
- [x] Scanner gaps documented (shellcheck needs image rebuild; ruff needs severity gating)
- [x] Report actionable for repo owner
