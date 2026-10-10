# RC all-in-one build report

**Date:** 2026-06-10  
**Commit:** `e3e19ec`  
**Image tag:** `repository-detective:rc-e3e19ec` (also `repository-detective:all-in-one`)

## Build command

```bash
docker build --target all-in-one \
  -t repository-detective:rc-e3e19ec \
  -t repository-detective:all-in-one \
  --build-arg VERSION=rc-e3e19ec \
  --build-arg COMMIT=e3e19ec \
  .
```

## Result

| Check | Status |
|-------|--------|
| Image build | **PASS** (~22 min) |
| Binary at `/app/repository-detective` | **PASS** (20.5M, executable) |
| Alpine musl static binary | **PASS** (built in golang:alpine builder) |
| Secrets in image | **PASS** (no `.env` copied) |
| Scanner tools layer | included (INSTALL_EXTERNAL_TOOLS default true) |

## Root cause of prior hot-swap failure

Host `make beta-release` produces a binary via `golang:1.23-bookworm` (glibc). All-in-one runtime is Alpine (musl). Copying host binary → `exec: not found`. **Image rebuild is the correct fix.**

## Image ID

`7eaae54e67a6` (542MB)
