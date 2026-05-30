# Troubleshooting

## Health check fails

**Wrong port?**

| Compose file | curl |
|--------------|------|
| `docker-compose.public.yml` / `offline` | `http://127.0.0.1:8081/health` |
| `docker-compose.minimal.yml` | `http://127.0.0.1:8080/health` |

```bash
docker ps | grep bugbot
ss -tlnp | grep -E '8080|8081'
docker logs gitea-bugbot --tail 50
```

**Config errors** (in `docker logs`):

| Message | Fix |
|---------|-----|
| `gitea_url is required` | `BUGBOT_GITEA_URL` in `.env` |
| `gitea_token is required` | `BUGBOT_GITEA_TOKEN` |
| `configure ai_provider` | `BUGBOT_AI_PROVIDER` + `BUGBOT_AI_BASE_URL` if needed |
| Connection timeout at startup | `BUGBOT_SKIP_STARTUP_CHECKS=true` |

Run in foreground:

```bash
docker compose -f docker-compose.public.yml up --build
```

**503 on /health:** Normal for a few seconds while components initialize.

---

## Gitea webhooks fail

Gitea cannot reach private IPs. Bugbot needs a public URL — [NETWORKING.md](NETWORKING.md).

1. Confirm external access: `curl https://bugbot.example.com/health`
2. Set `BUGBOT_PUBLIC_URL` in `.env`, restart container
3. Webhook URL: `{PUBLIC_URL}/webhook`
4. Secret in Gitea must match `BUGBOT_WEBHOOK_SECRET`
5. Test delivery in Gitea webhook settings

---

## `cannot unmarshal array into RepositoryContent`

Fixed in commit `c0580f6`. Pull latest and rebuild or reload image.

---

## No issues after push

- `BUGBOT_AUTO_CREATE_ISSUES=true`
- Token needs issue write permission
- Check `docker logs gitea-bugbot`
- Repo may match `repository_exclude_patterns` in `config/config.yaml`

---

## Wizard API returns 401

`X-Bugbot-API-Key` must match `BUGBOT_API_KEY` in `.env`.

---

## Cannot build image on target host

Build elsewhere, transfer image:

```bash
docker build -t gitea-bugbot:latest .
docker save gitea-bugbot:latest -o gitea-bugbot-image.tar
# copy to target
docker load -i gitea-bugbot-image.tar
docker compose -f docker-compose.offline.yml up -d
```
