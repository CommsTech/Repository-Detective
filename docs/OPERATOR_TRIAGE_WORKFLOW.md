# Operator workflow — from CVE alert to verified resolution

**Audience:** solo maintainers, small teams, and homelab operators running a self-hosted Gitea forge. This is an operator guide, not a promise that every CVE can be patched automatically. Forgejo remains a best-effort target pending independent acceptance.

## Before you triage

1. Sign in at `/ui/login`. Check `/health/ready` and the **System Health** page: a partially failed scan is not evidence that a repository is clean.
2. Confirm each repository's scanner profile, credentials, network access and latest scan date. Run a report-only scan first. Keep Gitea auto-issue filing off until evidence quality is trustworthy.
3. For private-only installations, leave external CVE intelligence disabled. Optional local Ollama assistance can be enabled separately; review the privacy boundary before sending source-derived evidence to any model.

## Five-minute daily routine

| Task | Location | What counts as completion |
|---|---|---|
| Identify the most urgent findings | Dashboard → **Triage critical/high** | Open findings, critical first, then high; severity is not proof of exploitability |
| Verify that each affected package is present | Finding → **Inspect scanner evidence** | Affected version, file/image and scanner run are present and credible |
| Understand current exploitation signals, if permitted | Finding → **Check live signals** | KEV / EPSS returned or explicitly marked unavailable; no assumptions from missing data |
| Plan a safe change | Finding → **Review remediation plan** | Reviewed fixed version, dependency impact, test plan and rollback path |
| Implement and verify | Approved team PR → deploy → scan again → closure evidence | Required scanners completed; vulnerable package/finding absent with traceable evidence |

**Important:** AI recommendations are suggestions. Repository Detective does **not** currently upgrade dependency lockfiles, rotate credentials, merge PRs or prove runtime reachability. A remediation-plan approval does not execute the fix. Existing deterministic patchers can open PRs only for a narrower set of supported low-risk changes, when separately enabled and validated.

## Work the priority queue

Open `/ui/findings?focus=1`. The **Priority** tab keeps the queue open-critical/high, **All findings** searches all non-suppressed records by default, and **Resolved** shows historical resolved findings. Focus means **open critical and high**, independent of recent low-severity noise. The database filters and orders before page limits. The UI displays total matching findings, 50 per focus page or 100 per general page; next/previous links preserve the active filters. Use the top search field for **literal substring** matching against CVE/rule identifier, package name, finding title, file path, fingerprint, or repository name (case-insensitive; maximum 120 characters). The search queries SQLite, not just the visible page; the same query is preserved across pagination and CSV/JSON exports. Search stays inside the active view — switch to **All findings** to search beyond critical/high.

Expand **Advanced filters** only when needed. On All findings, you can filter status, repository, scanner, severity or category. Explicit `suppressed` or `false_positive` status selections now include those records without requiring the separate “Include suppressed” checkbox; the “Only suppressed” checkbox overrides conflicting open-status selection. The UI uses native HTML forms and works without JavaScript.

**Finding page order:** the investigation brief appears first, followed by the remediation plan and closure evidence where enabled. Raw evidence is one expandable section below these actions. You can jump directly between steps with in-page links. Generating a new plan is a deliberate action and does not implement the fix.

**Do not use suppression as an alternative to risk acceptance.** Mark something false positive only when evidence is actually incorrect. When risk is accepted, document the exception and approval in the system of record; don't mislabel a real vulnerability.

Use **Export CSV** for spreadsheet reporting (cells beginning with formula operators are text-protected) or **Export JSON** for the original structured finding text. Exports contain the entire matching set **up to 20,000 findings**, not just the current page or 200 newest entries. Requests above 20,000 are rejected with an explicit explanation; narrow the filters for large estates. Exports and page contents can change if new scans are writing at the same time; for formal evidence use a stable scan window.

## Decision aid: who does what?

| Evidence / action | Repository Detective | Operator / team |
|---|---|---|
| Deterministic scanner finding | Collect, normalize and deduplicate | Confirm scanner health and affected asset |
| CISA KEV + FIRST EPSS | On-demand, opt-in lookup of public CVE ID | Interpret against actual deployment exposure |
| Remediation strategy | Deterministic plan; optional AI advice | Verify vendor fixes and choose a change |
| Dependency version upgrade | **Not automatic** | Edit manifest/lockfile, build/test, review PR |
| Supported low-risk PR recipe | Guarded and disabled by default | Review and approve before merge |
| Closure | Requires scan and evidence checks | Confirm the deployed artifact, coverage and issue state |

## Common symptoms

- **No findings**: inspect scanner coverage and latest scan status before assuming there are no vulnerabilities.
- **No Gitea issue**: findings can exist without a forge issue because filing may be disabled or below severity/confidence gates.
- **AI not available**: scanner findings and deterministic plans remain usable. Verify provider, model, container-to-model network route and separate remediation AI setting.
- **No PR button**: remediation PR execution is disabled by default, or the specific plan is ineligible. A dependency CVE is not a supported auto-patch recipe.
- **KEV/EPSS unavailable**: enrichment is opt-in and external networks can fail; unknown does **not** mean safe.
- **Finding persists after merge**: confirm the fixed version made it into the built/deployed artifact and run a new complete scan. Do not close merely because code was merged.

See [CVE intelligence and AI remediation](CVE_TRIAGE_AND_AI_REMEDIATION.md), [remediation safety](REMEDIATION.md), [scanner coverage](SCANNERS.md), and [evidence-based closure](EVIDENCE_CLOSURE.md).
