<p align="center">
  <img src="ui/static/logo.svg" alt="Repository Detective logo" width="140">
</p>

<h1 align="center">Repository Detective</h1>

<p align="center">
  <strong>Inspect. Analyze. Improve.</strong><br>
  Gitea-first repository assessment, issue lifecycle, and evidence-based remediation.
</p>

---

## Community private beta

| | |
|---|---|
| **Edition** | Community private beta — single-operator homelab |
| **Forge** | Gitea-first (GitHub manual/bulk scans optional) |
| **UI auth** | API-key mode by default; optional local login (`auth_mode=local`) |
| **Not yet** | SaaS, multi-tenant, billing, auto-merge, third-party auto-submit |
| **Editions docs** | [Community](docs/COMMUNITY_EDITION.md) · [Commercial](docs/COMMERCIAL_ENTERPRISE.md) · [Editions overview](docs/EDITIONS.md) |

> **Naming:** The product is **Repository Detective**. Prefer `REPOSITORY_DETECTIVE_*` env vars and `X-Repository-Detective-API-Key`. Silent legacy aliases remain for existing deployments — see [docs/NAMING.md](docs/NAMING.md).

Repo: https://git.commsnet.org/commstech/repository-detective.git

**Beta feedback:** use [Gitea issue templates](https://git.commsnet.org/commstech/repository-detective/issues/new) (`.gitea/ISSUE_TEMPLATE/`) — include scan ID and finding fingerprint; never paste secrets.

## Setup

**Start here:** [docs/SETUP.md](docs/SETUP.md) — step-by-step from clone to working webhooks.

**Operator docs:** [docs/README.md](docs/README.md) · [Dashboard](docs/DASHBOARD_GUIDE.md) · [Auth (local)](docs/AUTH_LOCAL.md) · [Privacy](docs/PRIVACY_AND_DATA_PROTECTION.md)

Quick local trial:

```bash
git clone https://git.commsnet.org/commstech/repository-detective.git && cd repository-detective
docker compose -f docker-compose.minimal.yml up -d --build
curl http://localhost:8080/health
```

Then open http://localhost:8080/onboard

## What it does

- Scans changed files on push; scans PR diff files on pull requests
- Runs **deterministic checks first**: static rules, [Trivy](https://github.com/aquasecurity/trivy), [Grype](https://github.com/anchore/grype), golangci-lint, ruff, shellcheck
- Uses LLM analysis only on flagged files (or disable entirely with `REPOSITORY_DETECTIVE_ENABLE_LLM_AUDITORS=false`)
- Creates Gitea issues with severity, file, line, code snippet, and PoC when available
- Optional LLM backends — **off by default** in beta (`enable_llm_auditors: false`)
- Remediation planner yes; **remediation PRs off by default**. Issue dedup is fingerprint + forge mapping (SQLite)
- **No auto-merge** and **no automatic third-party issue submission**
- **AI recommendations** (optional, off by default) — provider-neutral advisory layer with CAH gating; see [docs/AI_RECOMMENDATIONS.md](docs/AI_RECOMMENDATIONS.md)
- **Issue providers:** Gitea supported; GitHub code path exists but RC-unproven; GitLab not implemented — [docs/ISSUE_PROVIDERS.md](docs/ISSUE_PROVIDERS.md)
- **Marketing:** not ready — see [docs/release/RC_ACCEPTANCE_BASELINE.md](docs/release/RC_ACCEPTANCE_BASELINE.md)

## Go module proxy (supply chain)

Recommended for builds and CI:

```bash
GOPROXY=https://proxy.golang.org,direct
GOSUMDB=sum.golang.org
```

Enterprise: use your internal artifact proxy. Offline: `go mod vendor` then `GOPROXY=off`. See [docs/SECURITY_HARDENING.md](docs/SECURITY_HARDENING.md).

## Configuration

Environment variables prefer the `REPOSITORY_DETECTIVE_` prefix. Legacy `REPOSITORY_DETECTIVE_*` variables remain supported.

| Setting | Preferred variable | Legacy alias |
|---------|-------------------|--------------|
| HTTP port | `REPOSITORY_DETECTIVE_PORT` | `BUGBOT_PORT` |
| API key | `REPOSITORY_DETECTIVE_API_KEY` | `BUGBOT_API_KEY` |
| Public URL for webhooks | `REPOSITORY_DETECTIVE_PUBLIC_URL` | `BUGBOT_PUBLIC_URL` |
| Gitea | `REPOSITORY_DETECTIVE_GITEA_URL`, `REPOSITORY_DETECTIVE_GITEA_TOKEN` | `BUGBOT_GITEA_*` |
| Webhook secret | `REPOSITORY_DETECTIVE_WEBHOOK_SECRET` | `BUGBOT_WEBHOOK_SECRET` |
| Local auth | `REPOSITORY_DETECTIVE_AUTH_MODE`, `REPOSITORY_DETECTIVE_SESSION_SECRET` | `REPOSITORY_DETECTIVE_*` |

Full reference: [docs/CONFIGURATION.md](docs/CONFIGURATION.md)

## HTTP endpoints

| Path | Auth | Notes |
|------|------|-------|
| `GET /health` | none | Orchestrator probe |
| `GET /onboard` | none | Setup wizard |
| `POST /webhook` | HMAC (`X-Gitea-Signature`) | Gitea calls this |
| `/api/v1/*` | API key (preferred or legacy header) | Automation |
| `/ui/*` | API key (default) or session (`auth_mode=local`) | Operator UI |

**Preferred** API key header:

```http
X-Repository-Detective-API-Key: your-key
```

Legacy `X-Repository-Detective-API-Key` still accepted. See [docs/API_ROUTES.md](docs/API_ROUTES.md).

## Documentation

See [docs/README.md](docs/README.md) for the full index.

## License

See repository license file. Edition strategy: [docs/EDITIONS.md](docs/EDITIONS.md).
