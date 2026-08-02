# Deployment

See **[docs/DEPLOYMENT_ISSUES.md](docs/DEPLOYMENT_ISSUES.md)** for known deployment problems and workarounds on this host.

See **[docs/SETUP.md](docs/SETUP.md)** for the full walkthrough.

## Prerequisites

- Gitea with an API token (repo read, hook write, issue write if auto-creating issues)
- An AI backend — [docs/AI_PROVIDERS.md](docs/AI_PROVIDERS.md)
- Docker, or Go 1.21+ to build from source
- A URL Gitea can reach for webhooks — [docs/NETWORKING.md](docs/NETWORKING.md)

## Quick deploy

```bash
git clone https://git.commsnet.org/commstech/repository-detective.git
cd repository-detective
cp .env.example .env   # or copy from a legacy install at ~/bugbot/.env
# edit .env
docker compose up -d --build
curl -m 5 http://127.0.0.1:8081/health
```

Or use the helper script:

```bash
./deploy.sh
./deploy.sh --scan     # optional: dogfood scan on commstech/repository-detective
```

### DNS-filtered networks

If `docker build` fails on `storage.googleapis.com` (Go module proxy redirects), vendor dependencies first:

```bash
./scripts/vendor-deps.sh
docker compose up -d --build
```

This vendors dependencies using the official Go module proxy first:

```bash
GOPROXY=https://proxy.golang.org,direct GOSUMDB=sum.golang.org ./scripts/vendor-deps.sh
docker compose up -d --build
```

If `proxy.golang.org` is blocked, `vendor-deps.sh` can retry with `goproxy.io` as a **temporary local workaround only** — not recommended for security-sensitive or government deployments. Prefer an internal artifact proxy or fully offline `vendor/` + `GOPROXY=off`. The `vendor/` directory is not committed.

When Docker bridge IP pools are exhausted, the default `docker-compose.yml` uses `network_mode: host` (listens on port 8081).

Disable the legacy systemd unit after Docker is healthy:

```bash
sudo systemctl disable --now bugbot.service
```

## Compose files

| File | Port | Builds image? |
|------|------|---------------|
| `docker-compose.minimal.yml` | 8080 | yes |
| `docker-compose.yml` | 8081 | yes |
| `docker-compose.offline.yml` | 8081 | no — load tar first |

## Pre-built image transfer

When the target host cannot build (no internet, no Go proxy):

```bash
docker compose build
docker save repository-detective:latest -o repository-detective-image.tar
# copy tar to target host
docker load -i repository-detective-image.tar
docker compose -f docker-compose.offline.yml up -d
```

Works with legacy `docker-compose` 1.x as well as `docker compose` v2.

## Health checks

```bash
curl -m 5 http://127.0.0.1:8081/health
curl -H "X-Repository-Detective-API-Key: $REPOSITORY_DETECTIVE_API_KEY" http://127.0.0.1:8081/api/v1/status
docker logs repository-detective --tail 50
```

## CI/CD

`.gitea/workflows/` runs tests on push to `main`. Tag `v*` to build release binaries.

Local test guide: [docs/TESTING.md](docs/TESTING.md)

## Docs

- [SETUP.md](docs/SETUP.md)
- [NETWORKING.md](docs/NETWORKING.md)
- [TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)
