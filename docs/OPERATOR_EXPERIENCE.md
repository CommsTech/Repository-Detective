# Repository Detective — operator experience and UI decision contract

**Status:** design and implementation guide for the unreleased findings/search workflow (2026-11-16). This is not a claim of production deployment, penetration testing, or formal accessibility certification.

## Audience and intended outcomes

| Operator | Main question | Primary path | Explicit boundary |
|---|---|---|---|
| Solo Gitea maintainer / homelab admin | What must I look at today? | Dashboard → Priority findings | Severity ≠ exploitability; partial scans ≠ clean |
| Developer on a small team | Which component do I change? | Search CVE/package/rule → finding detail → evidence | A scanner-suggested fixed version does not prove upgrade safety |
| Reviewer / team lead | Can we approve this proposal? | Remediation plan → validation commands → manual change / guarded PR | AI advice is never automatic approval or deployment |
| Operator verifying closure | Is this truly gone? | Rescan → closure evidence → history | A PR or merge by itself never closes a finding |
| New administrator | How do I begin without unsafe automation? | First-run checklist → repositories → Doctor → report-only scan | Issue filing and AI can remain disabled while establishing a baseline |

## UI flow

1. **Dashboard** communicates backlog, critical/high volume and scanner health. On an empty instance it shows a concrete three-step onboarding path instead of only blank charts.
2. **Findings → Priority** shows open critical/high first. The **All findings** tab exposes the wider inventory and **Resolved** is a history shortcut. Tabs intentionally reset other filters so their scope is unmistakable.
3. **Search** runs against the SQLite inventory, not the current HTML page. It matches title, CVE/rule ID, package, file path, fingerprint and repository. A term with `%` or `_` is treated literally; it is not an SQL wildcard or a glob. Results, total count, pagination and export use the same filter.
4. **Advanced filters** are collapsed until requested or until a filter is active. Focus mode restricts severity to critical/high and status to open. Suppressed/false-positive selections intentionally reveal those records, but they are excluded from ordinary All results by default.
5. **Finding detail** answers what was found, how much is known about runtime exposure, and the next safe action. Operators proceed from investigation to plan to scan-backed closure before exploring full forensic detail. If planner, PR engine, or closure verification is disabled, the interface describes that limitation, never a fake action.
6. **Mobile and assistive tech** use a toggled nav, text severity badges, labeled search and status, native form submission, keyboard-focus rings, and expandable evidence. No public SaaS login or external JavaScript CDN is needed for the new flow.

## Design principles

- **Operator task before decoration:** urgent queue and actions precede charts; charts remain supplementary evidence on the dashboard.
- **Honest states:** `not_checked`, `unavailable`, `not_verified`, `not_eligible`, and `disabled` are different from `safe`, `fixed`, and `complete`.
- **No surprise writes:** page views, navigation, filters, searching and exports do not make repository changes. Plan generation remains an explicit POST with CSRF protection. Regeneration asks for confirmation in supporting browsers.
- **Minimal external disclosure:** searching and sorting use SQLite locally. Optional KEV/EPSS lookups stay user-triggered and send a CVE ID rather than source code.
- **Strong defaults:** unverified dependency-upgrade changes require human ownership. Existing supported low-risk PR recipes have independent eligibility and remain disabled by default.
- **Avoid treating false positives as accepted risk:** use a risk register/issue for acceptance decisions, not scanner evidence reclassification.

## Staging acceptance — required before release

- [ ] Go 1.26.10: `go test ./store ./ui` and `go test ./...`; record commit hash and results.
- [ ] Search for exact CVE, package, filename, repository; test literal `%` and `_`; verify counts match list/export across 100+ findings.
- [ ] Verify focus mode always excludes medium/low/resolved and that older critical issues remain on page 1.
- [ ] Verify explicit suppressed status and only-suppressed checkbox display appropriate rows; empty results explain that scanner coverage still matters.
- [ ] Test first-run empty instance and populated instance; confirm no actions are advertised when disabled.
- [ ] Test viewport widths 320px, 768px and desktop, plus 200% and 400% zoom in light/dark/system themes.
- [ ] Keyboard-only: skip link, nav toggle, Escape/focus return, search, details, pagination, in-page anchors, CSRF-protected forms, and focus management.
- [ ] Complete an end-to-end disposable Gitea scan → triage → human-reviewed plan → changed branch → rescan → verified closure.
- [ ] Validate UI with live scanner failures, missing feeds, API failure, no AI, and no network access. Check accessible contrast and screen-reader announcements.

See [Daily triage](OPERATOR_TRIAGE_WORKFLOW.md), [Accessibility](ACCESSIBILITY.md), [CVE safety](CVE_TRIAGE_AND_AI_REMEDIATION.md), and the [verification receipt](release/VERIFICATION_OPERATOR_FLOW_2026-11-16.md).
