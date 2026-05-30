# Gitea Bugbot Plugin

Automated AI code review for Gitea — scans pushes and pull requests, detects security and quality issues, and creates labeled Gitea issues with code references and proof-of-concept details.

## Features

- **CAH security pipeline** — Prepare → Scan → Validate → Dedup → Prove
- **Deterministic pre-scan** — regex-based checks run before LLM calls to save tokens
- **Multi-provider AI** — OpenAI, Anthropic, OpenRouter, Ollama, OpenWebUI, OpenClaw
- **Scoped analysis** — push events analyze changed files only; PRs analyze diff files only
- **Repository filtering** — include/exclude patterns per repo
- **Onboarding Web UI** — browser wizard to test connections, pick repos, and register webhooks
- **Automatic issues** — labeled issues with file, line, snippet, and PoC when available
- **Webhook security** — rate limiting and secret verification
- **CI/CD** — Gitea Actions for lint, test, and release builds

## Quick start

### Option A: Onboarding wizard (recommended)

1. Start Bugbot (Docker or `go run .`)
2. Open **`http://localhost:8080/onboard`**
3. Enter your API key (`BUGBOT_API_KEY`), Gitea token, AI settings, and public URL
4. Test connections → load repos → register webhooks

See [docs/ONBOARDING.md](docs/ONBOARDING.md) for details.

### Option B: Docker Compose

```bash
git clone https://git.commsnet.org/commstech/Bugbot.git
cd Bugbot

# Edit config/config.yaml or set env vars in docker-compose.minimal.yml
docker compose -f docker-compose.minimal.yml up -d --build
```

### Option C: Manual config

Edit `config/config.yaml`:

```yaml
port: "8080"
api_key: "change-me"
public_url: "http://localhost:8080"

gitea_url: "https://git.example.com"
gitea_token: "your-gitea-token"
webhook_secret: "your-webhook-secret"

ai_provider: "openai"
ai_api_key: "your-key"
ai_model: "gpt-4o-mini"

enable_security: true
enable_quality: true
auto_create_issues: true
```

## Configuration

Environment variables use the `BUGBOT_` prefix (e.g. `BUGBOT_GITEA_URL`).

| Option | Env var | Description |
|--------|---------|-------------|
| `port` | `BUGBOT_PORT` | HTTP port (default `8080`) |
| `api_key` | `BUGBOT_API_KEY` | Protects `/api/v1/*` and onboarding API |
| `public_url` | `BUGBOT_PUBLIC_URL` | Public URL Gitea uses for webhooks |
| `gitea_url` | `BUGBOT_GITEA_URL` | Gitea server URL |
| `gitea_token` | `BUGBOT_GITEA_TOKEN` | Gitea API token |
| `webhook_secret` | `BUGBOT_WEBHOOK_SECRET` | Webhook HMAC secret |
| `ai_provider` | `BUGBOT_AI_PROVIDER` | AI backend (see [docs/AI_PROVIDERS.md](docs/AI_PROVIDERS.md)) |
| `ai_base_url` | `BUGBOT_AI_BASE_URL` | Provider base URL (optional) |
| `ai_api_key` | `BUGBOT_AI_API_KEY` | Provider API key |
| `ai_model` | `BUGBOT_AI_MODEL` | Model name |
| `enable_security` | `BUGBOT_ENABLE_SECURITY` | Security scanning (static + LLM) |
| `enable_quality` | `BUGBOT_ENABLE_QUALITY` | Quality checks (static rules) |
| `repository_include_patterns` | — | Allow-list (empty = all) |
| `repository_exclude_patterns` | — | Block-list (e.g. `archived-*`) |
| `max_concurrent_analyses` | `BUGBOT_MAX_CONCURRENT_ANALYSES` | Parallel analysis limit |
| `skip_startup_checks` | `BUGBOT_SKIP_STARTUP_CHECKS` | Skip Gitea/AI ping on startup |

Legacy `openwebui_url` / `openwebui_token` still work and map to the OpenWebUI provider.

## API endpoints

| Endpoint | Auth | Purpose |
|----------|------|---------|
| `GET /health` | None | Health check |
| `GET /onboard` | None | Onboarding wizard UI |
| `POST /webhook` | Webhook secret | Gitea webhook receiver |
| `POST /api/v1/analyze` | API key | Trigger manual analysis |
| `GET /api/v1/status` | API key | Service status |
| `POST /api/v1/config/reload` | API key | Reload config |
| `POST /api/v1/onboard/*` | API key | Onboarding API |

### Manual analysis

```bash
curl -X POST http://localhost:8080/api/v1/analyze \
  -H "X-Bugbot-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{"owner":"org","repository":"repo","ref":"main","type":"repository"}'
```

## Analysis pipeline

1. **Prepare** — map repository structure and attack surface
2. **Scan** — deterministic pattern rules, then LLM auditors on flagged files (full scan if static finds nothing)
3. **Validate** — advocate/counsel debate (static high-confidence hits skip debate)
4. **Dedup** — merge duplicate findings
5. **Prove** — generate PoC for validated findings

See [docs/CAH_PIPELINE.md](docs/CAH_PIPELINE.md) for the full specification.

## Project structure

```
Bugbot/
├── main.go
├── ai/                 # Multi-provider AI client
├── analyzers/          # CAH engine + static rules
├── gitea/              # Gitea API + hooks/labels
├── handlers/           # Webhooks, onboarding, repo filters
├── issues/             # Issue creation with labels
├── web/                # Embedded onboarding UI
├── config/             # config.yaml
├── docs/               # AI providers, CAH, onboarding
└── .gitea/workflows/   # CI and release
```

## Development

```bash
go mod download
go build -o gitea-bugbot .
go test ./...
```

## Documentation

- [Onboarding Web UI](docs/ONBOARDING.md)
- [AI providers](docs/AI_PROVIDERS.md)
- [CAH pipeline](docs/CAH_PIPELINE.md)
- [Architecture](architecture.md)
- [Status](status.md)

## License

MIT License
