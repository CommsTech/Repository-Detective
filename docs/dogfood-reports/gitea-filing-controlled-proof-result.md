# Gitea filing controlled proof — result

**Date:** 2026-06-12  
**Status:** **pass** (owned scratch repo only)  
**Live revision:** `rc-e3e19ec` (not redeployed during proof)  
**Code commit:** `311e97c` (structured issue body on next deploy)

## Scratch repo

| Field | Value |
|-------|-------|
| Repository | `commstech/rd-filing-scratch` (private) |
| RD repo ID | 221 |
| Filing enabled | per-repo only (`policy_level=issue_only`, `issue_policy=all`, `severity_gate=low`) |
| Product repo `commstech/Repository-Detective` | unchanged — 0 forge issues |
| Beta repo `commstech/PCAP_Analyser` | unchanged — 0 forge issues |

## Fixture

Controlled placeholder secret in `fixture.go` (high severity, obvious test value):

```go
const api_key = "rd-filing-proof-001"
```

**Note:** First filing attempts with medium `REL-INTERNAL-INFRA-REF` were **blocked by backlog-control** (allows only high/critical ≥0.85 confidence). High fixture required — documented for operators.

## Scan timeline

| Step | Scan ID | Report-only | Issues created | Issues updated |
|------|---------|-------------|----------------|----------------|
| 1 Register repo | `6fd5f7aa03ca44de` | yes | 0 | 0 |
| 2 Filing attempt (medium) | `ef4aa3b925843356` | no | 0 | 0 (backlog-control blocked) |
| 3 **Filing proof** | `381d556f3daadb45` | no | **1** | 0 |
| 4 Duplicate check | `e4d39913be7b1d55` | no | 0 | **1** |
| 5 Dry-run | `7e19138686d6fd4e` | yes | 0 | 0 |

## Issue created

| Field | Value |
|-------|-------|
| Issue number | **#1** |
| URL | https://git.commsnet.org/commstech/rd-filing-scratch/issues/1 |
| Owner/repo | `commstech/rd-filing-scratch` ✓ (not Repository-Detective, not PCAP_Analyser) |
| Title | `[HIGH] Possible hardcoded secret` |
| Fingerprint | `rd-b5f47d413661a577` |
| Rule | `SEC-HARDCODED-SECRET` |

Issue body includes fingerprint marker and structured sections (live deploy uses pre-`311e97c` body headings such as `## Finding Type`; filing mechanics verified).

## Duplicate prevention

Second non-dry-run scan (`e4d39913be7b1d55`):

- Docker log: `Updated existing issue #1 for fingerprint rd-b5f47d413661a577`
- Open issues count remained **1**
- Comment added on existing issue (semantic update path)

## Negative cases

| Case | Result |
|------|--------|
| Dry-run on scratch repo | **0 new issues** (`7e19138686d6fd4e`, `report_only_dry_run: true`) |
| Pre-install audit | **0 issues** (policy: always report-only; not re-run) |
| Container scan default | **0 issues** (opt-in disabled; unchanged) |
| Beta tester repo filing | **not enabled** — PCAP_Analyser `forge_open_issues=0` |

## Cleanup / rollback

| Action | Status |
|--------|--------|
| Issue #1 closed | **yes** — state `closed`, title prefixed `[CLOSED] Filing proof` |
| Filing disabled on scratch repo | **yes** — `policy_level=monitor_only`, `issue_policy=off` |
| Scratch repo | retained for operator audit (private) |

## Conclusion

Live Gitea filing creates issues in the **correct owner/repo**, respects **report-only dry-run**, updates by **fingerprint** without duplicates, and does not file on beta/product repos during this test.

## Related

- [gitea-filing-controlled-test-plan.md](gitea-filing-controlled-test-plan.md)
- [gitea-issue-target-regression-report.md](gitea-issue-target-regression-report.md)
