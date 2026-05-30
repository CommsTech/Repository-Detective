# Offline deploy (build on .11, run on .10)

Clustermgr (`192.168.255.10`) cannot reach the Go module proxy during `docker build`. Build the image on the ai host (`192.168.255.11`), transfer it, and start without building on `.10`.

## Host roles

| Host | IP | Role |
|------|-----|------|
| clustermgr | `192.168.255.10` | Run Bugbot (port 8081). Legacy `docker-compose` 1.23.2. No Go. |
| ai | `192.168.255.11` | Build Docker image, export tar. Clean up after `.10` is stable. |

Repo on clustermgr: `/home/commstech/bugbot` (commit `a52646e` or later).

## Step 1 — Build and export on `.11`

```bash
ssh commstech@192.168.255.11
cd ~/bugbot   # or wherever the repo lives on .11
git pull      # if needed
chmod +x scripts/build-and-export-image.sh
./scripts/build-and-export-image.sh
```

Produces `gitea-bugbot-image.tar` in the repo root (~30–80 MB).

## Step 2 — Copy tar to `.10`

From `.11`:

```bash
scp gitea-bugbot-image.tar commstech@192.168.255.10:/home/commstech/bugbot/
```

Or from your workstation if you have the tar locally.

## Step 3 — Load and start on `.10`

```bash
ssh commstech@192.168.255.10
cd /home/commstech/bugbot
chmod +x scripts/import-and-start.sh
./scripts/import-and-start.sh
```

This runs:

```bash
docker load -i gitea-bugbot-image.tar
docker-compose -f docker-compose.clustermgr.offline.yml up -d
```

Confirm:

```bash
curl -m 5 http://127.0.0.1:8081/health
docker logs gitea-bugbot --tail 30
```

## `.env` on clustermgr

Must exist before `import-and-start.sh`. Required variables:

```
BUGBOT_API_KEY=...
BUGBOT_GITEA_URL=https://git.commsnet.org
BUGBOT_GITEA_TOKEN=...
BUGBOT_WEBHOOK_SECRET=...
BUGBOT_AI_PROVIDER=openclaw
BUGBOT_AI_BASE_URL=http://192.168.255.11:...   # reachable from .10
BUGBOT_AI_API_KEY=...
BUGBOT_AI_MODEL=...
BUGBOT_SKIP_STARTUP_CHECKS=true
BUGBOT_PUBLIC_URL=                             # set after networking step
```

`BUGBOT_AI_BASE_URL` must be an address **clustermgr can reach** (usually the ai host on the LAN, not localhost from `.10`'s perspective).

## Do not build on `.10`

These will fail without outbound module proxy access:

```bash
docker-compose -f docker-compose.clustermgr.yml up --build   # avoid
docker build .                                               # avoid
```

Use `docker-compose.clustermgr.offline.yml` instead (image only, no `build:` key).

## After `.10` is stable — clean up `.11`

On `.11`, remove stray Bugbot deploy files if you copied the repo there only for building:

```bash
# example — adjust paths to what you actually created on .11
cd ~/bugbot
docker-compose down 2>/dev/null || true
# keep or remove ~/bugbot as you prefer; no Bugbot service should listen on .11
ss -tlnp | grep 8081   # should show nothing
```

## Updating Bugbot later

Repeat the cycle: build on `.11` → `scp` tar → `import-and-start.sh` on `.10`.

Or fix outbound access on `.10` and use `docker-compose.clustermgr.yml` with `--build` locally.

## See also

- [SETUP.md](SETUP.md) — full setup including webhooks
- [NETWORKING.md](NETWORKING.md) — expose `.10:8081` to Gitea
