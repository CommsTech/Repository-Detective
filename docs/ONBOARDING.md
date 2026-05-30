# Onboarding Web UI

Bugbot includes a browser-based setup wizard for connecting Gitea repositories without hand-editing webhooks.

## Access

| URL | Auth |
|-----|------|
| `http://your-bugbot-host:8080/onboard` | Public (UI only) |
| `http://your-bugbot-host:8080/` | Redirects to `/onboard` |

API calls from the wizard require the Bugbot API key (`BUGBOT_API_KEY` / `api_key` in config). Send it as header:

```
X-Bugbot-API-Key: your-api-key
```

## Wizard steps

1. **Connection settings** — Gitea URL, access token, Bugbot public URL, webhook secret, API key
2. **AI provider** — provider, base URL, API key, model (test connection before continuing)
3. **Select repositories** — loads repos visible to your token
4. **Register webhooks** — creates push + pull request hooks on selected repos
5. **Environment export** — copy/paste variables for Docker or systemd deployment

## Required configuration

Set `public_url` so Gitea can reach Bugbot for webhooks:

```yaml
public_url: "https://bugbot.example.com"
```

If Bugbot runs on an internal network and Gitea is public, expose Bugbot via port forward, reverse proxy, or tunnel — see [docs/NETWORKING.md](docs/NETWORKING.md). Cloudflare tunnel is optional ([TUNNEL.md](docs/TUNNEL.md)).

Or environment variable:

```bash
BUGBOT_PUBLIC_URL=https://bugbot.example.com
```

The wizard registers hooks at `{public_url}/webhook`.

## API endpoints

All endpoints are under `/api/v1/onboard/` and require `X-Bugbot-API-Key`.

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/onboard/defaults` | Server-side defaults (Gitea URL, public URL, AI provider) |
| POST | `/api/v1/onboard/test-gitea` | Test Gitea token |
| POST | `/api/v1/onboard/test-ai` | Test AI provider |
| POST | `/api/v1/onboard/repos` | List user repositories |
| POST | `/api/v1/onboard/webhooks` | Register webhooks on selected repos |

### Example: test Gitea

```bash
curl -X POST http://localhost:8080/api/v1/onboard/test-gitea \
  -H "X-Bugbot-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{"gitea_url":"https://git.example.com","gitea_token":"your-token"}'
```

### Example: register webhooks

```bash
curl -X POST http://localhost:8080/api/v1/onboard/webhooks \
  -H "X-Bugbot-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "gitea_url": "https://git.example.com",
    "gitea_token": "your-token",
    "public_url": "https://bugbot.example.com",
    "webhook_secret": "shared-secret",
    "repositories": ["org/repo1", "org/repo2"]
  }'
```

## Token permissions

The Gitea token used for onboarding needs:

- Read access to repositories you want to scan
- Write access to repository hooks (for webhook registration)
- Write access to issues (if `auto_create_issues` is enabled)

## Manual webhook setup

If you prefer not to use the wizard, configure Gitea webhooks manually:

- **URL**: `{BUGBOT_PUBLIC_URL}/webhook`
- **Content type**: `application/json`
- **Secret**: same as `webhook_secret`
- **Events**: Push, Pull request

See also [AI_PROVIDERS.md](AI_PROVIDERS.md) for AI backend configuration.
