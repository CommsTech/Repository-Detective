# Quick Setup

For the full production walkthrough, use **[docs/SETUP.md](docs/SETUP.md)**.

## Local dev (Windows)

```powershell
cd C:\Users\commstech\Github\Gitea_AI_Bugbot
docker compose -f docker-compose.minimal.yml up -d --build
curl http://localhost:8080/health
```

Open http://localhost:8080/onboard

Or run without Docker:

```powershell
.\scripts\run-dev.ps1
```

## Clustermgr (Linux)

```bash
git pull
cp .env.clustermgr.example .env   # edit with your tokens
docker compose -f docker-compose.clustermgr.yml up -d --build
curl -m 5 http://127.0.0.1:8081/health
```

Then expose port 8081 to the internet — see [docs/NETWORKING.md](docs/NETWORKING.md).

## Required env vars

```
BUGBOT_API_KEY
BUGBOT_GITEA_URL
BUGBOT_GITEA_TOKEN
BUGBOT_WEBHOOK_SECRET
BUGBOT_AI_PROVIDER
BUGBOT_AI_BASE_URL    # if not using a provider default
BUGBOT_AI_API_KEY     # if provider requires it
BUGBOT_AI_MODEL
BUGBOT_PUBLIC_URL     # set after Bugbot is reachable from Gitea
```

## Verify

```bash
curl http://localhost:8080/health                              # or :8081 on clustermgr
curl -H "X-Bugbot-API-Key: your-key" http://localhost:8080/api/v1/status
docker logs gitea-bugbot --tail 50
```

## Problems?

[docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)
