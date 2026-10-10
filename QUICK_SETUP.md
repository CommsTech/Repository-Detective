# Quick Setup

## Recommended Installation (port 8081)

One path from clone to a running instance. Full detail: **[docs/QUICKSTART.md](docs/QUICKSTART.md)** · **[docs/SETUP.md](docs/SETUP.md)**

```bash
cp .env.example .env   # edit required core vars below
docker compose pull && docker compose up -d
curl -m 5 http://127.0.0.1:8081/health
```

Open http://127.0.0.1:8081/onboard

### Required core configuration

```
REPOSITORY_DETECTIVE_API_KEY
REPOSITORY_DETECTIVE_GITEA_URL          # when connecting a forge
REPOSITORY_DETECTIVE_GITEA_TOKEN
REPOSITORY_DETECTIVE_WEBHOOK_SECRET
REPOSITORY_DETECTIVE_PUBLIC_URL         # after Repository Detective is reachable from Gitea
```

### Optional AI configuration

AI is **not required**. Deterministic scanners run with LLM auditors off by default.

```
# Leave ENABLE_LLM_AUDITORS=false for deterministic-only operation.
# To enable AI later (local Ollama recommended for privacy):
# REPOSITORY_DETECTIVE_ENABLE_LLM_AUDITORS=true
# REPOSITORY_DETECTIVE_AI_PROVIDER=ollama
# REPOSITORY_DETECTIVE_AI_BASE_URL=http://10.x.x.x:11434
# REPOSITORY_DETECTIVE_AI_MODEL=qwen2.5-coder
# REPOSITORY_DETECTIVE_AI_API_KEY=     # if your provider needs one
```

See [docs/AI_PROVIDERS.md](docs/AI_PROVIDERS.md).

## Advanced Installation Options

| Option | Port | Command |
|--------|------|---------|
| Build from source (default compose) | 8081 | `docker compose up -d --build` |
| Minimal / contrib compose | 8080 | `docker compose -f docker-compose.minimal.yml up -d --build` |
| Offline / preloaded image | 8081 | `docker compose -f docker-compose.offline.yml up -d` |

Expose port **8081** to Gitea for the recommended path — [docs/NETWORKING.md](docs/NETWORKING.md).

## Problems

[docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) · Report install issues on [GitHub](https://github.com/CommsTech/Repository-Detective/issues/new?template=installation_problem.yml)

## Testing

[docs/TESTING.md](docs/TESTING.md)
