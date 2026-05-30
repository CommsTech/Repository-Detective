# Quick Setup Guide

## Fastest path: Onboarding wizard

1. **Start Bugbot**

```powershell
cd C:\Users\commstech\Github\Gitea_AI_Bugbot

# Docker (recommended)
docker compose -f docker-compose.minimal.yml up -d --build

# Or local dev
$env:BUGBOT_SKIP_STARTUP_CHECKS="true"
$env:BUGBOT_GITEA_URL="https://git.commsnet.org"
$env:BUGBOT_GITEA_TOKEN="your-token"
$env:BUGBOT_API_KEY="your-api-key"
$env:BUGBOT_PUBLIC_URL="http://localhost:8080"
go run .
```

2. **Open the wizard**

```
http://localhost:8080/onboard
```

3. **Complete the steps**
   - Enter your Bugbot API key and Gitea token
   - Test Gitea and AI connections
   - Load repositories → select repos → register webhooks
   - Copy the generated environment block for production

See [docs/ONBOARDING.md](docs/ONBOARDING.md) for API details.

## Docker deployment

```powershell
# Edit docker-compose.minimal.yml with your values, then:
docker compose -f docker-compose.minimal.yml up -d --build

# Verify
curl http://localhost:8080/health
```

### Key environment variables

```yaml
BUGBOT_PORT=8080
BUGBOT_API_KEY=your-secure-api-key
BUGBOT_PUBLIC_URL=https://bugbot.yourdomain.com
BUGBOT_GITEA_URL=https://git.commsnet.org
BUGBOT_GITEA_TOKEN=your-gitea-token
BUGBOT_WEBHOOK_SECRET=your-webhook-secret
BUGBOT_AI_PROVIDER=openai
BUGBOT_AI_API_KEY=your-ai-key
BUGBOT_AI_MODEL=gpt-4o-mini
BUGBOT_ENABLE_SECURITY=true
BUGBOT_ENABLE_QUALITY=true
```

## Manual webhook setup

If not using the wizard:

1. Repository → Settings → Webhooks → Add webhook
2. URL: `{BUGBOT_PUBLIC_URL}/webhook`
3. Content type: `application/json`
4. Secret: same as `BUGBOT_WEBHOOK_SECRET`
5. Events: **Push**, **Pull request**

## Verify it works

```powershell
# Health
curl http://localhost:8080/health

# Status (requires API key)
curl -H "X-Bugbot-API-Key: your-key" http://localhost:8080/api/v1/status

# Logs
docker compose -f docker-compose.minimal.yml logs -f gitea-bugbot
```

## What happens after setup

1. **Push code** → Bugbot analyzes changed files only
2. **Open/update PR** → Bugbot analyzes diff files only
3. **Issues created** with labels, file/line references, and PoC when available

## Troubleshooting

| Problem | Check |
|---------|-------|
| Wizard API calls fail | `BUGBOT_API_KEY` set and entered in UI |
| Webhooks not firing | `BUGBOT_PUBLIC_URL` reachable from Gitea |
| No issues created | `auto_create_issues: true`, token has issue write access |
| AI errors | Provider config — see [docs/AI_PROVIDERS.md](docs/AI_PROVIDERS.md) |
