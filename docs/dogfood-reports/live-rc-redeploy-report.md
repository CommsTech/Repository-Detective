# Live RC redeploy report

**Date:** 2026-06-10  
**Image:** `repository-detective:rc-e3e19ec`  
**Git commit:** `e3e19ec` (+ baseline doc `caf5f48`)

## Deploy method

Image rebuild from Dockerfile `all-in-one` target — **no binary hot-swap**.

```bash
docker stop repository-detective && docker rm repository-detective
docker run -d --name repository-detective --network host --restart unless-stopped \
  --env-file /home/commstech/Repository-Detective/.env \
  -v /home/commstech/Repository-Detective/config:/app/config:ro \
  -v /home/commstech/Repository-Detective/data:/app/data \
  -v /home/commstech/Repository-Detective/certs:/app/certs:ro \
  repository-detective:rc-e3e19ec
```

## Post-deploy verification

| Check | Result |
|-------|--------|
| `/health` | **healthy** |
| `/health` version | **`rc-e3e19ec`** |
| Migrations | complete (no fatal in logs) |
| `/api/v1/ai-recommendations/config` | **200** — feature `ai_recommendations`, enabled false |
| `/ui/repos/1/sbom` | **200** — SBOM summary panel |
| `/ui/findings/37361` | **200** — actionable sections present |
| UI route smoke (15 routes) | **all 200** |
| Container log health | 0 panics, 0 secrets in sample |
| Old binary routes | restored (no 404 on RC routes) |

## Reconciliation note

Product repo `active_present_open` reads **21** after redeploy (latest scan `e42b3e175e313904`). Prior pre-deploy baseline was **2** on old binary — investigate before marketing; may reflect reconciliation definition change or latest scan snapshot, not an all-repo scan.

## Acceptance

**Live RC deployed:** YES  
**Live revision:** `rc-e3e19ec`
