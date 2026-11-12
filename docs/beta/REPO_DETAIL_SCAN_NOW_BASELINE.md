# Repo detail Scan Now baseline

Generated: 2026-06-02

## Latest commit

`37e7111` — first tester feedback sprint (Scan Now on separate `/scan` route)

## Live route tested

`http://127.0.0.1:8081/ui/repos/31`

## Current behavior (pre-fix)

| Item | State |
|------|--------|
| Scan Now on repo detail | Link to `/ui/repos/:id/scan` (separate page) |
| Inline modal | Not present |
| Reconciliation panel | Below repo card, not grouped with scan controls |
| Live deployment | Pre-sprint image; favicon/reconciliation API may 404 |

## Expected behavior

| Item | Target |
|------|--------|
| Primary Scan Now | Prominent button on `/ui/repos/:id` opens inline modal |
| Report-only default | ON when issue filing disabled |
| Confirmation | Shows filing/PR/LLM/backlog status before start |
| After start | Scan ID, `manual` trigger, link to scan status (stay on page via AJAX) |
| Reconciliation | Compact summary adjacent to scan controls |
| Deep link | `/ui/repos/:id/scan` remains optional full-page form |

## Safety defaults (unchanged)

| Control | Default |
|---------|---------|
| Issue filing | Off |
| Report-only first scan | On |
| Remediation PRs | Off |
| LLM sanity gate | Off |
| All-repo scan | Not started |
| Backlog-control | Active (dogfood) |

## Product repo state

| Gate | Status |
|------|--------|
| Open issues | 1 (#48 operator task) |
| Active-present findings | 0 |
| Non-product issue filing | Disabled by default |

## Affected UI pages

- `/ui/repos/:id` — primary fix
- `/ui/repos/:id/scan` — optional deep link
- `ui/static/app.js` — modal + AJAX submit
- `ui/static/theme.css` — modal styles
