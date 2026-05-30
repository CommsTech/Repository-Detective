# Setup Guide

Follow these steps in order.

Repository: https://git.commsnet.org/commstech/Bugbot.git

---

## Step 1 — Clone and configure

```bash
git clone https://git.commsnet.org/commstech/Bugbot.git
cd Bugbot
cp .env.example .env
```

Edit `.env`. Required values:

```bash
BUGBOT_API_KEY=generate-a-long-random-string
BUGBOT_GITEA_URL=https://git.example.com
BUGBOT_GITEA_TOKEN=your-gitea-token
BUGBOT_WEBHOOK_SECRET=another-random-string
BUGBOT_AI_PROVIDER=openai          # or anthropic, ollama, openwebui, openclaw, etc.
BUGBOT_AI_API_KEY=your-ai-key      # if your provider needs one
BUGBOT_AI_MODEL=gpt-4o-mini
```

Set `BUGBOT_AI_BASE_URL` when the provider has no built-in default, or when the AI service runs on another host (use a hostname/IP reachable from the Bugbot container).

Leave `BUGBOT_PUBLIC_URL` empty until Step 4.

---

## Step 2 — Start Bugbot

**Server / LAN (port 8081):**

```bash
docker compose -f docker-compose.public.yml up -d --build
```

**Local dev (port 8080):**

```bash
docker compose -f docker-compose.minimal.yml up -d --build
```

Older installs with standalone `docker-compose` (no plugin): use `docker-compose` instead of `docker compose` in the commands above.

Logs:

```bash
docker logs gitea-bugbot --tail 50
```

---

## Step 3 — Confirm it runs

```bash
curl -m 5 http://127.0.0.1:8081/health    # public compose
curl -m 5 http://127.0.0.1:8080/health    # minimal compose
```

Expect `"status":"starting"` briefly, then `"status":"healthy"`.

If startup fails or hangs, see [TROUBLESHOOTING.md](TROUBLESHOOTING.md). Often helps:

```bash
BUGBOT_SKIP_STARTUP_CHECKS=true   # in .env
```

---

## Step 4 — Expose Bugbot to Gitea

If Gitea runs on the public internet and Bugbot is on a private network, Gitea must reach Bugbot via a public URL. See [NETWORKING.md](NETWORKING.md) for port forwarding, reverse proxy, Traefik, or Cloudflare tunnel.

After exposure:

```bash
BUGBOT_PUBLIC_URL=https://bugbot.example.com   # in .env
docker compose -f docker-compose.public.yml up -d
curl https://bugbot.example.com/health
```

---

## Step 5 — Register webhooks

Open `https://bugbot.example.com/onboard`, enter your API key, test Gitea and AI, select repos, register webhooks.

Manual alternative (per repo → Settings → Webhooks):

- URL: `{BUGBOT_PUBLIC_URL}/webhook`
- Content type: `application/json`
- Secret: same as `BUGBOT_WEBHOOK_SECRET`
- Events: Push, Pull request

---

## Step 6 — Verify

1. Test webhook delivery in Gitea (expect HTTP 200).
2. Push a commit to a watched repo.
3. `docker logs gitea-bugbot --tail 100`

---

## Compose files

| File | Use when |
|------|----------|
| `docker-compose.minimal.yml` | Local dev, port 8080 |
| `docker-compose.public.yml` | Server deploy, port 8081, builds image locally |
| `docker-compose.offline.yml` | Pre-built image only (no `docker build` on host) |
| `docker-compose.host-network.yml` | Linux, `network_mode: host` |
| `docker-compose.proxy-network.yml` | Existing Traefik network |

Do not use `docker-compose.yml` or `docker-compose.simple.yml` — outdated env var names.

### Deploy without building on the target host

When the runtime host has no outbound access (cannot run `go mod download`):

```bash
# Machine with network
docker build -t gitea-bugbot:latest .
docker save gitea-bugbot:latest -o gitea-bugbot-image.tar

# Target host
docker load -i gitea-bugbot-image.tar
docker compose -f docker-compose.offline.yml up -d
```

---

## Further reading

- [AI providers](AI_PROVIDERS.md)
- [Networking](NETWORKING.md)
- [Troubleshooting](TROUBLESHOOTING.md)
