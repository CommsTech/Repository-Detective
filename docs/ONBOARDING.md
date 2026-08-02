# Onboarding Web UI

Repository Detective includes a browser-based setup wizard for connecting Gitea repositories without hand-editing webhooks.

## Access

| URL | Auth |
|-----|------|
| `http://your-host:8080/onboard` | Public (UI only) |
| `http://your-host:8080/` | Redirects to `/onboard` |

API calls from the wizard require the operator API key (`REPOSITORY_DETECTIVE_API_KEY` in `.env`; legacy `REPOSITORY_DETECTIVE_API_KEY` still works). **Preferred** header:

```http
X-Repository-Detective-API-Key: your-api-key
```

Legacy header `X-Repository-Detective-API-Key` is still accepted.

## Wizard steps

1. **Connection settings** — Gitea URL, access token, public URL, webhook secret, API key
2. **AI provider** — provider, base URL, API key, model (test connection before continuing)
3. **Select repositories** — loads repos visible to your token
4. **Register webhooks** — creates push + pull request hooks on selected repos
5. **Environment export** — copy/paste `REPOSITORY_DETECTIVE_*` variables for Docker or systemd deployment

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

Legacy: `REPOSITORY_DETECTIVE_PUBLIC_URL` still works.

The wizard registers hooks at `{public_url}/webhook`.

## API endpoints

All endpoints are under `/api/v1/onboard/` and require the API key header (preferred: `X-Repository-Detective-API-Key`).

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
  -H "X-Repository-Detective-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{"gitea_url":"https://git.example.com","gitea_token":"your-token"}'
```

### Example: register webhooks

```bash
curl -X POST http://localhost:8080/api/v1/onboard/webhooks \
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

## Token permissions

Your Gitea personal access token needs at least:

- `read:repository`
- `write:repository` (webhook registration)
- `write:issue` (if auto-creating issues)

## Related

- [SETUP.md](SETUP.md) — full deployment walkthrough
- [BRANDING_MIGRATION.md](BRANDING_MIGRATION.md) — naming conventions
- [NAMING.md](NAMING.md) — product naming rules
