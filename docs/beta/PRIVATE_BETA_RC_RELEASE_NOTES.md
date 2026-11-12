# Private beta RC release notes

**Release commit:** `6d011cf`  
**Live revision:** `rc-e3e19ec`  
**Product:** Repository Detective — Inspect. Analyze. Improve.  
**Audience:** Invited operator beta testers (not public marketing)

## What this release is

Repository Detective is entering **controlled private beta**: invited operators can connect owned repos, run **report-only** scans, review findings, SBOM pages, repository maps, and pre-install audits. Product dogfood is clean (0 active-present). Core credibility checks passed on the RC homelab deployment.

This is an **invited operator beta**, not a production-ready public repo scanner announcement.

## What is ready

| Capability | Status |
|------------|--------|
| Multi-scanner repo analysis (Go, IaC, secrets, deps, health) | ready |
| Report-only / dry-run scans | ready |
| Findings detail (summary, risk, fix, calibration sections) | ready |
| SBOM UI + download (CycloneDX proof on homelab) | ready |
| Repository map / code graph | ready |
| Pre-install audit (public HTTPS repos) | ready, report-only |
| Learning / calibration review UI | ready |
| Gitea integration (connect, scan, webhook) | ready |
| UI route smoke (15/15) | ready |
| Container logs | clean |
| Screenshots / visual QA | 12 pages captured |
| Beta package (`make beta-release`) | ready |

## What is intentionally disabled (defaults)

| Control | Default | Notes |
|---------|---------|-------|
| Issue filing | off / report-only first | See issue provider matrix below |
| Remediation PR | **disabled** | Do not enable without operator approval |
| AI Recommendations | **disabled** | Provider-neutral; advisory only when enabled |
| Runner delegation | **disabled** | Container scans require explicit opt-in window |
| Container image scanning | **opt-in** | Not enabled by default |
| Gitea Actions backend | **disabled** | Not in beta scope |
| Auto disclosure submission | **disabled** | Never automatic |
| All-repo scanning | **not supported** | One repo per tester initially |
| LLM auditors / AI risk checks | off | Deterministic scanners are source of truth |

## Issue provider matrix

| Provider | Status |
|----------|--------|
| **Gitea** | Supported; live scratch-repo filing proof **pending** |
| **GitHub** | Implemented, **not release-proven** — do not claim in tester comms |
| **GitLab** | **Not implemented** |

## Known blockers (marketing, not private beta)

- Gitea wiki HTTP 500 (docs in repo/package; wiki remote unavailable)
- Live Gitea issue filing proof on owned scratch repo (planned, not run)
- Full external clean VM install proof (beta package proven; VM pending)
- GitHub live issue provider proof (optional)

## Tester limits

- **One repo per tester** initially
- **Report-only scan first** — verify 0 forge issues before any filing trial
- **Owned repos only** — no third-party or customer repos without written approval
- **No PHI/PII, secrets, or classified data** in beta repos
- **Small to medium** repos preferred (< ~2k files)
- **Gitea preferred** for first cohort
- Feedback via templates in `docs/beta/` — redact secrets

## Scanner and SBOM coverage (read before first scan)

| Message | Meaning |
|---------|---------|
| `binary_missing` | That scanner binary is not installed in this runtime — **the repo was not checked by that tool**. Not a clean result. |
| `sbom_tool_missing` | A dependency manifest exists but **Syft** is unavailable — no SBOM file was produced. |
| High finding count | Small homelab repos may trigger many **low/info** graph and debug heuristics; use severity filters and the scan page **grouped informational** summary. |

Install optional tools (trivy, grype, gitleaks, semgrep, syft) or use the full scanner image for broader coverage.

## Expected scan modes

1. **First scan:** `report_only_dry_run: true` or repo profile with issue filing off
2. **Review:** findings list, finding detail, SBOM page, graph, pre-install if applicable
3. **Optional second scan:** same repo after config review (still report-only unless operator approves filing)
4. **Never:** bulk all-repo scan, production container scan against live Docker hosts, auto-issue filing on third-party repos

## Safety defaults (explicit)

- Pre-install audits are **report-only** — no issues, no PRs
- Remediation PR is **disabled**
- AI Recommendations are **disabled by default**
- Runner delegation is **disabled by default**
- Container scanning is **opt-in**
- GitLab issue filing is **not implemented**
- GitHub issue filing is **implemented but not release-proven**
- Gitea issue filing is **supported**, but live scratch proof is **still pending**

## How to submit feedback

1. Copy [PRIVATE_BETA_FEEDBACK_TEMPLATE.md](PRIVATE_BETA_FEEDBACK_TEMPLATE.md) or the specialized bug/false-positive templates
2. Include scan ID, repo slug, commit SHA (`6d011cf` or newer)
3. Redact tokens, `.env`, and proprietary source
4. Send via operator-designated channel (Gitea issue on repository-detective, email, or chat — operator decides)

Categories: installation friction, UI confusion, finding quality, false positive, missed issue, scanner unavailable, SBOM/graph/pre-install quality, performance, docs gap, trust/safety concern.

## Operator references

- [PRIVATE_BETA_TEST_SCOPE.md](PRIVATE_BETA_TEST_SCOPE.md)
- [PRIVATE_BETA_OPERATOR_RUNBOOK_RC.md](PRIVATE_BETA_OPERATOR_RUNBOOK_RC.md)
- [CURRENT_READINESS_STATUS.md](../release/CURRENT_READINESS_STATUS.md)

## Upgrade path

Testers on older baselines (`76bd87d`, active-present 21) are stale. Current source of truth is `6d011cf` with dogfood clean at scan `926a5f56a26f03c9`.
