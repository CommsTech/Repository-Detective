# Quickstart — zero to first scan

**Repository Detective** — Inspect. Analyze. Improve.

Get a useful finding on screen after a clean install. Full path: [TEN_MINUTE_AHA.md](TEN_MINUTE_AHA.md).

---

## 1. Clone

```bash
git clone https://github.com/CommsTech/Repository-Detective.git
# Canonical Gitea: https://git.commsnet.org/commstech/Repository-Detective.git
cd Repository-Detective
```

## 2. Configure environment

```bash
cp .env.example .env
# optional YAML overlay:
# cp config/config.yaml.example config/config.yaml
```

Edit `.env` — **minimum required:**

```bash
# openssl rand -hex 32
REPOSITORY_DETECTIVE_API_KEY=<generated>
REPOSITORY_DETECTIVE_AUTH_MODE=local
REPOSITORY_DETECTIVE_SESSION_SECRET=<generated-32+-chars>
REPOSITORY_DETECTIVE_PUBLIC_URL=http://127.0.0.1:8081
REPOSITORY_DETECTIVE_GITEA_URL=https://git.your-domain.example
REPOSITORY_DETECTIVE_GITEA_TOKEN=<personal-access-token>
REPOSITORY_DETECTIVE_WEBHOOK_SECRET=<generated>
REPOSITORY_DETECTIVE_SKIP_STARTUP_CHECKS=false
# Keep report-only until you trust findings:
REPOSITORY_DETECTIVE_AUTO_CREATE_ISSUES=false
```

Startup **rejects** published example placeholders (for example `change-me-to-a-secure-random-string` or `https://git.example.com`).

**Auth:** use `/ui/bootstrap` then `/ui/login` for humans. Use `X-Repository-Detective-API-Key` for API/automation. Do **not** put API keys in browser URLs (`?api_key=` is rejected by default).

## Optional AI

Leave `REPOSITORY_DETECTIVE_ENABLE_LLM_AUDITORS=false` for deterministic-only operation. To enable privacy-preserving AI later, configure a local Ollama / OpenAI-compatible endpoint — see [AI_PROVIDERS.md](AI_PROVIDERS.md).

## 3. Start (published image)

**Recommended** — pull the published all-in-one image (avoids a long local scanner toolchain build):

```bash
export RD_IMAGE=git.commsnet.org/commstech/repository-detective:v0.1.0-beta.6
docker compose pull
docker compose up -d
```

Optional public mirror: `ghcr.io/commstech/repository-detective:v0.1.0-beta.6` (when published).

**Build from source** only when developing or the registry is unavailable:

```bash
docker compose up -d --build
```

Default Compose listens on **127.0.0.1:8081** (loopback only).

## 4. Confirm readiness

```bash
curl -s http://127.0.0.1:8081/health/live    # process up
curl -s http://127.0.0.1:8081/health/ready   # forge/components ready (expect HTTP 200)
export REPOSITORY_DETECTIVE_API_KEY='your-key-from-env'
./scripts/operator-smoke-test.sh
```

Do not treat `/health` alone as “fully configured” — prefer `/health/ready`.

## 5. Open UI

| URL | Purpose |
|-----|---------|
| `http://127.0.0.1:8081/ui/bootstrap` | Create first operator (localhost / private network) |
| `http://127.0.0.1:8081/ui/login` | Operator session login |
| `http://127.0.0.1:8081/onboard` | Setup wizard |

## 6. Add Gitea token in wizard

- Test Gitea connection
- Set **public URL** when Gitea must reach webhooks from outside your LAN — [NETWORKING.md](NETWORKING.md)

## 7. Configure webhook

Wizard registers push + pull_request hooks on selected repos.

Manual: repo → Settings → Webhooks → `{PUBLIC_URL}/webhook`, secret = `REPOSITORY_DETECTIVE_WEBHOOK_SECRET`.

## 8. Run first scan (report-only)

**API:**

```bash
curl -s -X POST http://127.0.0.1:8081/api/v1/analyze \
  -H "X-Repository-Detective-API-Key: $REPOSITORY_DETECTIVE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"owner":"your-org","repository":"your-repo","ref":"main"}'
```

**UI:** Repos → your repo → Scan.

No Gitea issues are filed until you set `REPOSITORY_DETECTIVE_AUTO_CREATE_ISSUES=true`.

## 9. Read results

- `/ui` → Findings (severity, file, scanner)
- Dashboard calibration summary
- Enable issue creation only after you review volume/quality

## 10. Common failures

| Symptom | Doc |
|---------|-----|
| Startup: published example placeholder | Generate real `API_KEY` / `SESSION_SECRET` / forge URL |
| 401 on API | [TROUBLESHOOTING.md](TROUBLESHOOTING.md#api-key-authentication) |
| Webhook 401 | Secret mismatch |
| No scanners | Use published all-in-one image — [DOCKER.md](DOCKER.md) |
| `/health/ready` 503 | Forge token/URL incomplete or startup checks still running |

---

## Beta-safe defaults

See [BETA_READINESS.md](BETA_READINESS.md). Recommended: `scan_profile: standard`, `AUTO_CREATE_ISSUES=false`, remediation PRs off.

---

## Next steps

| Doc | When |
|-----|------|
| [SETUP.md](SETUP.md) | Full deployment walkthrough |
| [CONFIGURATION.md](CONFIGURATION.md) | All config keys |
| [BETA_SMOKE_TEST.md](BETA_SMOKE_TEST.md) | Full validation checklist |
| [BACKUP_RESTORE.md](BACKUP_RESTORE.md) | Before production use |
