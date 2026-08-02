# Live all-in-one git redeploy report

Recorded: 2026-06-08

## Build

| Item | Value |
|------|-------|
| Git commit | `9ed0898` |
| Image tag | `repository-detective:all-in-one` |
| Image revision label | `9ed0898` |
| Build verification | `docker run --rm repository-detective:all-in-one git --version` → git 2.45.4 |

## Change

All-in-one stage now runs `docker-alpine-runtime-setup.sh wget su-exec git` and fails the build if `git --version` does not succeed.

## Deploy

```bash
docker stop repository-detective && docker rm repository-detective
docker run -d --name repository-detective --network host --restart unless-stopped \
  --env-file /home/commstech/Repository-Detective/.env \
  -v /home/commstech/Repository-Detective/config:/app/config:ro \
  -v /home/commstech/Repository-Detective/data:/app/data \
  -v /home/commstech/Repository-Detective/certs:/app/certs:ro \
  repository-detective:all-in-one
```

## Post-deploy verification

| Check | Result |
|-------|--------|
| `/health` status | healthy |
| `tools_summary.available_count` | 4 (was 3) |
| `git` in missing list | **removed** |
| `/api/v1/status` git tool | `enabled_available`, version 2.45.4 |
| Pre-install enabled | true |
| Secrets in image | none (env-file mount only) |
| Non-root runtime | `repositorydetective` user preserved |

## Notes

Other scanner binaries (trivy, grype, gitleaks, etc.) still reported missing in `/health` — pre-existing scanner-tools layer gap on this host build; unrelated to git fix.
