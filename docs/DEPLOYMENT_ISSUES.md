# Deployment Issues and Workarounds

Track A deployment notes for Bugbot / Repository Detective. Each item lists what we hit, the workaround applied on this host, and the upstream fix status.

Repository: https://git.commsnet.org/commstech/Bugbot

---

## 1. ERR_TOO_MANY_REDIRECTS on `/` and `/onboard`

**Symptom:** Browser shows `ERR_TOO_MANY_REDIRECTS` at `http://192.168.255.10:8081/`.

**Cause:** Gin `RedirectTrailingSlash` plus `StaticFS` mounted at `/onboard/static` created a loop:
- `/` → 302 `/onboard`
- `/onboard` → 301 `./` (resolves to `/`) → repeat

**Fix (in repo):**
- Serve onboarding HTML via embedded bytes (no `FileFromFS` redirect).
- Register both `/onboard` and `/onboard/`.
- Move static assets to `/onboard/assets` (was `/onboard/static`).
- Root redirect targets `/onboard/` (trailing slash).

**Workaround before fix:** Use `/health` or `/ui?api_key=YOUR_KEY` directly.

**Test:** `handlers/onboarding_test.go` — `TestOnboardingPageNoRedirectLoop`

---

## 2. Operator UI requires API key in browser

**Symptom:** `/ui` returns `401 API key required`.

**Cause:** UI is protected by API key middleware (by design).

**Workaround:** Open `http://HOST:8081/ui?api_key=YOUR_BUGBOT_API_KEY` after setting `BUGBOT_API_KEY` in `.env`.

**Onboarding:** Use `http://HOST:8081/onboard/` (no API key needed for the wizard UI).

---

## 3. Docker build fails when `storage.googleapis.com` is blocked

**Symptom:** `go mod download` fails with `dial tcp 0.0.0.0:443` for `storage.googleapis.com`.

**Cause:** Go module proxy redirects to Google Cloud Storage; some DNS filters sinkhole that domain.

**Fix (in repo):**
- `Dockerfile` supports vendored builds (`-mod=vendor`).
- `./scripts/vendor-deps.sh` retries with `goproxy.io` when `proxy.golang.org` fails.

**Workaround:** Run `./scripts/vendor-deps.sh` before `docker-compose build`, or build on a network without the sinkhole.

**Gitea issue:** #3

---

## 4. Docker bridge IP pool exhaustion

**Symptom:** `could not find an available, non-overlapping IPv4 address pool`.

**Cause:** Many stale `GITEA-ACTIONS-TASK-*` bridge networks.

**Fix (in repo):** `docker-compose.yml` uses `network_mode: host`.

**Workaround:** `docker network prune` periodically.

**Gitea issue:** #7

---

## 5. SQLite data volume permissions

**Symptom:** Container restart loop — `unable to open database file: out of memory (14)`.

**Cause:** Bind mount `./data` owned by host UID 1000; container runs as `bugbot` UID 1001.

**Fix (in repo):** `scripts/docker-entrypoint.sh` runs `chown bugbot:bugbot /app/data` on start.

---

## 6. Ruff install URL 404 in Dockerfile runtime stage

**Symptom:** Docker build fails at `ruff-x86_64-unknown-linux-musl.tar.gz` v0.4.8.

**Cause:** Pinned Ruff release URL removed from GitHub.

**Fix (in repo):** Use `releases/latest/download/ruff-x86_64-unknown-linux-musl.tar.gz` with correct tar path.

---

## 7. OpenClaw / self-signed TLS from container

**Symptom:** `AI provider connection check failed: context deadline exceeded` for `https://192.168.255.11:18789`.

**Cause:** Go HTTP client does not read `REQUESTS_CA_BUNDLE`; custom CA must be in system trust store.

**Fix (in repo):** Mount `./certs/*.crt`; entrypoint runs `update-ca-certificates`.

**Remaining:** Startup check may still timeout if the gateway is slow; set `BUGBOT_SKIP_STARTUP_CHECKS=true`.

**Gitea issue:** #8

---

## 8. Legacy systemd service vs Docker on port 8081

**Symptom:** Native binary and Docker compete for port 8081; systemd restarts native process.

**Workaround:** `sudo systemctl disable --now bugbot.service` after Docker is healthy. Legacy `~/bugbot/run.sh` repointed to `docker-compose up -d`.

**Gitea issue:** #9

---

## Quick reference — URLs on this host

| URL | Purpose |
|-----|---------|
| `http://192.168.255.10:8081/health` | Health check (no auth) |
| `http://192.168.255.10:8081/onboard/` | Setup wizard |
| `http://192.168.255.10:8081/ui?api_key=…` | Operator dashboard |
| `http://192.168.255.10:8081/api/v1/status` | API (header `X-Repository-Detective-API-Key`; legacy `X-Bugbot-API-Key` accepted) |

**Note:** Bugbot listens on port **8081**, not 80. Include `:8081` unless a reverse proxy maps 443/80 → 8081.
