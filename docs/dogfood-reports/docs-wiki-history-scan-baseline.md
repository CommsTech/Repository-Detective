# Docs, wiki, and Git-history secret scan baseline

**Date:** 2026-06-02  
**Latest commit:** `6e56eda` (docs(beta): verify preinstall branding and live policy)  
**Live container revision:** `b7dbe72`

## Repository hygiene

| Check | Status |
|-------|--------|
| `.env` staged | no |
| Local `repository-detective` ELF staged | no |
| `dist/` artifacts staged | no |
| Working tree clean | yes |

## Wiki

| Item | Status |
|------|--------|
| `docs/wiki/` folder present | yes (8 pages) |
| Gitea wiki populated | **no** (API 404 — wiki repo not initialized or empty) |
| Target wiki remote | `https://git.commsnet.org/commstech/repository-detective.wiki.git` |
| Publish script | `scripts/publish-gitea-wiki.sh` |

## Product repo state (before sprint)

| Metric | Value |
|--------|-------|
| Gitea open issues | 1 (#48 operator task) |
| DB `external_issues` open (forge_open_issues) | 132 (stale vs Gitea) |
| Active-present findings | 1201 |
| Latest product scan ID | `47993b1eecb63e47` (dry-run, issue_sync skipped) |

## Secret scanner behavior (before sprint)

| Mode | Supported |
|------|-----------|
| Current tree (`gitleaks dir`) | **yes** — default gitleaks integration |
| Full Git history | **no** — `RunGitleaks` uses filesystem dir mode only |
| Recent commits only | **no** |
| Changed files only | **no** |

Gitleaks findings include commit hash in JSON when present, but archive/API workspaces have no `.git`, so history cannot be scanned today.

## Sprint goals

1. Populate Gitea wiki from `docs/wiki/`
2. Add step-by-step beta guides with screenshot placeholders
3. Implement Git-history credential scanning with clear labeling
4. Full product-repo rescan + `external_issues` reconciliation
5. Fix any new active-present findings from rescan
