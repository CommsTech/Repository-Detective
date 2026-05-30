# Gitea Bugbot Plugin - Implementation Status

**Last updated:** 2026-05-30  
**Repository:** https://git.commsnet.org/commstech/Bugbot.git

## Current State: BUILD PASSING + TESTS PASSING

```bash
go build -o bin/gitea-bugbot .
go test ./...
```

## Recent additions

| Feature | Status |
|---------|--------|
| Deterministic static scanner (`analyzers/static.go`) | Done — runs before LLM |
| File content in SCAN stage | Done — fetched via Gitea API |
| `enable_security` / `enable_quality` flags | Done — wired in engine |
| Repository include/exclude filters | Done — webhook + config |
| Issue labels (resolve/create) | Done — `gitea.ResolveLabelIDs` |
| PoC / file / line in issues | Done — from Prove stage |
| Onboarding Web UI | Done — `/onboard` |
| Docker compose env alignment | Done — `docker-compose.minimal.yml` |
| Config unmarshaling fix | Done — `skip_patterns`, `language_mapping`, repo patterns |

## Multi-Provider AI

Supported backends: OpenAI, Anthropic, OpenRouter, Ollama, Open WebUI, OpenClaw

See [docs/AI_PROVIDERS.md](docs/AI_PROVIDERS.md).

## Onboarding

Browser wizard at `/onboard` — see [docs/ONBOARDING.md](docs/ONBOARDING.md).

Requires `public_url` / `BUGBOT_PUBLIC_URL` for webhook registration.

## CI/CD

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `.gitea/workflows/ci.yml` | push/PR to main | lint, vet, staticcheck, tests, build, Docker smoke |
| `.gitea/workflows/release.yml` | tag `v*` | multi-platform binaries + Gitea release |

## Architecture

```
main.go              → HTTP server, routes, webhook processor
handlers/onboarding  → Web UI + setup API
handlers/webhook     → Rate limit, secret verify, repo filter
analyzers/engine.go  → CAH pipeline + static pre-scan
analyzers/static.go  → Deterministic pattern rules
ai/client.go         → Multi-provider LLM client
gitea/hooks.go       → Repos, webhooks, labels
issues/manager.go    → Labeled issues with PoC
web/                 → Embedded onboarding assets
```

## Configuration (key settings)

```yaml
api_key: "..."
public_url: "https://bugbot.example.com"
ai_provider: openai
enable_security: true
enable_quality: true
repository_exclude_patterns:
  - "archived-*"
skip_startup_checks: false
```

Legacy `openwebui_url` / `openwebui_token` still supported.
