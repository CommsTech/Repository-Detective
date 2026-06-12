# Private beta — first tester scan

**Date:** 2026-06-12  
**Tester cohort:** `operator-cohort-1` (internal rehearsal)  
**Release commit:** `e4892d9`

## Scan record

| Field | Value |
|-------|-------|
| Repository | `commstech/PCAP_Analyser` |
| Repo ID | 13 |
| Scan ID | `512145e55d4488ea` |
| Trigger | manual |
| Mode | `report_only_dry_run: true` |
| Profile | `standard_deterministic` |
| Analysis depth | 2 |
| Graph | enabled |
| Duration | ~4.4s |
| Status | completed |

## Results

| Metric | Value |
|--------|-------|
| Files analyzed | 8 |
| Findings (scan) | 12 |
| High/critical | **0** |
| Severity mix | info 9, medium 2, low 1 |
| Overall score | 0.9 |
| Issues created | **0** ✓ |
| PRs created | **0** ✓ |
| Issue sync | skipped (report-only) |

## Scanner coverage

Enabled: trivy, grype, gitleaks, semgrep, govulncheck, gosec, staticcheck, hadolint, checkov, linters  
AI policy: disabled  
Remediation policy: off  
Issue policy: off

## Graph state

| Field | Value |
|-------|-------|
| State | **available** |
| Nodes | 29 |
| Edges | 23 |

## SBOM state

| Field | Value |
|-------|-------|
| Status | `sbom_no_supported_manifest` |
| Detail | No supported dependency manifest detected |
| UI | Honest empty state — acceptable |

## Safety verification

| Check | Result |
|-------|--------|
| Forge open issues for repo | 0 |
| Secrets in logs | none observed |
| Dry-run flag in summary | true |
| Pre-install side effects | none |

## UI / report first impression (operator)

| Area | Assessment |
|------|------------|
| Findings list | Understandable for small repo |
| Finding detail | Actionable sections available |
| Graph | Available; useful for tiny codebase |
| SBOM | Correct empty-state messaging |
| Executive report | Useful for beta triage |
| Scan duration | Fast (~4s) — good first impression |

## Tester feedback

**Status:** pending external tester response  
Operator rehearsal: scan acceptable for cohort onboarding.

## Acceptance

| Criterion | Result |
|-----------|--------|
| Scan completes | **PASS** |
| 0 issues filed | **PASS** |
| 0 PRs created | **PASS** |
| Findings understandable | **PASS** |
| Report useful | **PASS** |
| No secrets leaked | **PASS** |
| Feedback requested | **pending** |

## Next steps

1. Send outreach to first external tester (if different from operator proxy)
2. Collect [PRIVATE_BETA_FEEDBACK_TEMPLATE.md](../beta/PRIVATE_BETA_FEEDBACK_TEMPLATE.md)
3. Triage top 2 medium findings for false-positive rate
