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
docker compose up -d --build
curl -m 5 http://127.0.0.1:8081/health
```

Expose port 8081 to Gitea — [docs/NETWORKING.md](docs/NETWORKING.md).

## Required env vars

```
REPOSITORY_DETECTIVE_API_KEY
REPOSITORY_DETECTIVE_GITEA_URL
REPOSITORY_DETECTIVE_GITEA_TOKEN
REPOSITORY_DETECTIVE_WEBHOOK_SECRET
REPOSITORY_DETECTIVE_AI_PROVIDER
REPOSITORY_DETECTIVE_AI_MODEL
REPOSITORY_DETECTIVE_PUBLIC_URL          # after Repository-Detective is reachable from Gitea
```

## Problems

[docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)

## Testing

[docs/TESTING.md](docs/TESTING.md)
