# Exposing Bugbot to External Gitea

When Bugbot runs on an internal IP (e.g. `192.168.255.11:8081`) and Gitea is public (`git.commsnet.org`), Gitea cannot reach the webhook URL directly. Use a tunnel or reverse proxy.

## Option A: Cloudflare Tunnel (quick test)

On **clustermgr** (or any host that can reach Bugbot):

```bash
# Install cloudflared (Debian/Ubuntu)
curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg | sudo tee /usr/share/keyrings/cloudflare-main.gpg >/dev/null
echo "deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/cloudflared.list
sudo apt update && sudo apt install -y cloudflared

# Quick tunnel — prints a public *.trycloudflare.com URL
cloudflared tunnel --url http://192.168.255.11:8081
```

Copy the HTTPS URL (e.g. `https://random-name.trycloudflare.com`) and set:

```bash
BUGBOT_PUBLIC_URL=https://random-name.trycloudflare.com
```

Register webhooks at `{PUBLIC_URL}/webhook` via the onboarding UI or Gitea settings.

## Option B: Named Cloudflare Tunnel (persistent)

```bash
cloudflared tunnel login
cloudflared tunnel create bugbot
```

Create `/etc/cloudflared/config.yml`:

```yaml
tunnel: <TUNNEL-UUID>
credentials-file: /root/.cloudflared/<TUNNEL-UUID>.json

ingress:
  - hostname: bugbot.commsnet.org
    service: http://192.168.255.11:8081
  - service: http_status:404
```

Add DNS CNAME `bugbot` → `<TUNNEL-UUID>.cfargotunnel.com`, then:

```bash
sudo cloudflared service install
sudo systemctl enable --now cloudflared
```

Set `BUGBOT_PUBLIC_URL=https://bugbot.commsnet.org`.

## Option C: Reverse proxy on a public host

If you have nginx/caddy on a public server with access to the internal network:

```nginx
location / {
    proxy_pass http://192.168.255.11:8081;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

## Verify connectivity

From any external machine:

```bash
curl https://your-public-url/health
# {"status":"healthy","service":"gitea-bugbot",...}
```

From Gitea’s perspective, the webhook delivery test (repo → Settings → Webhooks → Test) should return `200`.

## Docker port mapping

If Bugbot runs in Docker on clustermgr:

```yaml
ports:
  - "8081:8080"
```

Tunnel target: `http://127.0.0.1:8081` or `http://192.168.255.11:8081`.

## Script

See `scripts/setup-cloudflared-tunnel.sh` for an automated quick-tunnel launcher.
