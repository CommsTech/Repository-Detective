# Live docs and history scan verification

**Date:** 2026-06-08  
**Live image revision:** `ca56dbf` (all-in-one rebuild + redeploy)  
**Git HEAD:** `c283d2a` (graph noise fix pending image rebuild)

## UI / API

| Check | Status |
|-------|--------|
| Pre-install audit enabled | **yes** (`preinstall_audit_enabled: true`) |
| `/ui/repos` | **pass** (smoke test) |
| `/ui/configure` | **pass** (operator UI healthy) |
| Secret scanning section in Configure model | **yes** (`secret-scanning` section in code) |
| `/ui/learning` | **pass** (prior sprint) |
| Manual scans | **pass** — scan `b21dc57c40411f31` completed |
| Wiki link | N/A — wiki not populated on Gitea (push 500) |

## Secret history scanning

| Item | Status |
|------|--------|
| Code shipped (`gitleaks-history`) | **yes** (commit `f64b922`) |
| Config defaults | `secret_scan_git_history_enabled: true` |
| Live container gitleaks binary | **verify on next all-in-one rebuild** |
| History findings in product scan | Not yet observed (requires git clone + gitleaks in scan path) |

## Product repo reconciliation

| Metric | Value |
|--------|-------|
| Latest scan | `b21dc57c40411f31` |
| Active-present after graph calibration v1 | **876** (down from 1192) |
| Gitea open issues | **1** (#48) |
| DB `external_issues` open | **0** |

## Operator smoke test

`./scripts/operator-smoke-test.sh` — **pass**

## Note

Binary hot-swap was attempted and reverted; use full `docker build --target all-in-one` for `c283d2a` to deploy orphan-function graph fix and verify active-present drop.
