# Quickstart — zero to first scan

**Repository Detective** — Inspect. Analyze. Improve.

Get a single-operator homelab running in ~15 minutes. Generic placeholders only.

---

## 1. Clone

```bash
git clone https://github.com/CommsTech/Repository-Detective.git
# or canonical: https://git.commsnet.org/commstech/Repository-Detective.git
cd Repository-Detective
```

## 2. Configure environment

```bash
cp .env.example .env
cp config/config.yaml.example config/config.yaml
```

Edit `.env` — **minimum required:**

```bash
REPOSITORY_DETECTIVE_API_KEY=generate-a-long-random-string
REPOSITORY_DETECTIVE_GITEA_URL=https://git.example.com
REPOSITORY_DETECTIVE_GITEA_TOKEN=your-gitea-personal-access-token
REPOSITORY_DETECTIVE_WEBHOOK_SECRET=another-random-string
REPOSITORY_DETECTIVE_SKIP_STARTUP_CHECKS=false
```

**Note:** Docker Compose loads `.env` via `env_file`, and `REPOSITORY_DETECTIVE_*` values **override** `config/config.yaml`. The example files are aligned so a fresh copy enables the full scanner fleet (gitleaks, semgrep, Go tools) with LLM auditors **off** (AI optional).

**Note:** Prefer the API key **header** or local session login. Do not put API keys in browser URLs. New installs should set `REPOSITORY_DETECTIVE_REJECT_QUERY_STRING_API_KEY=true`.

## Optional AI

Leave `REPOSITORY_DETECTIVE_ENABLE_LLM_AUDITORS=false` for deterministic-only operation. To enable privacy-preserving AI later, configure a local Ollama / OpenAI-compatible endpoint — see [AI_PROVIDERS.md](AI_PROVIDERS.md).

## 3. Start (all-in-one Docker)

**Recommended — pull a published image** (minutes, not a ~40 minute build):

```bash
docker login git.commsnet.org   # Gitea user + token with package read
# Optional pin: export RD_IMAGE=git.commsnet.org/commstech/repository-detective:v0.1.0-beta.3
docker compose pull
docker compose up -d
```

Images publish to **Gitea Package Registry** on version tags (`v*`). GitHub Container Registry is an optional public mirror.

| Tag | Use |
|-----|-----|
| `git.commsnet.org/commstech/repository-detective:all-in-one` | Homelab default (canonical) |
| `git.commsnet.org/commstech/repository-detective:vX.Y.Z` | Pin a release |
| `ghcr.io/commstech/repository-detective:all-in-one` | Public mirror (after sync) |

**Build from source** only when developing or the registry is unavailable:

```bash
docker compose build
docker compose up -d
```

Production compose listens on **port 8081**.

## 4. Confirm health

```bash
curl -s http://127.0.0.1:8081/health | jq .
export REPOSITORY_DETECTIVE_API_KEY='your-key-from-env'
./scripts/operator-smoke-test.sh
```

## 5. Open UI

| URL | Purpose |
|-----|---------|
| `http://127.0.0.1:8081/onboard` | Setup wizard |
| `http://127.0.0.1:8081/ui?api_key=YOUR_KEY` | Dashboard (query key homelab-only; prefer header for API) |

## 6. Add Gitea token in wizard

- Test Gitea connection
- Set **public URL** when Gitea must reach webhooks from outside your LAN — [NETWORKING.md](NETWORKING.md)

## 7. Configure webhook

Wizard registers push + pull_request hooks on selected repos.

Manual: repo → Settings → Webhooks → `{PUBLIC_URL}/webhook`, secret = `REPOSITORY_DETECTIVE_WEBHOOK_SECRET`.

## 8. Run first scan

**API:**

```bash
curl -s -X POST http://127.0.0.1:8081/api/v1/analyze \
  -H "X-Repository-Detective-API-Key: $REPOSITORY_DETECTIVE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"owner":"your-org","repository":"your-repo","ref":"main"}'
```

**UI:** Repos → your repo → Scan.

## 9. Read results

- `/ui` → Findings (severity, file, scanner)
- Gitea issues (if `auto_create_issues: true` and profile allows)
- Dashboard calibration summary

## 10. Common failures

| Symptom | Doc |
|---------|-----|
| 401 on API | [TROUBLESHOOTING.md](TROUBLESHOOTING.md#api-key-authentication) |
| Webhook 401 | Secret mismatch |
| No scanners | Rebuild all-in-one image — [DOCKER.md](DOCKER.md) |
| Container won't start | Missing Gitea token in `.env` |

---

## Beta-safe defaults

See [BETA_READINESS.md](BETA_READINESS.md). Recommended: `scan_profile: standard`, remediation PRs off.

---

## Next steps

| Doc | When |
|-----|------|
| [SETUP.md](SETUP.md) | Full deployment walkthrough |
| [CONFIGURATION.md](CONFIGURATION.md) | All config keys |
| [BETA_SMOKE_TEST.md](BETA_SMOKE_TEST.md) | Full validation checklist |
| [BACKUP_RESTORE.md](BACKUP_RESTORE.md) | Before production use |
