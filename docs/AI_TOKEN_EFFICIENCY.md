# AI token efficiency

Repository Detective avoids spending paid LLM tokens on unnecessary connection tests.

## Defaults

```yaml
ai_startup_test_enabled: false
ai_connection_test_mode: metadata_only   # metadata_only | manual | chat_completion
ai_connection_test_cache_minutes: 60
ai_max_tokens_per_scan: 0                # reserved for future budget enforcement
```

## Behavior

| Condition | Startup test |
|-----------|--------------|
| Deterministic profile / AI not required | **Skipped** — no AI client |
| `ai_startup_test_enabled: false` | **Skipped** — status: configured but not tested |
| `metadata_only` | GET `/v1/models` when supported (no generation) |
| `chat_completion` | Minimal chat (manual `/api/v1/ai/test-connection` only by default) |

## API

```text
GET  /api/v1/ai/status
POST /api/v1/ai/test-connection?force=true
```

Manual chat tests log a cost warning. Usage counts are recorded when providers return token metadata.

## What we do not do

- Send "Hello, connection test" on every startup
- Test AI for `beta_standard` / depth-2 deterministic profiles
- Repeat tests on every config reload (cache applies)

See [POLICY.md](POLICY.md) for AI policy per repo.
