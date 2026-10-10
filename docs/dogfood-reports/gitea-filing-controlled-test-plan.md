# Controlled Gitea filing proof — test plan

**Status:** `not_run`  
**Requires:** Operator approval before execution  
**Release context:** `6d011cf` / live `rc-e3e19ec`

## Purpose

Prove Repository Detective creates and updates Gitea issues in the **correct owner/repo**, with no duplicates, without touching third-party repositories.

## Preconditions

- [ ] Operator approves filing test window
- [ ] Owned **scratch Gitea repo** created (e.g. `commstech/rd-filing-scratch`)
- [ ] Scratch repo contains minimal fixture triggering one **low-severity** deterministic finding (or use existing known rule)
- [ ] Product dogfood remains clean (separate from scratch repo)
- [ ] `auto_create_issues` enabled **only for scratch repo** or single controlled scan
- [ ] Reporting policy allows filing for test severity (low/medium)

## Test steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Connect scratch repo in RD | Repo appears in UI |
| 2 | Enable issue filing for scratch repo only | Global product repos remain report-only |
| 3 | Run scan (no dry-run) | Scan completes |
| 4 | Check Gitea `commstech/rd-filing-scratch` issues | **1 issue** in scratch repo, correct title/body fingerprint |
| 5 | Check `external_issues` table | provider=gitea, correct owner/repo/number |
| 6 | Run **second scan** same ref | **Same issue updated**, not duplicated |
| 7 | Verify reconciliation | `FindingsWithIssue` matches; no duplicate forge rows |
| 8 | Fix or remove fixture finding | Optional: verify comment/closure if evidence closure enabled |
| 9 | Disable issue filing on scratch repo | Filing off |
| 10 | Archive/delete scratch repo | Operator choice |

## Negative cases (already proven elsewhere)

| Case | Evidence |
|------|----------|
| Dry-run creates 0 | PCAP_Analyser, ansible_playbooks scans |
| Pre-install creates 0 | Policy enforced |
| Container scan default 0 | alpine demo |
| Wrong repo cross-file | Code review — `IssueCreationRequest` uses scan repo |

## Artifacts to record

- Scratch repo name
- Scan IDs (first and second)
- Gitea issue number and URL
- `external_issues` row snapshot (redacted)
- Before/after reconciliation summary

## Rollback

1. Close/delete scratch Gitea issue
2. Disable filing on scratch repo
3. Remove scratch repo connection from RD
4. Confirm product repo `commstech/Repository-Detective` still report-only / 0 forge issues

## Default

**Do not run** during private beta expansion unless operator explicitly schedules. Update this doc with `status: pass` and evidence paths when complete.

## Related

- [gitea-issue-target-regression-report.md](gitea-issue-target-regression-report.md)
- [ISSUE_FILING_POLICY.md](../beta/ISSUE_FILING_POLICY.md)
