# First tester feedback sprint baseline

Generated: 2026-06-02

## Latest commit

`163ef33` — first tester release verification (pre-sprint)

## Product repo state

| Gate | Status |
|------|--------|
| Open issues (Gitea) | 1 (#48 operator task) |
| Active-present findings | **0** |
| Non-product issue filing | Disabled by default |
| All-repo scan | Not started |
| LLM sanity gate | Disabled by default |
| Report-only dry-run | Required for first tester scans |

## Tester feedback items

1. **Manual scan** — no clear way to trigger scans from UI (scheduled only)
2. **Repo settings UX** — policies hard to navigate; setting purpose unclear
3. **Favicon** — browser tab icon wrong (does not match product logo)
4. **Issue/finding counts** — repo open issues vs scan findings mismatch confusing
5. **Reconciliation clarity** — must distinguish scan findings, forge issues, report-only, resolved-verified, duplicates

## Validation case (AMMBER queue)

- Many findings with no forge issue (expected in report-only / backlog-control)
- `SEC-HARDCODED-SECRET dashboard/static/help.html:38` maps to forge issue `#205`
- UI must explain why finding count > mapped issue count without false mismatch warnings

## Safety defaults (unchanged)

| Control | Default |
|---------|---------|
| Issue filing | Off |
| Report-only first scan | On |
| Remediation PRs | Off |
| LLM sanity gate | Off |
| Backlog control | On |

## Affected UI pages

| Page | Changes planned |
|------|-----------------|
| `/ui/repos` | Scan Now action |
| `/ui/repos/:id` | Scan Now + reconciliation panel |
| `/ui/repos/:id/settings` | Sectioned settings with explanations |
| `/ui/scans` | Scan Now when repo context available |
| `/ui/scans/:id` | Reconciliation panel scoped to scan |
| Layout / favicon | Correct icon at `/favicon.ico` and `/ui/static/favicon.svg` |

## Git hygiene

| Check | Status |
|-------|--------|
| `.env` staged | No |
| `dist/` staged | No |
| Local `repository-detective` ELF staged | No |
