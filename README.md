# Gitea Bugbot

Automated code review for Gitea. Bugbot watches pushes and pull requests, runs security and quality checks, and opens Gitea issues when it finds problems.

Repo: https://git.commsnet.org/commstech/Bugbot.git

## Setup

**Start here:** [docs/SETUP.md](docs/SETUP.md) — step-by-step from clone to working webhooks.

Quick local trial:

```bash
git clone https://git.commsnet.org/commstech/Bugbot.git && cd Bugbot
docker compose -f docker-compose.minimal.yml up -d --build
curl http://localhost:8080/health
```

Then open http://localhost:8080/onboard

## What it does

- Scans changed files on push; scans PR diff files on pull requests
- Runs static pattern checks first, then LLM analysis on flagged files
- Creates Gitea issues with severity, file, line, code snippet, and PoC when available
- Supports OpenAI, Anthropic, OpenRouter, Ollama, OpenWebUI, and OpenClaw

## Configuration

Environment variables use the `BUGBOT_` prefix. Examples:

| Setting | Variable |
|---------|----------|
| HTTP port | `BUGBOT_PORT` (default `8080`) |
| Bind address | `BUGBOT_LISTEN_HOST` (default `0.0.0.0`) |
| API key | `BUGBOT_API_KEY` |
| Public URL for webhooks | `BUGBOT_PUBLIC_URL` |
| Gitea | `BUGBOT_GITEA_URL`, `BUGBOT_GITEA_TOKEN` |
| Webhook secret | `BUGBOT_WEBHOOK_SECRET` |
| AI | `BUGBOT_AI_PROVIDER`, `BUGBOT_AI_BASE_URL`, `BUGBOT_AI_API_KEY`, `BUGBOT_AI_MODEL` |
| Skip Gitea/AI ping on boot | `BUGBOT_SKIP_STARTUP_CHECKS=true` |

Repo include/exclude patterns and skip patterns are set in `config/config.yaml` only (not env vars).

Full AI provider examples: [docs/AI_PROVIDERS.md](docs/AI_PROVIDERS.md)

## HTTP endpoints

| Path | Auth | Notes |
|------|------|-------|
| `GET /health` | none | Returns `503 starting` then `200 healthy` |
| `GET /onboard` | none | Setup wizard |
| `POST /webhook` | webhook secret | Gitea calls this |
| `POST /api/v1/analyze` | API key | Manual scan trigger |
| `GET /api/v1/status` | API key | Runtime info |
| `POST /api/v1/onboard/*` | API key | Wizard backend |

API key header: `X-Bugbot-API-Key: your-key`

## Documentation

| Doc | Contents |
|-----|----------|
| [docs/SETUP.md](docs/SETUP.md) | Full setup, step by step |
| [docs/NETWORKING.md](docs/NETWORKING.md) | pfSense, reverse proxy, Traefik |
| [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) | Common failures |
| [docs/ONBOARDING.md](docs/ONBOARDING.md) | Wizard and API details |
| [docs/AI_PROVIDERS.md](docs/AI_PROVIDERS.md) | AI backend config |
| [docs/CAH_PIPELINE.md](docs/CAH_PIPELINE.md) | Analysis pipeline spec |
| [docs/SCANNERS.md](docs/SCANNERS.md) | Trivy, Grype, linters (deterministic) |
| [docs/TUNNEL.md](docs/TUNNEL.md) | Cloudflare tunnel (optional) |

## Development

```bash
go build -o gitea-bugbot .
go test ./...
```

## License

MIT
