# Private beta report-only validation

Date: 2026-06-07  
Mode: `report_only_dry_run: true` via `scripts/run-non-product-dry-run.py`

## Summary

| Repo | Scan ID | Findings | issue_sync | Issues Δ | PRs |
|------|---------|----------|------------|----------|-----|
| commstech/Repository-Detective | `1c4db8a1a7ed8d1e` | 1146 | skipped | 0 | 0 |
| commstech/netmapper | `c10f8f9829e940f4` | 257 | skipped | 0 | 0 |
| commstech/commsnet_optimizer | `d1cf22c5890a75d1` | 5 | skipped | 0 | 0 |

**Total issues created: 0**  
**Total PRs created: 0**

## Per-repo notes

### commstech/Repository-Detective (product)

- `dry_run_report_only: true` confirmed in scan pipeline
- Open Gitea issues: 1 before/after (#48 operator task)
- Active-present (open issue fingerprint in latest scan): **0**
- Scanner notes: staticcheck/hadolint/checkov/golangci-lint/ruff timed out in container; grype unavailable

### commstech/netmapper

- Ruff: 487 findings (homelab profile gating applies at report layer)
- Graph + static scanners ran
- Open issues unchanged (1 before/after)

### commstech/commsnet_optimizer

- Small repo: 5 findings, graph present
- Open issues: 0 before/after

## Learning & calibration

| Check | Status |
|-------|--------|
| Learning events in DB | 16 (prior operator review seed) |
| Active repo calibration rules | 3 (netmapper ×2, commsnet ×1) |
| Global calibration | None |
| LLM sanity gate | Disabled |

Note: Live container runs pre-learning image; new learning events on dry-run require image rebuild after `6a2cbfd+`.

## Safety

- [x] Issue filing disabled
- [x] No non-product issues created
- [x] High/critical findings not suppressed
- [x] Product active-present remains 0
