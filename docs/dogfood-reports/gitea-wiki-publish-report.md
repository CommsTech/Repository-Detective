# Gitea wiki publish report

**Date:** 2026-06-02

## Source

- `docs/wiki/` (23 markdown pages)
- Publish script: `scripts/publish-gitea-wiki.sh`

## Pages prepared

Home, Quick Start, Private Beta Install, Configuration, Report-Only Scans, Manual Scan Now, Repo Settings and Policies, Pre-install Audit, Scanner Coverage, Secret Scanning and Git History, SBOM, Learning and Calibration, Issue/Finding Reconciliation, Troubleshooting, Operator Runbook, FAQ, plus existing operator pages (Dashboard, Privacy, Scanner Health, etc.).

## Wiki remote

`https://git.commsnet.org/commstech/Bugbot.wiki.git`

## Publish result

| Step | Result |
|------|--------|
| Dry-run | **success** — 23 pages listed |
| Clone existing wiki | failed (empty wiki — expected on first publish) |
| Init + commit local wiki | **success** — 23 files committed locally in temp workdir |
| `git push origin HEAD` | **failed** — HTTP 500 from Gitea |

## Manual next step

1. Confirm Gitea wiki is enabled on `commstech/Bugbot` (API reports `has_wiki: true`).
2. Use a token with **wiki write** scope.
3. From Gitea UI: create initial wiki page **or** retry:

```bash
export BUGBOT_GITEA_TOKEN='…'
./scripts/publish-gitea-wiki.sh
```

4. If push still returns 500, check Gitea server logs — wiki git backend may need operator repair.

**Do not claim wiki is populated until `git push` to `Bugbot.wiki.git` succeeds.**
