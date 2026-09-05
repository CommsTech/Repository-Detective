# Setup Guide

Follow these steps in order for the **Recommended Installation** (Docker Compose, port **8081**, AI optional).

Repository: https://git.commsnet.org/commstech/Repository-Detective.git  
Public mirror: https://github.com/CommsTech/Repository-Detective.git

**Privacy:** Clone gives you source and install docs only. Your `.env`, `config/config.yaml`, and SQLite database stay on the machine that runs Repository Detective and are gitignored — they are never part of the shared Gitea tree. See [PRIVACY_AND_DATA_PROTECTION.md](PRIVACY_AND_DATA_PROTECTION.md).

---

## Step 1 — Clone and configure

```bash
git clone https://git.commsnet.org/commstech/Repository-Detective.git
# or: git clone https://github.com/CommsTech/Repository-Detective.git
cd Repository-Detective
cp .env.example .env
```

Edit `.env`.

### Required core configuration

```bash
REPOSITORY_DETECTIVE_API_KEY=generate-a-long-random-string
REPOSITORY_DETECTIVE_GITEA_URL=https://git.example.com
REPOSITORY_DETECTIVE_GITEA_TOKEN=your-gitea-token
REPOSITORY_DETECTIVE_WEBHOOK_SECRET=another-random-string
```

Leave `REPOSITORY_DETECTIVE_PUBLIC_URL` empty until Step 4.

### Optional AI configuration

AI is **not required**. Default operation is deterministic-only (`REPOSITORY_DETECTIVE_ENABLE_LLM_AUDITORS=false`).

When you want optional AI analysis, prefer a **local** OpenAI-compatible endpoint (Ollama / OpenWebUI):

```bash
# REPOSITORY_DETECTIVE_ENABLE_LLM_AUDITORS=true
# REPOSITORY_DETECTIVE_AI_PROVIDER=ollama
# REPOSITORY_DETECTIVE_AI_BASE_URL=http://10.x.x.x:11434
# REPOSITORY_DETECTIVE_AI_MODEL=qwen2.5-coder
# REPOSITORY_DETECTIVE_AI_API_KEY=          # only if the provider needs one
```

External cloud providers are supported when explicitly configured — see [AI_PROVIDERS.md](AI_PROVIDERS.md).

---

## Step 2 — Start Repository Detective

**Recommended — pull published image (port 8081):**

```bash
docker compose pull
docker compose up -d
```

**Build from source** (same port; longer):

```bash
docker compose up -d --build
```

Older installs with standalone `docker-compose` (no plugin): use `docker-compose` instead of `docker compose`.

Logs:

```bash
docker logs repository-detective --tail 50
```

---

## Step 3 — Confirm it runs

```bash
curl -m 5 http://127.0.0.1:8081/health
```

Expect `"status":"starting"` briefly, then `"status":"healthy"`. AI may report as disabled — that is normal when LLM auditors are off.

If startup fails or hangs, see [TROUBLESHOOTING.md](TROUBLESHOOTING.md). Often helps:

```bash
REPOSITORY_DETECTIVE_SKIP_STARTUP_CHECKS=true   # in .env
```

---

## Step 4 — Expose Repository Detective to Gitea

If Gitea runs on the public internet and Repository Detective is on a private network, Gitea must reach Repository Detective via a public URL. See [NETWORKING.md](NETWORKING.md) for port forwarding, reverse proxy, Traefik, or Cloudflare tunnel.

After exposure:

```bash
REPOSITORY_DETECTIVE_PUBLIC_URL=https://repository-detective.example.com   # in .env
docker compose up -d
curl https://repository-detective.example.com/health
```

---

## Step 5 — Register webhooks

Open `https://repository-detective.example.com/onboard` (or `http://127.0.0.1:8081/onboard` on LAN), enter your API key, test Gitea, select repos, register webhooks.

AI connection test in the wizard is **optional** — skip it for deterministic-only operation.

Manual alternative (per repo → Settings → Webhooks):

- URL: `{REPOSITORY_DETECTIVE_PUBLIC_URL}/webhook`
- Content type: `application/json`
- Secret: same as `REPOSITORY_DETECTIVE_WEBHOOK_SECRET` (Gitea uses this to HMAC-sign the body; Repository Detective checks the `X-Gitea-Signature` header)
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

---

## Advanced Installation Options

| Option | Port | Notes |
|--------|------|-------|
| `docker-compose.minimal.yml` | 8080 | Local contrib build without published image |
| `docker-compose.offline.yml` | 8081 | Preloaded tar / air-gap |
| `docker-compose.beta.yml` | 8081 | Private beta profile |
| Host-network / Traefik overlays | — | [NETWORKING.md](NETWORKING.md) |
| External database | — | [DATABASE.md](DATABASE.md) |
| Runner workers | — | [RUNNER_DELEGATION.md](RUNNER_DELEGATION.md) |

Default `docker-compose.yml` uses **bridge** networking with a port publish to **8081**. Host networking is an optional overlay (`docker-compose.host-network.yml`), not the default.

---

## Related docs

- [QUICKSTART.md](QUICKSTART.md) — shorter recommended path
- [AI providers](AI_PROVIDERS.md) — optional
- [CONFIGURATION.md](CONFIGURATION.md)
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- [PUBLIC_BETA.md](PUBLIC_BETA.md)
- Report install problems: [GitHub installation template](https://github.com/CommsTech/Repository-Detective/issues/new?template=installation_problem.yml)
