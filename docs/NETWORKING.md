# Exposing Bugbot on the Network

Gitea at `git.commsnet.org` must reach Bugbot's webhook URL over **HTTPS** (recommended) or HTTP. Bugbot listens on an internal host (e.g. `192.168.255.11:8081`) — you choose how to make that reachable.

**Cloudflare tunnel is optional.** Pick the option that matches your infrastructure.

| Option | Best for | Public URL |
|--------|----------|------------|
| [A. Docker port + firewall/NAT](#a-docker-port-publish--firewallnat-pfsense) | Homelab with pfSense/router | `https://bugbot.yourdomain.com` via port forward |
| [B. Host network mode](#b-docker-host-network-linux) | Simple Linux server, no NAT layers | Host IP + port directly |
| [C. Reverse proxy (nginx/Caddy)](#c-reverse-proxy-on-a-public-host) | Existing web front-end | Your domain with TLS |
| [D. Shared Docker proxy network](#d-shared-docker-proxy-network-traefiknginx) | Traefik / nginx-proxy stack | Auto via proxy labels |
| [E. Cloudflare tunnel](#e-cloudflare-tunnel-no-open-ports) | No open inbound ports | `*.trycloudflare.com` or custom hostname |

After exposure, set:

```bash
BUGBOT_PUBLIC_URL=https://your-public-url
```

Webhook URL: `{BUGBOT_PUBLIC_URL}/webhook`

---

## A. Docker port publish + firewall/NAT (pfSense)

Default homelab approach — publish a host port, then forward WAN → host in pfSense.

### 1. Run Bugbot with public port binding

```bash
cp .env.clustermgr.example .env   # edit with your tokens first
docker compose -f docker-compose.public.yml up -d --build
```

This binds **`0.0.0.0:8081`** on the host (all interfaces, LAN-reachable).

Verify on clustermgr:

```bash
curl http://192.168.255.11:8081/health
ss -tlnp | grep 8081
```

### 2. pfSense port forward

**Firewall → NAT → Port Forward → Add**

| Field | Value |
|-------|-------|
| Interface | WAN |
| Protocol | TCP |
| Destination port | `8081` (or `443` if terminating TLS on pfSense) |
| Redirect target IP | `192.168.255.11` |
| Redirect target port | `8081` |
| Description | Bugbot webhooks |

**Firewall → Rules → WAN** — allow the matching rule pfSense creates.

### 3. DNS + TLS (recommended)

Point `bugbot.commsnet.org` A record → your public IP.

Options for HTTPS:

- **pfSense HAProxy / ACME** — terminate TLS on pfSense, backend `192.168.255.11:8081`
- **Caddy/nginx on clustermgr** — see [Option C](#c-reverse-proxy-on-a-public-host)

Set:

```bash
BUGBOT_PUBLIC_URL=https://bugbot.commsnet.org
```

### 4. Test from outside your LAN

```bash
curl https://bugbot.commsnet.org/health
```

Then test webhook delivery in Gitea (repo → Settings → Webhooks → Test).

---

## B. Docker host network (Linux)

Container shares the host network stack — no port mapping layer. Bugbot listens directly on host `:8081`.

```bash
docker compose -f docker-compose.host-network.yml up -d --build
```

Requires Linux (`network_mode: host` does not work the same on Docker Desktop for Windows/Mac).

Set in `.env`:

```bash
BUGBOT_PORT=8081
BUGBOT_LISTEN_HOST=0.0.0.0
```

Then use pfSense NAT (Option A step 2) or expose the host IP directly on a routable network.

---

## C. Reverse proxy on a public host

Run nginx or Caddy on a machine with a public IP (or behind pfSense with 443 forwarded).

### Caddy example

```
bugbot.commsnet.org {
    reverse_proxy 192.168.255.11:8081
}
```

### nginx example

See `deploy/nginx-bugbot.conf.example`.

```nginx
server {
    listen 443 ssl;
    server_name bugbot.commsnet.org;

    location / {
        proxy_pass http://192.168.255.11:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Set `BUGBOT_PUBLIC_URL=https://bugbot.commsnet.org`.

---

## D. Shared Docker proxy network (Traefik/nginx)

If you already run Traefik or nginx-proxy on a shared Docker network, attach Bugbot to it.

### Traefik (docker-compose.proxy-network.yml)

1. Ensure Traefik network exists: `docker network create traefik-public` (or your network name)
2. Edit compose file — set your network name and domain labels
3. Run:

```bash
docker compose -f docker-compose.proxy-network.yml up -d --build
```

Traefik routes `bugbot.example.com` → Bugbot container. Set `BUGBOT_PUBLIC_URL` to that hostname.

### nginx-proxy / jwilder

Join the `nginx-proxy` external network and set:

```yaml
environment:
  VIRTUAL_HOST: bugbot.commsnet.org
  VIRTUAL_PORT: 8080
  LETSENCRYPT_HOST: bugbot.commsnet.org
```

---

## E. Cloudflare tunnel (no open ports)

Use when you **cannot** or **will not** open inbound ports on pfSense.

See [TUNNEL.md](TUNNEL.md) for quick and named tunnel setup.

```bash
cloudflared tunnel --url http://127.0.0.1:8081
```

---

## Docker Compose files

| File | Purpose |
|------|---------|
| `docker-compose.minimal.yml` | Basic local/dev, port 8080 |
| `docker-compose.clustermgr.yml` | Clustermgr, port 8081 |
| `docker-compose.public.yml` | Explicit `0.0.0.0:8081` for LAN + NAT |
| `docker-compose.host-network.yml` | Linux host networking |
| `docker-compose.proxy-network.yml` | Attach to Traefik external network |

---

## Security checklist

- Set a strong `BUGBOT_WEBHOOK_SECRET` — Gitea sends this with each webhook
- Set `BUGBOT_API_KEY` — protects `/api/v1/*` and onboarding API
- Prefer **HTTPS** for public webhook URLs
- Restrict WAN firewall rule to Gitea's IP if Gitea egress IP is stable
- Use repo include/exclude patterns to limit which repos trigger analysis

---

## Troubleshooting

| Symptom | Check |
|---------|-------|
| Works on LAN, not from internet | pfSense NAT + WAN firewall rule |
| Connection refused on host | `ss -tlnp \| grep 8081`, container running |
| Gitea webhook timeout | Public URL must reach Bugbot — test with `curl` from outside |
| Wrong URL in webhooks | `BUGBOT_PUBLIC_URL` must match what Gitea calls |

See also [TROUBLESHOOTING.md](TROUBLESHOOTING.md).
