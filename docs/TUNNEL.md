# Exposing Repository Detective to External Gitea

Optional — use when you cannot or will not open inbound firewall ports.

For port forwarding, reverse proxy, and Traefik, see **[NETWORKING.md](NETWORKING.md)**.

## Quick tunnel (testing)

Install [cloudflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/), then:

```bash
cloudflared tunnel --url http://127.0.0.1:8081
```

Use the printed `https://*.trycloudflare.com` URL as `REPOSITORY_DETECTIVE_PUBLIC_URL`.

## Named tunnel (persistent)

```bash
cloudflared tunnel login
cloudflared tunnel create repository-detective
```

`/etc/cloudflared/config.yml`:

```yaml
tunnel: <TUNNEL-UUID>
credentials-file: /root/.cloudflared/<TUNNEL-UUID>.json

ingress:
  - hostname: repository-detective.example.com
    service: http://127.0.0.1:8081
  - service: http_status:404
```

DNS: CNAME `repository-detective.example.com` → `<TUNNEL-UUID>.cfargotunnel.com`

```bash
sudo cloudflared service install
sudo systemctl enable --now cloudflared
```

Set `REPOSITORY_DETECTIVE_PUBLIC_URL=https://repository-detective.example.com`.

## Verify

```bash
curl https://repository-detective.example.com/health
```

Test webhook delivery in Gitea.
