# Setup Guide

Follow these steps in order. Skip nothing until Bugbot answers on `/health`.

Repository: https://git.commsnet.org/commstech/Bugbot.git

---

## Step 1 — Clone and create your env file

```bash
git clone https://git.commsnet.org/commstech/Bugbot.git
cd Bugbot
cp .env.clustermgr.example .env
```

Edit `.env`. At minimum set:

```bash
BUGBOT_API_KEY=generate-a-long-random-string
BUGBOT_GITEA_URL=https://git.commsnet.org
BUGBOT_GITEA_TOKEN=your-gitea-token
BUGBOT_WEBHOOK_SECRET=another-random-string
BUGBOT_AI_PROVIDER=openclaw          # or openai, ollama, openwebui, etc.
BUGBOT_AI_BASE_URL=http://your-ai-host:8080/v1
BUGBOT_AI_API_KEY=your-ai-key
BUGBOT_AI_MODEL=your-model
```

Leave `BUGBOT_PUBLIC_URL` blank for now. You set it in Step 4 after Bugbot is reachable from the internet.

---

## Step 2 — Start Bugbot

**On clustermgr (port 8081):**

```bash
docker compose -f docker-compose.clustermgr.yml up -d --build
```

**Local dev (port 8080):**

```bash
docker compose -f docker-compose.minimal.yml up -d --build
```

Logs go to stdout:

```bash
docker logs gitea-bugbot --tail 50
```

---

## Step 3 — Confirm it is running

```bash
# clustermgr
curl -m 5 http://127.0.0.1:8081/health

# local dev
curl -m 5 http://127.0.0.1:8080/health
```

You should see `"status":"starting"` for a few seconds, then `"status":"healthy"`.

If the command hangs or refuses connection, see [TROUBLESHOOTING.md](TROUBLESHOOTING.md). Common fix:

```bash
# add to .env
BUGBOT_SKIP_STARTUP_CHECKS=true
docker compose -f docker-compose.clustermgr.yml up -d --build
```

---

## Step 4 — Expose Bugbot so Gitea can reach it

Gitea at `git.commsnet.org` cannot call a private IP like `192.168.255.11`. Pick one method:

| Method | Guide |
|--------|-------|
| pfSense port forward (recommended for homelab) | [NETWORKING.md § A](NETWORKING.md#a-docker-port-publish--firewallnat-pfsense) |
| nginx or Caddy reverse proxy | [NETWORKING.md § C](NETWORKING.md#c-reverse-proxy-on-a-public-host) |
| Traefik docker network | [NETWORKING.md § D](NETWORKING.md#d-shared-docker-proxy-network-traefiknginx) |
| Cloudflare tunnel (no open ports) | [TUNNEL.md](TUNNEL.md) |

After exposure, confirm from **outside your LAN**:

```bash
curl https://bugbot.yourdomain.com/health
```

Set the URL in `.env`:

```bash
BUGBOT_PUBLIC_URL=https://bugbot.yourdomain.com
docker compose -f docker-compose.clustermgr.yml up -d
```

---

## Step 5 — Register webhooks

Open the onboarding wizard:

```
https://bugbot.yourdomain.com/onboard
```

1. Enter your `BUGBOT_API_KEY` (same value as in `.env`).
2. Enter your Gitea token and test the connection.
3. Test your AI provider.
4. Load repositories, select the ones you want, click **Register webhooks**.

The wizard creates hooks at `{BUGBOT_PUBLIC_URL}/webhook` for push and pull request events.

**Manual alternative:** In each Gitea repo → Settings → Webhooks → Add:

- URL: `https://bugbot.yourdomain.com/webhook`
- Content type: `application/json`
- Secret: same string as `BUGBOT_WEBHOOK_SECRET`
- Events: Push, Pull request

---

## Step 6 — Test end to end

1. In Gitea, open the webhook you created and click **Test delivery**. Expect HTTP 200.
2. Push a commit to a watched repo.
3. Check Bugbot logs: `docker logs gitea-bugbot --tail 100`
4. If findings exist and `auto_create_issues` is enabled, check the repo Issues tab.

---

## Which compose file?

| File | Use when |
|------|----------|
| `docker-compose.clustermgr.yml` | Production on clustermgr, host port 8081 |
| `docker-compose.public.yml` | Same as above, explicit `0.0.0.0:8081` bind |
| `docker-compose.minimal.yml` | Local dev, port 8080 |
| `docker-compose.host-network.yml` | Linux only, no port mapping |
| `docker-compose.proxy-network.yml` | You already run Traefik |

Do **not** use `docker-compose.yml` or `docker-compose.simple.yml` — they contain outdated environment variable names.

---

## Further reading

- [AI provider config](AI_PROVIDERS.md)
- [Troubleshooting](TROUBLESHOOTING.md)
- [CAH pipeline — what is implemented vs planned](CAH_PIPELINE.md)
