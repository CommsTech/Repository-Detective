# Onboarding Web UI

Repository Detective includes a browser-based setup wizard for connecting Gitea repositories without hand-editing webhooks.

## Access

| URL | Auth |
|-----|------|
| `http://your-host:8081/onboard` | Public (UI only) — **recommended install port** |
| `http://your-host:8081/` | Redirects to `/onboard` |

Advanced / minimal compose may use port **8080** instead.

API calls from the wizard require the operator API key (`REPOSITORY_DETECTIVE_API_KEY` in `.env`). **Preferred** header:

```http
X-Repository-Detective-API-Key: your-api-key
```

## Wizard steps

1. **Connection settings** — Gitea URL, access token, public URL, webhook secret, API key
2. **AI provider (optional)** — skip for deterministic-only; prefer local Ollama when enabling AI
3. **Select repositories** — loads repos visible to your token
4. **Register webhooks** — creates push + pull request hooks on selected repos
5. **Environment export** — copy/paste `REPOSITORY_DETECTIVE_*` variables (AI lines omitted when skipped)

You can complete onboarding and run deterministic scans **without** testing or configuring AI.

## Required configuration

Set `public_url` so Gitea can reach Repository Detective for webhooks:

```yaml
public_url: "https://detective.example.com"
```

If the service runs on an internal network and Gitea is public, expose it via port forward, reverse proxy, or tunnel — see [docs/NETWORKING.md](NETWORKING.md). Cloudflare tunnel is optional ([TUNNEL.md](TUNNEL.md)).

Or environment variable:

```bash
REPOSITORY_DETECTIVE_PUBLIC_URL=https://detective.example.com
```

The wizard registers hooks at `{public_url}/webhook`.

## API endpoints

All endpoints are under `/api/v1/onboard/` and require the API key header (preferred: `X-Repository-Detective-API-Key`).

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/onboard/defaults` | Server-side defaults (Gitea URL, public URL, AI provider) |
| POST | `/api/v1/onboard/test-gitea` | Test Gitea token |
| POST | `/api/v1/onboard/test-ai` | Test AI provider (optional) |
| POST | `/api/v1/onboard/repos` | List user repositories |
| POST | `/api/v1/onboard/webhooks` | Register webhooks on selected repos |

### Example: test Gitea

```bash
curl -X POST http://localhost:8081/api/v1/onboard/test-gitea \
  -H "X-Repository-Detective-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{"gitea_url":"https://git.example.com","gitea_token":"your-token"}'
```

### Example: register webhooks

```bash
curl -X POST http://localhost:8081/api/v1/onboard/webhooks \
  -H "X-Repository-Detective-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "gitea_url": "https://git.example.com",
    "gitea_token": "your-token",
    "public_url": "https://detective.example.com",
    "webhook_secret": "shared-secret",
    "repositories": ["org/repo1", "org/repo2"]
  }'
```

## Related

- [QUICKSTART.md](QUICKSTART.md) — recommended install
- [SETUP.md](SETUP.md) — full walkthrough
- [AI_PROVIDERS.md](AI_PROVIDERS.md) — optional AI
