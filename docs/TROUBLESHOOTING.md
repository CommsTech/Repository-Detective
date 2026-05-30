# Troubleshooting

## Health check hangs or connection refused

**Check the port.** Docker on clustermgr maps host 8081 → container 8080:

```bash
ss -tlnp | grep 8081
curl -m 5 http://127.0.0.1:8081/health
```

**Check the container is up:**

```bash
docker ps | grep bugbot
docker logs gitea-bugbot --tail 50
```

**Common startup failures** (visible in `docker logs`):

| Message | Fix |
|---------|-----|
| `gitea_url is required` | Set `BUGBOT_GITEA_URL` in `.env` |
| `gitea_token is required` | Set `BUGBOT_GITEA_TOKEN` |
| `configure ai_provider` | Set `BUGBOT_AI_PROVIDER` and `BUGBOT_AI_BASE_URL` |
| Gitea/AI connection timeout | Add `BUGBOT_SKIP_STARTUP_CHECKS=true` to `.env`, rebuild |

Run in foreground to watch startup:

```bash
docker compose -f docker-compose.clustermgr.yml up --build
```

**Empty bugbot.log:** Bugbot logs to stdout. Use `docker logs`, or `./scripts/run-clustermgr.sh 2>&1 | tee bugbot.log`.

**503 on /health:** Normal for a few seconds while components initialize. Wait and retry.

---

## Gitea webhooks fail

Gitea cannot reach private IPs. Bugbot needs a public URL.

1. Expose port 8081 — [NETWORKING.md](NETWORKING.md) (pfSense NAT, nginx, or Traefik)
2. Or use a Cloudflare tunnel — [TUNNEL.md](TUNNEL.md)
3. Set `BUGBOT_PUBLIC_URL=https://your-public-url` in `.env` and restart
4. Test from outside your network: `curl https://your-public-url/health`
5. In Gitea, test webhook delivery (repo → Settings → Webhooks → Test)

Webhook URL must be `{BUGBOT_PUBLIC_URL}/webhook`. The secret in Gitea must match `BUGBOT_WEBHOOK_SECRET`.

---

## `cannot unmarshal array into RepositoryContent`

Fixed in commit `c0580f6`. Pull and rebuild:

```bash
git pull
docker compose -f docker-compose.clustermgr.yml up -d --build
```

---

## No issues created after push

- `auto_create_issues` must be true (default)
- Gitea token needs issue write permission
- Check logs for analysis errors: `docker logs gitea-bugbot --tail 100`
- Repo may be filtered out — check `repository_exclude_patterns` in `config/config.yaml`

---

## SSH sessions timing out

Use short commands:

```bash
curl -m 5 http://127.0.0.1:8081/health
docker logs gitea-bugbot --tail 50
```

Run Bugbot in Docker with `restart: unless-stopped` so it keeps running after SSH drops.

---

## Wizard API calls fail

The onboarding API requires the same key as `BUGBOT_API_KEY`. Enter it in the wizard's first step.

If `/onboard` loads but API calls return 401, the API key in the UI does not match the server config.
