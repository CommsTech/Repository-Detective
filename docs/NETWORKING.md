# Exposing Repository-Detective on the Network

If Gitea is on the public internet and Repository-Detective runs on a private LAN, Gitea needs a URL it can reach for webhooks. Pick one approach below.

Set `REPOSITORY_DETECTIVE_PUBLIC_URL` to that URL. Webhooks go to `{REPOSITORY_DETECTIVE_PUBLIC_URL}/webhook`.

| Option | Best for |
|--------|----------|
| [A. Port publish + firewall NAT](#a-port-publish--firewall-nat) | Homelab router (pfSense, etc.) |
| [B. Host network mode](#b-docker-host-network-linux) | Linux server, direct host binding |
| [C. Reverse proxy](#c-reverse-proxy) | nginx, Caddy, or similar |
| [D. Traefik / shared proxy network](#d-traefik--shared-proxy-network) | Existing Docker reverse proxy |
| [E. Cloudflare tunnel](#e-cloudflare-tunnel) | No inbound ports |

---

## A. Port publish + firewall NAT

### 1. Start Repository Detective on port 8081

```bash
cp .env.example .env
docker compose up -d --build
curl http://127.0.0.1:8081/health
```

From another machine on the LAN: `curl http://<repository-detective-host-ip>:8081/health`

### 2. Forward WAN → Repository-Detective host

On your router/firewall, add a port forward:

| Field | Example |
|-------|---------|
| WAN port | 8081 or 443 |
| Internal IP | Repository-Detective host (e.g. 10.0.0.50) |
| Internal port | 8081 |

### 3. DNS and TLS

Point `repository-detective.example.com` at your public IP. Terminate TLS on the router (HAProxy/ACME) or on a reverse proxy (Option C).

```bash
REPOSITORY_DETECTIVE_PUBLIC_URL=https://repository-detective.example.com
```

### 4. Test externally

```bash
curl https://repository-detective.example.com/health
```

Test webhook delivery in Gitea.

---

## B. Docker host network (Linux)

Default `docker-compose.yml` uses **bridge** networking and publishes port **8081**.

For host networking (when bridge IP pools are exhausted or you need it for LAN reachability), apply the overlay:

```bash
docker compose -f docker-compose.yml -f docker-compose.host-network.yml up -d
```

Set `REPOSITORY_DETECTIVE_PORT=8081` in `.env`.

Not supported the same way on Docker Desktop for Windows/Mac.

---

## C. Reverse proxy

Example nginx upstream (see `deploy/nginx-repository-detective.conf.example`):

```nginx
upstream repository_detective {
    server 10.0.0.50:8081;
}

server {
    listen 443 ssl;
    server_name repository-detective.example.com;
    location / {
        proxy_pass http://repository_detective;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Caddy:

```
repository-detective.example.com {
    reverse_proxy 10.0.0.50:8081
}
```

---

## D. Traefik / shared proxy network

```bash
docker network create traefik-public   # if needed
docker compose -f docker-compose.yml -f docker-compose.traefik.yml up -d --build
```

Edit the `Host(...)` label in `docker-compose.traefik.yml` to match your hostname.

---

## E. Cloudflare tunnel

See [TUNNEL.md](TUNNEL.md). No inbound firewall rules required.

---

## Compose files

| File | Purpose |
|------|---------|
| `docker-compose.yml` | Default production (host network, port 8081) |
| `docker-compose.minimal.yml` | Dev, port 8080 |
| `docker-compose.offline.yml` | Pre-loaded image, no build |
| `docker-compose.traefik.yml` | Optional Traefik overlay |

---

## Security

- Set `REPOSITORY_DETECTIVE_WEBHOOK_SECRET` and use the same value in Gitea webhook config
- Set `REPOSITORY_DETECTIVE_API_KEY` for API and onboarding endpoints
- Prefer HTTPS for public webhook URLs

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) if webhooks fail.
