# Calibration operator review report

Date: 2026-06-02  
Mode: **report-only** — issue filing disabled  
Database: migrated to schema v20; 16 learning events from operator dry-run review

## Summary

| Metric | Value |
|--------|-------|
| Recommendations reviewed | 5 |
| Accepted (repo-scoped) | 3 |
| Rejected | 2 |
| Global accepted | 0 |
| Active repo rules (90-day expiry) | 3 |

## Decisions

| Repo | Rule | Action | Evidence | Decision | Reason | Expiry | Safety |
|------|------|--------|----------|----------|--------|--------|--------|
| commstech/netmapper | `graph/GRAPH-ORPHAN-FILE` | report_only | 3 FP events (100%) | accept_repo_scoped | Homelab graph orphan noise; maintainability only | 90d | No HIGH/CRITICAL hide |
| commstech/netmapper | `graph/GRAPH-ORPHAN-FUNCTION` | report_only | 3 FP events (100%) | accept_repo_scoped | Same pattern as ORPHAN-FILE | 90d | Findings remain visible |
| commstech/commsnet_optimizer | `graph/GRAPH-ORPHAN-FILE` | report_only | 3 FP events (100%) | accept_repo_scoped | Repo-scoped only; does not affect netmapper | 90d | No security downgrade |
| commstech/commsnet_optimizer | `graph/GRAPH-ORPHAN-FUNCTION` | report_only | 3 FP events | reject | Insufficient cross-scan evidence for second rule | — | Conservative |
| commstech/nextcloud_scripts | `graph/GRAPH-ORPHAN-FILE` | report_only | 3 FP events | reject | needs_more_evidence — not in prior dry-run scope | — | No change |

## Safety checks

- [x] No issue filing enabled
- [x] No high/critical security findings hidden
- [x] Global calibration accept blocked in API (beta policy)
- [x] Accepted rules expire in 90 days
- [x] Findings remain visible in reports (confidence/display only)

## Expected effect

- **netmapper** and **commsnet_optimizer**: graph orphan findings show calibration note; lower issue-filing eligibility for accepted rules only in that repo.
- **Repository-Detective product repo**: unaffected (no rules accepted for repo id 1).
- **nextcloud_scripts**: no calibration applied pending more dry-run evidence.
