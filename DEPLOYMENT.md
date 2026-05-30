# Deployment Guide

**Repository:** https://git.commsnet.org/commstech/Bugbot.git

## Prerequisites

- Gitea server with API access token
- AI provider (OpenAI, Anthropic, Ollama, OpenWebUI, etc.) — see [docs/AI_PROVIDERS.md](docs/AI_PROVIDERS.md)
- Docker (recommended) or Go 1.21+

## Deploy with Docker Compose

```bash
git clone https://git.commsnet.org/commstech/Bugbot.git
cd Bugbot

# Minimal stack (recommended)
docker compose -f docker-compose.minimal.yml up -d --build
```

Configure via environment variables in `docker-compose.minimal.yml` or mount `config/config.yaml`.

## Post-deploy: onboard repositories

### Using the Web UI (recommended)

1. Set `BUGBOT_PUBLIC_URL` to the URL Gitea can reach (e.g. `https://bugbot.example.com`)
2. Open `https://bugbot.example.com/onboard`
3. Enter API key, test connections, select repos, register webhooks

### Manual webhooks

Point each repository webhook to:

```
https://bugbot.example.com/webhook
```

## Configuration reference

| Setting | Purpose |
|---------|---------|
| `BUGBOT_PUBLIC_URL` | Public URL for webhook registration |
| `BUGBOT_API_KEY` | Secures API and onboarding endpoints |
| `BUGBOT_WEBHOOK_SECRET` | Validates incoming Gitea webhooks |
| `BUGBOT_GITEA_URL` / `BUGBOT_GITEA_TOKEN` | Gitea API access |
| `BUGBOT_AI_PROVIDER` / `BUGBOT_AI_API_KEY` / `BUGBOT_AI_MODEL` | AI backend |
| `BUGBOT_ENABLE_SECURITY` | Static + LLM security scanning |
| `BUGBOT_ENABLE_QUALITY` | Static quality rules |
| `BUGBOT_SKIP_STARTUP_CHECKS` | Set `true` when AI/Gitea unavailable at boot |

## Repository filtering

In `config/config.yaml`:

```yaml
repository_include_patterns:
  - "myorg/*"
repository_exclude_patterns:
  - "archived-*"
  - "test-*"
```

Empty include list = all repositories allowed (minus excludes).

## CI/CD

Gitea Actions run on push to `main`:

- Lint, vet, staticcheck
- Unit tests
- Docker build smoke test

Tag releases with `v*` to trigger binary builds and Gitea release upload.

## Health monitoring

```bash
curl http://localhost:8080/health
curl -H "X-Bugbot-API-Key: your-key" http://localhost:8080/api/v1/status
```

## Documentation

- [README.md](README.md) — overview and API
- [docs/ONBOARDING.md](docs/ONBOARDING.md) — Web UI setup
- [docs/AI_PROVIDERS.md](docs/AI_PROVIDERS.md) — AI configuration
- [architecture.md](architecture.md) — system design
- [status.md](status.md) — implementation status
