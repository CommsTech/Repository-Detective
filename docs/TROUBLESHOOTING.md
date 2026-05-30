# Troubleshooting Clustermgr Deployment

## Symptom: `curl localhost:8081/health` hangs or times out

### Cause A: Server not listening yet (startup checks blocking)

**Before fix:** Bugbot only opened the HTTP port *after* Gitea + AI connection tests (up to 60s). During that window nothing listened on the port.

**Now:** The server binds immediately. `/health` returns `503 {"status":"starting"}` during init, then `200 {"status":"healthy"}` when ready.

**Quick fix for slow/unreachable AI at boot:**

```bash
export BUGBOT_SKIP_STARTUP_CHECKS=true
```

### Cause B: Wrong port

| Deployment | Bugbot listens | You should curl |
|------------|----------------|-----------------|
| Docker `8081:8080` | container :8080 | host `:8081` |
| Native `BUGBOT_PORT=8081` | host :8081 | host `:8081` |
| Default config | :8080 | `:8080` |

Check what's listening:

```bash
ss -tlnp | grep -E '8080|8081'
```

### Cause C: Process exited on config error

Common failures:

- `gitea_url is required`
- `gitea_token is required`
- `configure ai_provider + ai_base_url`

Run in foreground to see errors:

```bash
BUGBOT_SKIP_STARTUP_CHECKS=true ./scripts/run-clustermgr.sh
```

Or with Docker:

```bash
docker compose -f docker-compose.clustermgr.yml up --build
# Ctrl+C to stop; logs print to terminal
```

### Cause D: Empty bugbot.log

Bugbot logs to **stdout**, not `bugbot.log`, unless you redirect:

```bash
./scripts/run-clustermgr.sh 2>&1 | tee bugbot.log
```

Docker logs:

```bash
docker logs gitea-bugbot --tail 100
```

---

## Symptom: `cannot unmarshal array into RepositoryContent`

Gitea returns a JSON **array** for directory listings; the client expected a single object.

**Fixed** in `gitea/decodeRepositoryContents` (commit `c0580f6+`). Pull latest and rebuild:

```bash
git pull
docker compose -f docker-compose.clustermgr.yml up -d --build
```

---

## Symptom: Gitea webhooks fail — can't reach Bugbot

Gitea at `git.commsnet.org` cannot reach internal `192.168.255.11:8081`.

**Option 2 — pfSense port forward (no tunnel):**

WAN TCP `8081` (or `443` via reverse proxy) → `192.168.255.11:8081`.  
Use `docker-compose.public.yml` and set `BUGBOT_PUBLIC_URL=https://bugbot.yourdomain.com`.  
Full steps: [NETWORKING.md](NETWORKING.md).

**Option 3 — Cloudflare quick tunnel:**

```bash
./scripts/install-cloudflared.sh
export PATH="$HOME/bin:$PATH"
cloudflared tunnel --url http://127.0.0.1:8081
```

Set `BUGBOT_PUBLIC_URL` to the `https://*.trycloudflare.com` URL. Webhook URL: `{PUBLIC_URL}/webhook`.

**Option 2 — pfSense port forward** from WAN to `192.168.255.11:8081`.

**Option 4 — pfSense / reverse proxy:** See [NETWORKING.md](NETWORKING.md) — no cloudflared required.

See [TUNNEL.md](TUNNEL.md) for persistent Cloudflare tunnel setup.

---

## Symptom: SSH sessions die after 15–30s

Use short commands:

```bash
# Good
curl -m 5 http://127.0.0.1:8081/health
docker logs gitea-bugbot --tail 50
ss -tlnp | grep 8081

# Avoid on flaky SSH
tail -f bugbot.log
docker compose logs -f
```

Run Bugbot in Docker with `restart: unless-stopped` so it survives SSH drops.

---

## Recommended clustermgr stack

```bash
git pull
cp .env.clustermgr.example .env
# edit .env

docker compose -f docker-compose.clustermgr.yml up -d --build
curl -m 5 http://127.0.0.1:8081/health

# In another session — tunnel
./scripts/install-cloudflared.sh
cloudflared tunnel --url http://127.0.0.1:8081
```

Update `.env` with the tunnel URL as `BUGBOT_PUBLIC_URL`, restart Bugbot, then register webhooks at `/onboard`.
