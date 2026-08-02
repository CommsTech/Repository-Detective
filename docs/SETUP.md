# Setup Guide

Follow these steps in order.

Repository: https://git.commsnet.org/commstech/repository-detective.git

---

## Step 1 — Clone and configure

```bash
git clone https://git.commsnet.org/commstech/repository-detective.git
cd repository-detective
cp .env.example .env
```

Edit `.env`. Required values:

```bash
REPOSITORY_DETECTIVE_API_KEY=generate-a-long-random-string
REPOSITORY_DETECTIVE_GITEA_URL=https://git.example.com
REPOSITORY_DETECTIVE_GITEA_TOKEN=your-gitea-token
REPOSITORY_DETECTIVE_WEBHOOK_SECRET=another-random-string
REPOSITORY_DETECTIVE_AI_PROVIDER=openai          # or anthropic, ollama, openwebui, openclaw, etc.
REPOSITORY_DETECTIVE_AI_API_KEY=your-ai-key      # if your provider needs one
REPOSITORY_DETECTIVE_AI_MODEL=gpt-4o-mini
```

Set `REPOSITORY_DETECTIVE_AI_BASE_URL` when the provider has no built-in default, or when the AI service runs on another host (use a hostname/IP reachable from the Repository-Detective container).

Leave `REPOSITORY_DETECTIVE_PUBLIC_URL` empty until Step 4.

---

## Step 2 — Start Repository Detective

**Server / LAN (port 8081):**

```bash
docker compose up -d --build
```

**Local dev (port 8080):**

```bash
docker compose -f docker-compose.minimal.yml up -d --build
```

Older installs with standalone `docker-compose` (no plugin): use `docker-compose` instead of `docker compose` in the commands above.

Logs:

```bash
docker logs repository-detective --tail 50
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
REPOSITORY_DETECTIVE_SKIP_STARTUP_CHECKS=true   # in .env
```

---

## Step 4 — Expose Repository Detective to Gitea

If Gitea runs on the public internet and Repository-Detective is on a private network, Gitea must reach Repository-Detective via a public URL. See [NETWORKING.md](NETWORKING.md) for port forwarding, reverse proxy, Traefik, or Cloudflare tunnel.

After exposure:

```bash
REPOSITORY_DETECTIVE_PUBLIC_URL=https://repository-detective.example.com   # in .env
docker compose up -d
curl https://repository-detective.example.com/health
```

---

## Step 5 — Register webhooks

Open `https://repository-detective.example.com/onboard`, enter your API key, test Gitea and AI, select repos, register webhooks.

Manual alternative (per repo → Settings → Webhooks):

- URL: `{REPOSITORY_DETECTIVE_PUBLIC_URL}/webhook`
- Content type: `application/json`
- Secret: same as `REPOSITORY_DETECTIVE_WEBHOOK_SECRET` (Gitea uses this to HMAC-sign the body; Repository-Detective checks the `X-Gitea-Signature` header)
- Events: Push, Pull request

---

## Step 6 — Verify

1. Test webhook delivery in Gitea (expect HTTP 200).
2. Push a commit to a watched repo.
3. `docker logs repository-detective --tail 100`

Look for deterministic scanner output:

```bash
docker logs repository-detective 2>&1 | grep -E 'SCANNER|CAH:SCAN'
```

4. Confirm scanner binaries in the image (after rebuild):

```bash
docker exec repository-detective sh -c 'trivy --version && grype version && golangci-lint version'
```

5. Run unit tests locally — see [TESTING.md](TESTING.md).

---

## Compose files

| File | Use when |
|------|----------|
| `docker-compose.yml` | **Default** — production deploy, port 8081, host networking |
| `docker-compose.minimal.yml` | Local dev, port 8080, bridge networking |
| `docker-compose.offline.yml` | Pre-built image only (no `docker build` on host) |
| `docker-compose.traefik.yml` | Optional overlay with `docker-compose.yml` for Traefik |

### Deploy without building on the target host

When the runtime host has no outbound access (cannot run `go mod download`):

```bash
# Machine with network
docker compose build
docker save repository-detective:latest -o repository-detective-image.tar

# Target host
docker load -i repository-detective-image.tar
docker compose -f docker-compose.offline.yml up -d
```

---

## Further reading

- [AI providers](AI_PROVIDERS.md)
- [Deterministic scanners](SCANNERS.md)
- [Testing](TESTING.md)
- [Networking](NETWORKING.md)
- [Troubleshooting](TROUBLESHOOTING.md)
