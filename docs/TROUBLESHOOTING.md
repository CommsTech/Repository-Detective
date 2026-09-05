# Troubleshooting

**Repository Detective** — Inspect. Analyze. Improve.

Operator-focused fixes for private beta deployments. Prefer `REPOSITORY_DETECTIVE_*` env vars; legacy `REPOSITORY_DETECTIVE_*` still works.

---

## API key authentication

**Symptoms:** `401 Unauthorized`, wizard/API calls fail.

**Fix:**

1. Confirm `.env` has `REPOSITORY_DETECTIVE_API_KEY` (or legacy `REPOSITORY_DETECTIVE_API_KEY`).
2. Send **preferred** header:

   ```bash
   curl -H "X-Repository-Detective-API-Key: $REPOSITORY_DETECTIVE_API_KEY" \
     http://127.0.0.1:8081/api/v1/status
   ```

3. Legacy header `X-Repository-Detective-API-Key` still accepted.
4. Prefer the `X-Repository-Detective-API-Key` header for API calls. Legacy query `?api_key=` is **deprecated** (leaks into logs/history); the UI stores it in an HttpOnly cookie and redirects to a clean URL when used once.
5. Restart container after changing `.env`.

See [CONFIGURATION.md](CONFIGURATION.md), [BRANDING_MIGRATION.md](BRANDING_MIGRATION.md).

---

## Local auth (`auth_mode=local`)

**Symptoms:** Service fails to start; cannot sign in; bootstrap loop.

| Message / symptom | Fix |
|-------------------|-----|
| `auth_mode=local requires session_secret` | Set `REPOSITORY_DETECTIVE_SESSION_SECRET` (32+ random bytes) in `.env` |
| `auth_mode=local requires database_enabled` | Enable SQLite (`database_enabled: true`) |
| Redirect to `/ui/bootstrap` | Expected when no users exist — create owner account |
| `/ui/bootstrap` redirects to login | Users already exist; use `/ui/login` |
| `invalid or missing CSRF token` on form POST | Include hidden `csrf_token` from page; do not strip cookies |
| Locked out | Set `auth_mode: api_key_only`, restart; API key UI works again |
| API scripts fail after enabling local auth | API still uses `X-Repository-Detective-API-Key` — unchanged |

Guide: [AUTH_LOCAL.md](AUTH_LOCAL.md).

---

## Go module download / Docker build fails on proxy

**Symptoms:** `storage.googleapis.com` blocked; `go mod download` or `docker build` fails.

**Recommended:**

```bash
GOPROXY=https://proxy.golang.org,direct
GOSUMDB=sum.golang.org
./scripts/vendor-deps.sh
docker compose up -d --build
```

**Enterprise:** `GOPROXY=https://your-internal-artifact-proxy,direct`

**Offline:** `go mod vendor` then `GOPROXY=off`

**Emergency only:** `vendor-deps.sh` may retry `goproxy.io` — temporary workaround, not for DoD/government deployments. Do **not** use `goproxy.cn`.

See [SECURITY_HARDENING.md](SECURITY_HARDENING.md).

---

## Health check fails

**Wrong port?**

| Compose file | curl |
|--------------|------|
| `docker-compose.yml` / `offline` | `http://127.0.0.1:8081/health` |
| `docker-compose.minimal.yml` | `http://127.0.0.1:8080/health` |

```bash
docker ps | grep repository-detective
ss -tlnp | grep -E '8080|8081'
docker logs repository-detective --tail 50
```

**Config errors** (in `docker logs`):

| Message | Fix |
|---------|-----|
| `gitea_url is required` | `REPOSITORY_DETECTIVE_GITEA_URL` in `.env` |
| `gitea_token is required` | `REPOSITORY_DETECTIVE_GITEA_TOKEN` |
| `configure gitea_token and/or github_token` | At least one forge token |
| `configure ai_provider` | Only if LLM auditors enabled |
| Connection timeout at startup | `REPOSITORY_DETECTIVE_SKIP_STARTUP_CHECKS=true` |

Run in foreground:

```bash
docker compose up --build
```

**503 on /health:** Normal for a few seconds while components initialize.

---

## Slow `/health` (~4 seconds)

**Cause:** Health endpoint probes scanner binary availability.

**Mitigation:** Expected on all-in-one; use for monitoring, not per-request UI polling. Caching improvement is backlog.

**Check:**

```bash
time curl -s http://127.0.0.1:8081/health >/dev/null
```

---

## Gitea webhooks fail

Gitea cannot reach private IPs. Repository Detective needs a **public URL** — [NETWORKING.md](NETWORKING.md).

1. Confirm external access: `curl https://detective.example.com/health`
2. Set `REPOSITORY_DETECTIVE_PUBLIC_URL` in `.env`, restart container
3. Webhook URL: `{PUBLIC_URL}/webhook`
4. Secret in Gitea must match `REPOSITORY_DETECTIVE_WEBHOOK_SECRET`
5. Gitea sends HMAC-SHA256 in `X-Gitea-Signature` — verified automatically

**401 on webhook test:** Secret mismatch or missing signature header.

---

## Gitea token permissions

Token needs:

- Read repository
- Write webhooks (onboarding)
- Write issues (if `auto_create_issues: true`)

**Symptoms:** Empty repo list, webhook registration fails, issues not created.

---

## Scanner missing binary

**Symptoms:**

```text
[SCANNER:trivy] binary not found
tools_summary.missing: ["trivy", ...]
```

**Fix:**

- Use **all-in-one** image (`docker-compose.yml` default)
- Rebuild: `docker compose build repository-detective`
- Verify inside container:

  ```bash
  docker exec repository-detective sh -c \
    'for t in trivy grype gitleaks semgrep govulncheck gosec staticcheck hadolint checkov; do command -v $t || echo MISSING:$t; done'
  ```

See [SCANNERS.md](SCANNERS.md), [DOCKER.md](DOCKER.md).

---

## Gitleaks parse failures

**Symptoms:** Scanner status `parse_failed` despite findings on disk.

**Cause:** gitleaks 8.x ignores `--report-path -` (stdout).

**Fix:** Included in main — writes temp report file. Confirm image has gitleaks **8.21.2+**. Rebuild from current `main`.

---

## Scanner timeouts

**Symptoms:** Scanner status `timeout`, partial results.

**Fix:**

- Increase `scanner_timeout_seconds` or per-scanner timeout in config
- Reduce repo scope / use `beta_standard` profile
- checkov/grype on large monorepos — retry or exclude paths

---

## Database locked / SQLite errors

**Symptoms:** `database is locked`, dashboard empty.

**Fix:**

1. Only one writer — stop duplicate containers binding same `./data`
2. Check permissions: container user must read/write `data/repository-detective.db`
3. Restore from backup if corrupted — [BACKUP_RESTORE.md](BACKUP_RESTORE.md)

**Restore:**

```bash
docker compose stop repository-detective
cp /backups/repository-detective-YYYY-MM-DD.db data/repository-detective.db
docker compose start repository-detective
```

---

## Docker permissions

**Symptoms:** Cannot write DB, config read errors.

**Fix:**

- Ensure `./data` owned or writable by UID **1001** (`repositorydetective`) or run with matching volume permissions
- Config mount is `:ro` — edit on host, not inside container

**Security:** Default compose does **not** mount Docker socket or use `privileged: true`.

---

## Archive / workspace mode failure

**Symptoms:** Scan fails cloning or extracting repo archive.

**Fix:**

- Confirm Gitea token can read repo
- Check disk space under scanner workspace temp dir
- Large repos — adjust timeouts and `max_file_size`

---

## Pre-install private network rejection

**Symptoms:** Pre-install audit rejects URL with private IP / localhost.

**Expected:** SSRF protection — HTTPS public URLs only.

**Override (homelab only):** `preinstall_allow_private_networks: true` — see [PREINSTALL_AUDIT.md](PREINSTALL_AUDIT.md).

---

## Theme not persisting

**Symptoms:** Light/dark resets on navigation.

**Fix:** Included in beta — `ui/static/theme.js` + localStorage. Hard-refresh browser; clear stale cache. Test: `go test ./ui/ -run Theme`.

---

## Legacy Repository-Detective naming confusion

| You see | Meaning |
|---------|---------|
| `REPOSITORY_DETECTIVE_*` in old docs | Use `REPOSITORY_DETECTIVE_*` — both work |
| `X-Repository-Detective-API-Key` | Legacy — prefer `X-Repository-Detective-API-Key` |
| `repository-detective.db` | Database filename — intentional |
| `commstech/Repository-Detective` git repo | Forge repo name — not product name |
| Container `repository-detective` | Old name — current is `repository-detective` |

See [BRANDING_MIGRATION.md](BRANDING_MIGRATION.md), [BRANDING_COMPATIBILITY_AUDIT.md](BRANDING_COMPATIBILITY_AUDIT.md).

---

## Wizard API returns 401

Same as [API key authentication](#api-key-authentication).

---

## Labels not attached to issues

Repository Detective uses Gitea label API. Check token issue-write permission. Missing labels auto-created.

---

## Scans run but no LLM output

Expected when `enable_llm_auditors: false` (beta default) or no flagged files for LLM stage.

---

## Too many false-positive Gitea issues

See [FALSE_POSITIVES.md](FALSE_POSITIVES.md). Raise `min_issue_confidence`, use `beta_standard`, apply suppressions.

---

## AI provider TLS errors

For homelab private CA:

```yaml
ai_insecure_skip_tls_verify: true
```

Prefer installing CA on host when possible.

---

## Cannot build image on target host

```bash
./scripts/vendor-deps.sh   # DNS-filtered networks
docker save / docker load  # transfer pre-built image
```

See [DOCKER.md](DOCKER.md), [DEPLOYMENT.md](DEPLOYMENT.md).

---

## Bulk scan stopped early

Upgrade to latest build; use detached scan context. `./deploy.sh --scan-all-quick`.

---

## `cannot unmarshal array into RepositoryContent`

Pull latest and rebuild.

---

## Disk full — Docker build or verify fails

**Symptoms:**

```text
database or disk is full (13)
health check timed out
Bind for 0.0.0.0:18081 failed: port is already allocated
ERROR: not enough free disk for Docker build verify
```

**Production impact:** Usually **none** if `repository-detective` on port 8081 is still running — failures hit isolated verify containers or new builds.

**Check:**

```bash
df -h /
docker system df -v
```

**Clean (safe order):**

```bash
docker container prune -f
docker builder prune -f
docker image prune -af
```

**Volumes:** Repository Detective homelab uses bind mount `./data` for SQLite — **not** an anonymous Docker volume. Review `docker volume ls` before `docker volume prune`; skip if unsure.

**Stale verify containers:**

```bash
docker ps -aq --filter "name=rd-verify-" | xargs -r docker rm -f
```

**Targets:** 10–20 GB free minimum; **30+ GB** preferred for all-in-one rebuilds.

**Retry:**

```bash
./scripts/docker-build-verify.sh
```

The script runs a disk preflight and removes stale `rd-verify-*` containers before smoke tests.

---

## Getting more help

1. `./scripts/operator-smoke-test.sh`
2. [BETA_SMOKE_TEST.md](BETA_SMOKE_TEST.md)
3. [KNOWN_LIMITATIONS.md](KNOWN_LIMITATIONS.md)
4. [TEST_MATRIX.md](TEST_MATRIX.md)
