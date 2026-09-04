# Deployment

See **[docs/DEPLOYMENT_ISSUES.md](docs/DEPLOYMENT_ISSUES.md)** for known deployment problems and workarounds on this host.

See **[docs/SETUP.md](docs/SETUP.md)** for the full walkthrough and **[docs/QUICKSTART.md](docs/QUICKSTART.md)** for the recommended path.

## Prerequisites

### Required

- Docker (Compose v2 recommended)
- A Gitea API token when connecting a forge (repo read, hook write, issue write if auto-creating issues)
- A URL Gitea can reach for webhooks when using forge integration — [docs/NETWORKING.md](docs/NETWORKING.md)

### Optional

- An AI backend — **not required** for deterministic scanning. Prefer local Ollama / OpenAI-compatible endpoints for privacy — [docs/AI_PROVIDERS.md](docs/AI_PROVIDERS.md)
- Go 1.25+ only if building from source without Docker

## Recommended Installation

```bash
git clone https://git.commsnet.org/commstech/Repository-Detective.git
cd Repository-Detective
cp .env.example .env   # or copy from a legacy install
# edit .env — API key required; AI vars optional
docker compose pull && docker compose up -d
curl -m 5 http://127.0.0.1:8081/health
```

Or use the helper script:

```bash
./deploy.sh
./deploy.sh --scan     # optional: dogfood scan on commstech/Repository-Detective
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

When Docker bridge IP pools are exhausted, apply the **optional** host-network overlay:

```bash
docker compose -f docker-compose.yml -f docker-compose.host-network.yml up -d
```

Default `docker-compose.yml` uses **bridge** networking and publishes port **8081**.

Disable the legacy systemd unit after Docker is healthy:

```bash
sudo systemctl disable --now repository-detective.service
```

## Advanced Installation Options — Compose files

| File | Port | Builds image? | Role |
|------|------|---------------|------|
| `docker-compose.yml` | **8081** | pull preferred | **Recommended** |
| `docker-compose.minimal.yml` | 8080 | yes | Local contrib / no registry |
| `docker-compose.offline.yml` | 8081 | no — load tar first | Air-gap |
| `docker-compose.beta.yml` | 8081 | yes | Private beta profile |
| `docker-compose.host-network.yml` | 8081 | — | Overlay when bridge pools exhausted |
| `docker-compose.traefik.yml` | — | — | Reverse-proxy overlay |

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

`ai_provider: disabled` (or AI Analysis: Disabled) with LLM auditors off is expected — not a failed install.

## CI/CD

`.gitea/workflows/` runs tests on push to `main`. Tag `v*` to build release binaries.

Local test guide: [docs/TESTING.md](docs/TESTING.md)

## Docs

- [SETUP.md](docs/SETUP.md)
- [NETWORKING.md](docs/NETWORKING.md)
- [TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)
- [AI_PROVIDERS.md](docs/AI_PROVIDERS.md) (optional)
