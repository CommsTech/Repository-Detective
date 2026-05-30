# Exposing Bugbot to External Gitea

When Bugbot runs on an internal IP (e.g. `192.168.255.11:8081`) and Gitea is public (`git.commsnet.org`), Gitea cannot reach the webhook URL directly.

**Cloudflare tunnel is one option — not the only one.**

For pfSense port forwards, Docker host networking, nginx/Caddy, and Traefik, see **[NETWORKING.md](NETWORKING.md)**.

---

## Option A: Cloudflare Tunnel (quick test)

No inbound firewall ports required.

```bash
./scripts/install-cloudflared.sh
cloudflared tunnel --url http://127.0.0.1:8081
```

Copy the HTTPS URL and set `BUGBOT_PUBLIC_URL`.

## Option B: Named Cloudflare Tunnel (persistent)

```yaml
ingress:
  - hostname: bugbot.commsnet.org
    service: http://192.168.255.11:8081
  - service: http_status:404
```

Set `BUGBOT_PUBLIC_URL=https://bugbot.commsnet.org`.

## Verify connectivity

```bash
curl https://your-public-url/health
```

Test webhook delivery in Gitea (repo → Settings → Webhooks → Test).

## Script

`scripts/setup-cloudflared-tunnel.sh` — quick tunnel launcher.

## Other exposure methods

| Method | Compose file |
|--------|--------------|
| LAN port + pfSense NAT | `docker-compose.public.yml` |
| Host network (Linux) | `docker-compose.host-network.yml` |
| Traefik proxy network | `docker-compose.proxy-network.yml` |

Full guide: [NETWORKING.md](NETWORKING.md)
