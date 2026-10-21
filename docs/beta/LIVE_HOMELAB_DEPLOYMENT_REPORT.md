# Live homelab beta deployment report

Date: 2026-06-02  
Operator instance: `http://127.0.0.1:8081`

## Goal

Rebuild live homelab container from current `main` so UI routes (`/ui/configure`, `/ui/learning`, static assets) match the private beta packaging sprint.

## Prior state

| Item | Value |
|------|-------|
| Image revision | `f64789d` (2026-06-06) |
| `/ui/configure` | 404 |
| `/ui/learning` | 404 |
| API `/health` | healthy (stale binary) |

## Build method

Full `./scripts/docker-build-verify.sh` **not** run (~23 min matrix). Minimum homelab target built directly:

```bash
cd /home/commstech/Repository-Detective
export RD_VERSION=beta
export RD_COMMIT=$(git rev-parse --short HEAD)
export RD_BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

docker build --target all-in-one -t repository-detective:all-in-one \
  --build-arg INSTALL_EXTERNAL_TOOLS=true \
  --build-arg VERSION="$RD_VERSION" \
  --build-arg COMMIT="$RD_COMMIT" \
  --build-arg BUILD_DATE="$RD_BUILD_DATE" \
  .
```

Build result: **PASS** (~20 min)

Note: `docker-compose -f docker-compose.yml -f docker-compose.host-network.yml build` failed on docker-compose v1.23.2 (`network_mode` and `networks` cannot be combined). Used direct `docker build` + manual recreate instead.

## Deploy

Secrets preserved via bind mounts and `--env-file` only — **not** baked into image.

```bash
docker stop repository-detective && docker rm repository-detective

docker run -d --name repository-detective \
  --network host \
  --restart unless-stopped \
  --env-file /home/commstech/Repository-Detective/.env \
  -v /home/commstech/Repository-Detective/config:/app/config:ro \
  -v /home/commstech/Repository-Detective/data:/app/data \
  -v /home/commstech/Repository-Detective/certs:/app/certs:ro \
  repository-detective:all-in-one
```

## New image metadata

| Label | Value |
|-------|-------|
| `org.opencontainers.image.revision` | `46cf4bf` |
| `org.opencontainers.image.version` | `beta` |
| `org.opencontainers.image.created` | `2026-06-07T17:48:54Z` |

## Post-deploy verification

```bash
curl -sS http://127.0.0.1:8081/health
curl -sS -H "X-Repository-Detective-API-Key: $REPOSITORY_DETECTIVE_API_KEY" \
  http://127.0.0.1:8081/api/v1/status
```

| Check | Result |
|-------|--------|
| `/health` | healthy |
| `database_healthy` | true |
| `/api/v1/status` | PASS |
| Secrets in image | None (verified) |
| Scanners available | 10 configured |

## UI parity

See [LIVE_UI_ROUTE_VERIFICATION.md](LIVE_UI_ROUTE_VERIFICATION.md) — configure, learning, preinstall, health, static assets **PASS** after redeploy.

## Config notes

Live `config/config.yaml` still uses operator homelab settings (`scan_profile: standard_deterministic`, `preinstall_audit_enabled: false`). Testers receive `config/private-beta.example.yaml` via beta bundle. Operator may merge beta safety keys separately.

## Rollback

```bash
# If needed, redeploy previous image by tag/digest from docker images history
docker stop repository-detective && docker rm repository-detective
# restore prior image ID 40503c7708ac (f64789d) with same run command
```

Database bind mount `./data` preserved across redeploy — no migration rollback required.
