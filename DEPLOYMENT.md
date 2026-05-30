# Deployment

Use **[docs/SETUP.md](docs/SETUP.md)** for the full step-by-step guide.

## Prerequisites

- Gitea instance with a personal access token (repo read, hook write, issue write)
- An AI backend — see [docs/AI_PROVIDERS.md](docs/AI_PROVIDERS.md)
- Docker on the host, or Go 1.21+ to build from source
- A public URL Gitea can reach (port forward, reverse proxy, or tunnel) — see [docs/NETWORKING.md](docs/NETWORKING.md)

## Production on clustermgr

```bash
git clone https://git.commsnet.org/commstech/Bugbot.git
cd Bugbot
cp .env.clustermgr.example .env
# edit .env
docker compose -f docker-compose.clustermgr.yml up -d --build
```

Expose port 8081, set `BUGBOT_PUBLIC_URL`, register webhooks at `/onboard`.

## Compose files

| File | Port | Purpose |
|------|------|---------|
| `docker-compose.clustermgr.yml` | 8081 | Production default |
| `docker-compose.public.yml` | 8081 | Explicit LAN bind for pfSense NAT |
| `docker-compose.minimal.yml` | 8080 | Local dev |

Avoid `docker-compose.yml` and `docker-compose.simple.yml` — outdated env var names.

## Health checks

```bash
curl -m 5 http://127.0.0.1:8081/health
curl -H "X-Bugbot-API-Key: $BUGBOT_API_KEY" http://127.0.0.1:8081/api/v1/status
docker logs gitea-bugbot --tail 50
```

## CI/CD

Gitea Actions in `.gitea/workflows/` run tests on push to `main`. Tag `v*` to build release binaries.

Requires `GITEA_TOKEN` secret for automated release upload.

## Docs index

- [SETUP.md](docs/SETUP.md) — start here
- [NETWORKING.md](docs/NETWORKING.md) — public exposure
- [TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) — when something breaks
- [ONBOARDING.md](docs/ONBOARDING.md) — wizard details
