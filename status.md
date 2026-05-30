# Gitea Bugbot Plugin - Implementation Status

**Last updated:** 2026-05-30  
**Repository:** https://git.commsnet.org/commstech/Bugbot.git

## Current State: BUILD PASSING + TESTS PASSING

```bash
go build -o bin/gitea-bugbot .
go test ./...
```

## Multi-Provider AI (NEW)

Bugbot is no longer locked to OpenWebUI. Supported backends:

- OpenAI, Anthropic Claude, OpenRouter, Ollama, Open WebUI, OpenClaw

See [docs/AI_PROVIDERS.md](docs/AI_PROVIDERS.md) for configuration examples.

## CI/CD

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `.gitea/workflows/ci.yml` | push/PR to main | lint, vet, staticcheck, tests, build, Docker smoke test, changelog artifact |
| `.gitea/workflows/release.yml` | tag `v*` | multi-platform binaries, SHA256SUMS, Gitea release upload |

Release example: `git tag v1.1.0 && git push origin v1.1.0`

Requires `GITEA_TOKEN` secret for automated release publishing.

## Architecture

```
main.go              → HTTP server, multi-provider AI config
ai/client.go         → Provider-agnostic CAH pipeline client
ai/openai_transport  → OpenAI-compatible APIs
ai/anthropic_transport → Claude Messages API
handlers/            → Webhook security + changed-file scoping
analyzers/engine.go  → CAH 5-stage pipeline
limiter/             → Concurrency control
```

## Configuration (key settings)

```yaml
ai_provider: openai        # openai|anthropic|openrouter|ollama|openwebui|openclaw
ai_api_key: "..."
ai_model: "gpt-4o-mini"
skip_startup_checks: false # set true for CI/docker smoke tests
```

Legacy `openwebui_url` / `openwebui_token` still supported.
