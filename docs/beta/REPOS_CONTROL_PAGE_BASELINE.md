# Repos control page baseline

Generated: 2026-06-02

## Latest commit

`ab7e79a` — repo 31 direct Scan Now verified

## Current `/ui/repos` behavior

| Item | State |
|------|--------|
| Page role | Repository inventory list |
| Columns | Name, forge, last scan, status, open findings, actions |
| Scan Now | Link to `/ui/repos/:id/scan` (separate page) |
| Enable/disable scanning | Not on list — requires settings page |
| Issue filing / report-only state | Not shown |
| Reconciliation metrics | Not shown |
| Scheduling state | Not shown |

## Missing controls

- Per-row enable/disable scanning
- Inline Scan Now modal from list
- Issue filing mode / report-only visibility
- Latest scan ID, issue sync status
- Active-present vs forge open issues
- Compact reconciliation counts (AMMBER-style mismatch)

## Expected behavior

`/ui/repos` becomes the **fleet control panel**:

- Enable/disable scanning per repo
- Scan Now per repo (inline modal, report-only default)
- Scheduling, issue filing, scan profile visible per row
- Latest scan status/time + reconciliation summary
- Links to detail, settings, report

`/ui/repos/:id` remains repo detail.

## Safety defaults (unchanged)

| Control | Default |
|---------|---------|
| Issue filing | Off |
| Report-only first scan | On |
| Remediation PRs | Off |
| LLM sanity gate | Off |
| All-repo scan | Not started |
| Backlog-control | Active |

## Product repo state

| Gate | Status |
|------|--------|
| Open issues | 1 (#48 operator task) |
| Active-present findings | 0 |
| `/ui/repos/31` Scan Now | Working (post-redeploy) |

## Affected files

- `ui/templates/repos.html`
- `ui/repos_control_model.go` (new)
- `ui/repos_control_handlers.go` (new)
- `store/repo_control_list.go` (new)
- `api/handler.go` — enable/disable scanning routes
- `ui/static/app.js` — list scan modal + toggle forms
