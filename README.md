# Repository Detective

**Inspect. Analyze. Improve.**

Automated code review and repository assessment for **Gitea** (primary). Repository Detective watches pushes and pull requests, runs security and quality checks, and opens forge issues when configured.

**Private beta scope:** single-operator, API-key auth, SQLite, deterministic-first scanning. **Not** multi-user SaaS. See [docs/BETA_READINESS.md](docs/BETA_READINESS.md) and [docs/FEATURE_COMPLETENESS_AUDIT.md](docs/FEATURE_COMPLETENESS_AUDIT.md).

> **Naming:** [Repository Detective](docs/NAMING.md) is the product name. **Bugbot** legacy env vars (`BUGBOT_*`), labels (`bugbot/*`), and fingerprints (`bugbot-<hex>`) remain supported. Prefer `REPOSITORY_DETECTIVE_*` for new deployments — see [docs/BRANDING_MIGRATION.md](docs/BRANDING_MIGRATION.md).

Repo: https://git.commsnet.org/commstech/Bugbot.git

## Setup

**Start here:** [docs/SETUP.md](docs/SETUP.md) — step-by-step from clone to working webhooks.

**Operator docs:** [docs/README.md](docs/README.md) · [Dashboard](docs/DASHBOARD_GUIDE.md) · [Scanner health](docs/SCANNER_HEALTH.md) · [Privacy](docs/PRIVACY_AND_DATA_PROTECTION.md)

Quick local trial:

```bash
git clone https://git.commsnet.org/commstech/Bugbot.git && cd Bugbot
docker compose -f docker-compose.minimal.yml up -d --build
curl http://localhost:8080/health
```

Then open http://localhost:8080/onboard

## What it does

- Scans changed files on push; scans PR diff files on pull requests
- Runs **deterministic checks first**: static rules, [Trivy](https://github.com/aquasecurity/trivy), [Grype](https://github.com/anchore/grype), golangci-lint, ruff, shellcheck
- Uses LLM analysis only on flagged files (or disable entirely with `REPOSITORY_DETECTIVE_ENABLE_LLM_AUDITORS=false`; legacy `BUGBOT_ENABLE_LLM_AUDITORS` still works)
- Creates Gitea issues with severity, file, line, code snippet, and PoC when available
- Optional LLM backends (OpenAI, Anthropic, OpenRouter, Ollama, OpenWebUI, OpenClaw) — **off by default** in beta (`enable_llm_auditors: false`)
- Optional GitHub manual/bulk scans when token configured — **not** full webhook/PR parity ([docs/GITHUB_SCANNING.md](docs/GITHUB_SCANNING.md))
- Remediation planner yes; **remediation PRs off by default**. Qdrant semantic dedup **off by default**

## Configuration

Environment variables prefer the `REPOSITORY_DETECTIVE_` prefix. Legacy `BUGBOT_*` variables remain supported.

| Setting | Preferred variable | Legacy alias |
|---------|-------------------|--------------|
| HTTP port | `REPOSITORY_DETECTIVE_PORT` | `BUGBOT_PORT` |
| Bind address | `REPOSITORY_DETECTIVE_LISTEN_HOST` | `BUGBOT_LISTEN_HOST` |
| API key | `REPOSITORY_DETECTIVE_API_KEY` | `BUGBOT_API_KEY` |
| Public URL for webhooks | `REPOSITORY_DETECTIVE_PUBLIC_URL` | `BUGBOT_PUBLIC_URL` |
| Gitea | `REPOSITORY_DETECTIVE_GITEA_URL`, `REPOSITORY_DETECTIVE_GITEA_TOKEN` | `BUGBOT_GITEA_*` |
| Webhook secret | `REPOSITORY_DETECTIVE_WEBHOOK_SECRET` | `BUGBOT_WEBHOOK_SECRET` |
| AI | `REPOSITORY_DETECTIVE_AI_*` | `BUGBOT_AI_*` |
| Deterministic scanners | `REPOSITORY_DETECTIVE_ENABLE_TRIVY`, etc. | `BUGBOT_ENABLE_*` |
| LLM auditors | `REPOSITORY_DETECTIVE_ENABLE_LLM_AUDITORS` | `BUGBOT_ENABLE_LLM_AUDITORS` |
| Label compat mode | `REPOSITORY_DETECTIVE_LABEL_COMPAT_MODE` | `BUGBOT_LABEL_COMPAT_MODE` |
| Skip Gitea/AI ping on boot | `REPOSITORY_DETECTIVE_SKIP_STARTUP_CHECKS=true` | `BUGBOT_SKIP_STARTUP_CHECKS` |

Repo include/exclude patterns and skip patterns are set in `config/config.yaml` only (not env vars).

Full AI provider examples: [docs/AI_PROVIDERS.md](docs/AI_PROVIDERS.md)

## HTTP endpoints

| Path | Auth | Notes |
|------|------|-------|
| `GET /health` | none | Returns `503 starting` then `200 healthy` |
| `GET /onboard` | none | Setup wizard |
| `POST /webhook` | webhook secret | Gitea calls this |
| `POST /api/v1/analyze` | API key | Manual scan trigger |
| `GET /api/v1/status` | API key | Runtime info |
| `POST /api/v1/onboard/*` | API key | Wizard backend |

**Preferred** API key header:

```http
X-Repository-Detective-API-Key: your-key
```

Legacy header `X-Bugbot-API-Key` is still accepted. See [docs/BRANDING_MIGRATION.md](docs/BRANDING_MIGRATION.md).

## Documentation

| Doc | Contents |
|-----|----------|
| [docs/SETUP.md](docs/SETUP.md) | Full setup, step by step |
| [docs/NETWORKING.md](docs/NETWORKING.md) | pfSense, reverse proxy, Traefik |
| [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) | Common failures |
| [docs/ONBOARDING.md](docs/ONBOARDING.md) | Wizard and API details |
| [docs/AI_PROVIDERS.md](docs/AI_PROVIDERS.md) | AI backend config |
| [docs/CAH_PIPELINE.md](docs/CAH_PIPELINE.md) | Analysis pipeline spec |
| [docs/SCANNERS.md](docs/SCANNERS.md) | Trivy, Grype, linters (deterministic) |
| [docs/RUBRICS.md](docs/RUBRICS.md) | Security, pipeline, release, and optimization rubrics |
| [docs/QDRANT.md](docs/QDRANT.md) | Semantic dedup via existing Qdrant server |
| [docs/TESTING.md](docs/TESTING.md) | Unit tests, Docker smoke test, E2E |
| [docs/TUNNEL.md](docs/TUNNEL.md) | Cloudflare tunnel (optional) |
| [docs/EDITIONS.md](docs/EDITIONS.md) | Community / Commercial / Enterprise |
| [docs/LICENSING_STRATEGY.md](docs/LICENSING_STRATEGY.md) | Proposed licensing model |
| [docs/AUTH_RBAC_PLAN.md](docs/AUTH_RBAC_PLAN.md) | Multi-user auth design (not implemented) |
| [docs/BETA_READINESS.md](docs/BETA_READINESS.md) | Private beta checklist |
| [docs/FEATURE_COMPLETENESS_AUDIT.md](docs/FEATURE_COMPLETENESS_AUDIT.md) | Feature inventory and honest claims |
| [docs/API_ROUTES.md](docs/API_ROUTES.md) | API route reference |

## Development

```bash
go build -o gitea-bugbot .
go test ./...
```

See [docs/TESTING.md](docs/TESTING.md) for CI parity checks, Docker smoke tests, and scanner verification.

## License

**Planning:** Community Edition under [AGPL-3.0-or-later](docs/LICENSING_STRATEGY.md) (proposed); Commercial/Enterprise under a separate paid license. **Not yet finalized** — see [docs/LICENSING_STRATEGY.md](docs/LICENSING_STRATEGY.md). Current tree may still reference MIT until legal review.
