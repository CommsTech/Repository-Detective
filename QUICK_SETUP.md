# Quick Setup

Full guide: **[docs/SETUP.md](docs/SETUP.md)**

## Local dev

```bash
cp .env.example .env   # edit as needed
docker compose -f docker-compose.minimal.yml up -d --build
curl http://localhost:8080/health
```

Open http://localhost:8080/onboard

## Server (port 8081)

```bash
cp .env.example .env
docker compose -f docker-compose.public.yml up -d --build
curl -m 5 http://127.0.0.1:8081/health
```

Expose port 8081 to Gitea — [docs/NETWORKING.md](docs/NETWORKING.md).

## Required env vars

```
BUGBOT_API_KEY
BUGBOT_GITEA_URL
BUGBOT_GITEA_TOKEN
BUGBOT_WEBHOOK_SECRET
BUGBOT_AI_PROVIDER
BUGBOT_AI_MODEL
BUGBOT_PUBLIC_URL          # after Bugbot is reachable from Gitea
```

## Problems

[docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)

## Testing

[docs/TESTING.md](docs/TESTING.md)
