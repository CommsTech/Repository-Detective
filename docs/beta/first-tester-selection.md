# First beta tester selection

**Date:** 2026-06-12  
**Release commit:** `e4892d9`  
**Cohort:** Invited operator beta #1

## Selected tester

| Field | Value |
|-------|-------|
| **Handle** | `operator-cohort-1` (internal homelab proxy) |
| **Role** | Trusted technical operator; homelab maintainer |
| **Beta agreement** | Report-only first scan; structured feedback with scan ID |
| **External name** | Pending — this cohort validates onboarding packet before external invite |

> **Note:** Scan executed as operator onboarding rehearsal on an owned non-product repo. Replace handle with external tester name when formal invite is sent.

## Repo candidate

| Field | Value |
|-------|-------|
| **Repository** | `commstech/PCAP_Analyser` |
| **Repo ID** | 13 |
| **Forge** | Gitea (`git.commsnet.org`) |
| **Primary language** | Python (small tooling repo) |
| **Expected size** | Small (~8 files analyzed in prior scans) |

## Sensitivity review

| Check | Result |
|-------|--------|
| Owned by tester/org | **yes** (commstech) |
| PHI/PII | **none expected** — network capture analysis tooling |
| Customer secrets | **none** |
| Production credentials in tree | **not expected** — operator spot-check if promoting to external tester |
| Non-sensitive homelab tooling | **yes** |

## Approved scan mode

```text
report_only_dry_run: true
issue_policy: off
analysis_depth: 2
enable_code_graph: true
```

## Features enabled

- Multi-scanner repo analysis
- Findings list and detail
- Repository map / graph
- SBOM page (honest empty state if no manifest)
- Executive report
- Learning/calibration review (read-only)

## Features disabled

| Feature | State |
|---------|-------|
| Issue filing | **off** |
| Remediation PR | **off** |
| AI Recommendations | **off** |
| Runner delegation | **off** |
| Container scanning | **off** |
| All-repo scan | **not allowed** |
| Third-party disclosure | **off** |

## Feedback deadline

**7 days** from invite send date (operator sets when external tester is named).

## Approval

| Role | Status |
|------|--------|
| Operator | approved for cohort-1 rehearsal |
| External tester invite | pending |
