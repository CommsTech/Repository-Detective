# Deployment

See **[docs/SETUP.md](docs/SETUP.md)** for the full walkthrough.

## Prerequisites

- Gitea with an API token (repo read, hook write, issue write if auto-creating issues)
- An AI backend — [docs/AI_PROVIDERS.md](docs/AI_PROVIDERS.md)
- Docker, or Go 1.21+ to build from source
- A URL Gitea can reach for webhooks — [docs/NETWORKING.md](docs/NETWORKING.md)

## Quick deploy

```bash
git clone https://git.commsnet.org/commstech/Bugbot.git
cd Bugbot
cp .env.example .env
# edit .env
docker compose -f docker-compose.public.yml up -d --build
curl -m 5 http://127.0.0.1:8081/health
```

## Compose files

| File | Port | Builds image? |
|------|------|---------------|
| `docker-compose.minimal.yml` | 8080 | yes |
| `docker-compose.public.yml` | 8081 | yes |
| `docker-compose.offline.yml` | 8081 | no — load tar first |

## Pre-built image transfer

When the target host cannot build (no internet, no Go proxy):

```bash
docker build -t gitea-bugbot:latest .
docker save gitea-bugbot:latest -o gitea-bugbot-image.tar
# copy tar to target host
docker load -i gitea-bugbot-image.tar
docker compose -f docker-compose.offline.yml up -d
```

Works with legacy `docker-compose` 1.x as well as `docker compose` v2.

## Health checks

```bash
curl -m 5 http://127.0.0.1:8081/health
curl -H "X-Bugbot-API-Key: $BUGBOT_API_KEY" http://127.0.0.1:8081/api/v1/status
docker logs gitea-bugbot --tail 50
```

## CI/CD

`.gitea/workflows/` runs tests on push to `main`. Tag `v*` to build release binaries.

## Docs

- [SETUP.md](docs/SETUP.md)
- [NETWORKING.md](docs/NETWORKING.md)
- [TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)
