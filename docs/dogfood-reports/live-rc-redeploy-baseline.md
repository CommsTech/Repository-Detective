# Live RC redeploy baseline

**Recorded:** 2026-06-10  
**Git commit:** `e3e19ec`  
**Mission:** Rebuild all-in-one from source and redeploy — no binary hot-swap.

## Pre-redeploy live state

| Item | Value |
|------|-------|
| Container | `repository-detective` |
| Image tag | `repository-detective:all-in-one` (pre-RC build) |
| Network | `host` |
| Health | `healthy` |
| `/health` version | `beta` (not `e3e19ec`) |
| Product active-present (repo 1) | 2 |
| High/critical product findings | 0 |

## Routes missing on old binary

| Route | Pre-redeploy |
|-------|----------------|
| `GET /api/v1/ai-recommendations/config` | **404** |
| `GET /ui/repos/1/sbom` | **404** |
| Engineer-actionable finding detail template | not deployed |

## Hot-swap failure (prior attempt)

Host-built static binary copied into running all-in-one caused:

```text
/app/repository-detective: not found
```

Likely cause: host `golang:1.23-bookworm` (glibc) binary vs Alpine (musl) runtime in all-in-one image. **Fix:** rebuild image from Dockerfile `all-in-one` target.

## Known blockers (unchanged)

- Gitea wiki HTTP 500
- GitHub issue filing RC-unproven
- External clean install not proven
- Marketing NOT READY

## Redeploy plan

1. `docker build --target all-in-one -t repository-detective:rc-e3e19ec .`
2. Recreate container with host network + existing volume mounts
3. Verify RC routes and finding 37361 live
