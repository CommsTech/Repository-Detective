# Issue provider RC verification

**Date:** 2026-06-10  
**Revision:** `rc-e3e19ec`

## Gitea

| Check | Status |
|-------|--------|
| Issue manager forge adapter | implemented |
| Create/update via product dogfood | historically proven |
| Repo mapping regression (this pass) | **not re-run** — manual RC test pending |
| Dry-run (`report_only_dry_run`) | **proven** — beta scans created 0 issues |
| Pre-install auto-file | disabled by policy |
| Container scan auto-file | disabled by default |

Beta scan evidence:

- `1a4fc7a409f6d376` — `dry_run_report_only: true`, issues filed: 0
- `64684cbab7682847` — `dry_run_report_only: true`, issues filed: 0

## GitHub

| Check | Status |
|-------|--------|
| Forge adapter in code | **yes** (`issues.Manager` + `GitHubForge`) |
| Unit test | `TestGitHubForgeCreateIssue` **PASS** |
| Live org integration | **not RC-proven** (401 on startup check with configured token) |
| UI/docs status | **implemented but not release-proven** |

## GitLab

| Check | Status |
|-------|--------|
| Implementation | **not_implemented** |
| UI/docs | must not imply support — **PASS** |

## Cross-repo filing

Policy documented: issue target must match source repo provider/owner/repo. No cross-repo filing test run this pass.

## Acceptance

| Provider | RC status |
|----------|-----------|
| Gitea | **supported** (partial — mapping regression pending) |
| GitHub | **implemented, not release-proven** |
| GitLab | **not_implemented** |
