# Private beta tester guide

Repository Detective — **Inspect. Analyze. Improve.**

This guide is for private beta testers. The product scans connected Git repositories, surfaces findings in a dashboard, and can optionally file issues — but **issue filing is off by default** in this beta.

## What Repository Detective does

- Connects to Gitea (or GitHub) via API token and webhooks
- Runs deterministic security, quality, health, and code-graph scanners
- Persists findings with severity, confidence, and scanner transparency
- Closes issues when evidence shows a finding is fixed (evidence closure)
- Recommends per-repo calibration rules from scan history (learning engine)
- Produces executive reports and printable/PDF views per repository

## Private beta safety defaults

| Feature | Default |
|---------|---------|
| Issue filing | **Off** (`auto_create_issues: false`) |
| First scan mode | **Report-only** (`report_only_dry_run: true`) |
| Remediation PRs | Off |
| Runner delegation | Off |
| Notifications | Off |
| LLM sanity gate | Off |
| LLM auditors | Off |
| Evidence closure | On |
| Backlog control | On |
| Global auto-calibration | Blocked |

**Do not enable issue filing until your operator explicitly approves it.**

## Supported platforms

| Platform | Method |
|----------|--------|
| Linux amd64 | Binary or Docker (recommended) |
| Linux arm64 | Docker all-in-one image |
| macOS | Docker (binary build not shipped in beta bundle) |

Prerequisites: Docker 24+ (recommended) or glibc Linux for the binary; network access to your forge; 2 GB RAM minimum; writable `./data` directory.

## Install prerequisites

1. Gitea (or GitHub) account with API token scoped to your test repo
2. Random API key for Repository Detective UI/API (32+ chars)
3. Optional: webhook secret if using push-triggered scans

## Docker quickstart (recommended)

```bash
# Unpack beta bundle
cd repository-detective-beta
cp .env.example .env
cp config.example.yaml ../config/config.yaml   # or copy into ./config/

# Edit .env — set REPOSITORY_DETECTIVE_API_KEY, GITEA_URL, GITEA_TOKEN, WEBHOOK_SECRET
# Never commit .env

mkdir -p ../data ../config
docker compose -f docker-compose.beta.yml up -d --build
```

Open: `http://127.0.0.1:8081/ui`

## Binary quickstart

```bash
cd repository-detective-beta
cp config.example.yaml config/config.yaml
export REPOSITORY_DETECTIVE_API_KEY='your-secure-key'
export REPOSITORY_DETECTIVE_GITEA_URL='https://git.example.com'
export REPOSITORY_DETECTIVE_GITEA_TOKEN='your-token'
mkdir -p data
./repository-detective
```

Verify checksum: `sha256sum -c checksums.txt`

## Gitea API token setup

1. Gitea → Settings → Applications → Generate Token
2. Scopes: `read:repository`, `write:repository` (write only if issue filing approved later), `read:user`
3. Set `REPOSITORY_DETECTIVE_GITEA_TOKEN` in `.env` — never in git

## Webhook setup

1. Repository → Settings → Webhooks → Add Webhook
2. URL: `{PUBLIC_URL}/webhook/gitea` (e.g. `https://detective.example.com/webhook/gitea`)
3. Secret: match `REPOSITORY_DETECTIVE_WEBHOOK_SECRET`
4. Events: Push (and Pull Request if you want PR scans)

For local testing without inbound HTTPS, trigger scans via API instead of webhooks.

## First scan — report-only mode

Connect one test repo, then run a dry-run scan (no issues created):

```bash
curl -X POST "http://127.0.0.1:8081/api/v1/analyze" \
  -H "X-Repository-Detective-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "owner": "your-org",
    "repo": "your-test-repo",
    "report_only_dry_run": true
  }'
```

Poll status: `GET /api/v1/scans/{scan_id}`

## View the dashboard

- Dashboard: `/ui`
- Repository detail: `/ui/repos/{owner}/{repo}`
- Findings list with filters
- Risk map / code graph (when enabled)
- Learning health: `/ui/learning`
- Configure repo: `/ui/configure/{owner}/{repo}`

## Configure features

Use **Configure** per repository:

- Scan profile (`beta_standard` recommended)
- Enable/disable individual scanners
- Severity and confidence gates
- Issue policy (leave **off** until approved)
- Workspace mode and analysis depth

Global defaults live in `config/config.yaml` and `.env`.

## Enable or disable scanners

Per repo on Configure page, or globally in config:

```yaml
enable_trivy: true
enable_grype: true
enable_gitleaks: true
enable_staticcheck: true
# ... see config.example.yaml
```

Missing scanners are reported in `/health` and skipped gracefully.

## Pre-install audit

Navigate to `/ui/preinstall` to assess a third-party repo before adoption. Uses conservative scanners and does not file issues.

## Export reports / PDF

1. Open repository → **Executive report**
2. Use browser Print → Save as PDF
3. API: `GET /api/v1/repos/{owner}/{repo}/report` (authenticated)

## What not to enable yet

- Issue filing (`auto_create_issues` / issue policy **all**)
- Remediation PRs
- All-repo scan (`POST /api/v1/analyze/all`)
- Runner delegation to external CI
- LLM sanity gate or mandatory LLM auditors
- Global calibration auto-accept
- Notifications to Slack/Telegram until configured safely

## Default tester workflow

1. Install (Docker or binary)
2. Connect **one** test repo
3. Run **report-only** scan
4. Review executive report
5. Review findings and scanner transparency
6. Review learning/calibration recommendations at `/ui/learning`
7. **Do not** enable issue filing until approved
8. Submit feedback (see below)

## Provide feedback

Use [PRIVATE_BETA_FEEDBACK_TEMPLATE.md](PRIVATE_BETA_FEEDBACK_TEMPLATE.md). Send to your operator or file in the designated feedback channel.

## Collect logs safely

```bash
docker logs repository-detective-beta --tail 200 2>&1 | \
  sed -E 's/(api_key|token|secret|password)=[^ ]+/REDACTED/gi'
```

Do **not** share `.env`, database files, or raw forge tokens.

## Uninstall

```bash
docker compose -f docker-compose.beta.yml down
rm -rf data/   # optional — removes scan history
```

Remove webhooks from test repos.

## Avoid committing secrets

- Keep secrets in `.env` only (gitignored)
- Use `config.example.yaml` as template — empty token fields
- Never commit `data/repository-detective.db`, `.env`, or the `repository-detective` binary
- Rotate tokens if accidentally exposed

## Troubleshooting

| Symptom | Check |
|---------|-------|
| `/health` unhealthy | DB permissions on `./data`, logs |
| 401 on API | `REPOSITORY_DETECTIVE_API_KEY` header |
| No findings | Scanner availability in `/health` |
| Scan timeout | Increase `analysis_timeout` in config |
| Graph empty | `enable_code_graph: true`, repo language support |

Operator details: [PRIVATE_BETA_OPERATOR_RUNBOOK.md](PRIVATE_BETA_OPERATOR_RUNBOOK.md)
