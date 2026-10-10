# Pre-install git and product rescan baseline

Recorded: 2026-06-08

## Repository state

| Item | Value |
|------|-------|
| Git HEAD | `6679e7f` |
| Working tree | clean |
| No `.env` / binaries / `dist/` staged | confirmed |

## Live deployment (pre-fix)

| Item | Value |
|------|-------|
| Container | `repository-detective` |
| Image | `repository-detective:all-in-one` |
| Image revision label | `b5f44f6` |
| `/health` status | healthy |

## Scanner / tool status (`/health` tools_summary)

| Metric | Value |
|--------|-------|
| available_count | 3 |
| configured_count | 10 |
| missing | git, trivy, grype, gitleaks, semgrep, hadolint, checkov |

**Root cause:** `git` not present in runtime image (`docker run … git --version` → not found; `/usr/bin/git` absent). Pre-install shallow clone fails live despite sandbox hardening.

## Product repo (ID 1)

| Metric | Value |
|--------|-------|
| Latest scan ID | `1ec1a0ebe4bd660e` |
| Graph state (API) | available |
| Graph nodes / edges | 3674 / 6053 |
| Active-present | 87 |
| Gitea open issues | 1 (#48 operator task) |

## Pre-install sandbox

| Control | Status |
|---------|--------|
| Sandbox enabled (config) | yes |
| Private IP blocking (live) | verified prior sprint |
| Live clone audit | **blocked** — git missing |
| Report-only / 0 issues / 0 PRs | yes |

## Sprint goal

Install `git` in all-in-one image, redeploy, verify live pre-install clone, run clean product rescan + issue sync, triage remaining active-present.
