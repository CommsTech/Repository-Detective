# Gitea issue target correctness regression

**Date:** 2026-06-11  
**Revision:** musl deploy with dogfood fixes

## Test matrix

| Test | Method | Result |
|------|--------|--------|
| Dry-run creates 0 issues | PCAP_Analyser + ansible_playbooks scans (`report_only_dry_run: true`) | **PASS** — 0 issues filed |
| Pre-install creates 0 | Policy: pre-install report-only | **PASS** (enforced by config) |
| Container scan default filing | DB: alpine scans completed, 0 issues | **PASS** |
| Fingerprint dedupe | Unit: `TestFindIssueByFingerprint`, `TestFindIssueByFingerprintPaginatesBeyondFirstPage` | **PASS** |
| Gitea forge create | Unit: `TestGitHubForgeCreateIssue` (GitHub mock); Gitea client tests in `issues/` | **PASS** |
| Wrong repo cross-file | No code path for cross-repo filing; `IssueCreationRequest` uses scan repo owner/name | **PASS** (by design) |
| Live filing in correct repo | Not re-run this pass (avoid unsolicited Gitea issues) | **deferred** |

## Evidence

```text
POST /api/v1/analyze report_only_dry_run:true → 0 forge issues
Reconciliation: forge_open_issues=0, mapped_open_issues=0
Product rescan d3d6c4f279eeaf8c: 0 issues created
```

## Mapping storage

`external_issues` table links `finding_id` → forge type, issue number, URL. `FindIssueByFingerprint` searches open issues by fingerprint marker in body.

## Acceptance

| Item | Status |
|------|--------|
| Gitea supported | **yes** |
| Target correctness proven live | **partial** — dry-run + unit tests; live filing deferred |
| Duplicate prevention | **unit proven** |

## Next action

Controlled filing test on owned scratch repo with `ForceIssueCreation` when operator approves.
