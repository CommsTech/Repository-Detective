# RC screenshot / visual QA report

**Date:** 2026-06-11  
**Method:** Headless Chromium via `zenika/alpine-chrome` Docker (host has no native Chromium)

## Captured

| File | Page |
|------|------|
| `dashboard.png` | Dashboard |
| `repos-list.png` | Repositories |
| `repo-detail.png` | Repo detail |
| `finding-detail.png` | Finding 37361 detail |
| `sbom.png` | Scan SBOM (`926a5f56a26f03c9`) |
| `repository-map.png` | Scan graph |
| `preinstall.png` | Pre-install audit |
| `containers.png` | Container scan page |
| `learning.png` | Learning/calibration |
| `configure.png` | Configure |
| `health.png` | Health |
| `runner-status.png` | Runner status |

## Secret scan

`strings` grep on PNGs: no API keys, tokens, or `api_key=` query strings embedded in image bytes.

## Visual checks

| Check | Result |
|-------|--------|
| No clipped primary nav | pass |
| No raw debug stack traces | pass |
| Dark theme default | acceptable |
| Configure shows provider-neutral AI naming | pass |
| SBOM page shows artifact summary | pass |

## Acceptance

| Item | Status |
|------|--------|
| Screenshots captured | **yes** (12 pages) |
| Secrets in screenshots | **none detected** |
| Marketing blocker | **cleared** for screenshots |
