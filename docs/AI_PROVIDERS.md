# AI Provider Configuration

Repository Detective uses a **provider-agnostic AI layer**. All CAH pipeline stages (Prepare, Scan, Validate, Prove) call the same chat interface regardless of backend.

## Supported Providers

| Provider | `ai_provider` | Default Base URL | Default Model | API Key Required |
|----------|---------------|------------------|---------------|------------------|
| OpenAI | `openai` | `https://api.openai.com/v1` | `gpt-4o-mini` | Yes |
| Anthropic Claude | `anthropic` | `https://api.anthropic.com/v1` | `claude-3-5-haiku-latest` | Yes |
| OpenRouter | `openrouter` | `https://openrouter.ai/api/v1` | `openai/gpt-4o-mini` | Yes |
| Ollama | `ollama` | `http://127.0.0.1:11434/v1` | `llama3.2` | No |
| Open WebUI | `openwebui` | *(must configure)* | `default` | Optional |
| OpenClaw | `openclaw` | `http://127.0.0.1:18789/v1` | `openclaw/default` | Optional |

## Configuration

### YAML (`config/config.yaml`)

```yaml
ai_provider: openai
ai_base_url: ""           # optional override
ai_api_key: "sk-..."
ai_model: "gpt-4o-mini"
# Homelab / private CA only (never on production internet-facing hosts):
# ai_insecure_skip_tls_verify: true
```

### Environment Variables

```bash
REPOSITORY_DETECTIVE_AI_PROVIDER=anthropic
REPOSITORY_DETECTIVE_AI_API_KEY=sk-ant-...
REPOSITORY_DETECTIVE_AI_MODEL=claude-3-5-sonnet-latest
```

Legacy `REPOSITORY_DETECTIVE_AI_*` env vars remain supported.

### Legacy OpenWebUI

Existing deployments using `openwebui_url` continue to work without changes. The factory auto-maps legacy settings to `ai_provider: openwebui`.

## Provider Examples

### OpenAI

```yaml
ai_provider: openai
ai_api_key: sk-proj-...
ai_model: gpt-4o
```

### Anthropic

```yaml
ai_provider: anthropic
ai_api_key: sk-ant-...
ai_model: claude-3-5-sonnet-latest
```

### OpenRouter

```yaml
ai_provider: openrouter
ai_api_key: sk-or-...
ai_model: anthropic/claude-3.5-sonnet
```

### Ollama (local)

```yaml
ai_provider: ollama
ai_base_url: http://127.0.0.1:11434/v1
ai_model: llama3.2
```

### Open WebUI

```yaml
ai_provider: openwebui
ai_base_url: http://openwebui:8080
ai_api_key: your-token
ai_model: llama3
```

### OpenClaw

OpenClaw exposes an OpenAI-compatible `/v1/chat/completions` endpoint on the gateway (default port 18789). Enable it in `~/.openclaw/openclaw.json`:

```json
{
  "gateway": {
    "http": {
      "endpoints": {
        "chatCompletions": { "enabled": true }
      }
    }
  }
}
```

Repository Detective config:

```yaml
ai_provider: openclaw
ai_base_url: http://127.0.0.1:18789/v1
ai_api_key: your-gateway-token
ai_model: openclaw/default
```

## Architecture

```
analyzers/engine.go
        │
        ▼
   ai.Client  ──► ChatTransport interface
                      ├── OpenAICompatibleTransport (OpenAI, OpenRouter, Ollama, OpenWebUI, OpenClaw)
                      └── AnthropicTransport (Claude Messages API)
```

## Testing Connection

On startup Repository Detective calls `TestConnection()` against the configured provider unless `skip_startup_checks: true`.

Check runtime status:

```bash
curl -H "X-Repository-Detective-API-Key: $REPOSITORY_DETECTIVE_API_KEY" http://localhost:8080/api/v1/status
```

(Legacy header `X-Repository-Detective-API-Key` and env `REPOSITORY_DETECTIVE_API_KEY` still work.)

Response includes `ai_provider` and `ai_model`.

## CI / Release

- **CI** (`.gitea/workflows/ci.yml`): lint, vet, staticcheck, tests, build, Docker smoke test
- **Release** (`.gitea/workflows/release.yml`): triggered on `v*` tags, builds multi-platform binaries, publishes Gitea release

To cut a release:

```bash
git tag v1.1.0
git push origin v1.1.0
```

Set `GITEA_TOKEN` repository secret for automated release upload.
