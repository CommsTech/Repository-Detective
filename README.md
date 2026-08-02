<p align="center">
  <img src="ui/static/logo.svg" alt="Repository Detective logo" width="140">
</p>

<h1 align="center">Repository Detective</h1>

<p align="center">
  <strong>Inspect. Analyze. Improve.</strong><br>
  Gitea-first repository assessment, issue lifecycle, and evidence-based remediation.
</p>

<p align="center">
  <a href="https://git.commsnet.org/commstech/repository-detective/actions?workflow=ci.yml&amp;actor=0&amp;status=0">
    <img src="https://git.commsnet.org/commstech/repository-detective/actions/workflows/ci.yml/badge.svg?branch=main" alt="CI Status">
  </a>
  <a href="https://pkg.go.dev/git.commsnet.org/commstech/repository-detective">
    <img src="https://img.shields.io/badge/go-1.25-00ADD8?style=flat&logo=go&logoColor=white" alt="Go 1.25">
  </a>
  <a href="docs/LICENSING_STRATEGY.md">
    <img src="https://img.shields.io/badge/license-AGPL--3.0%20(proposed)-blue?style=flat" alt="License: AGPL-3.0 proposed">
  </a>
  <a href="docs/DOCKER.md">
    <img src="https://img.shields.io/badge/platforms-linux%2Famd64-lightgrey?style=flat&logo=linux&logoColor=white" alt="Platforms: linux/amd64">
  </a>
  <a href="docs/DOCKER.md">
    <img src="https://img.shields.io/badge/docker-all--in--one-2496ED?style=flat&logo=docker&logoColor=white" alt="Docker">
  </a>
  <a href="docs/COMMUNITY_EDITION.md">
    <img src="https://img.shields.io/badge/edition-private%20beta-orange?style=flat" alt="Private beta">
  </a>
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

> **Naming:** The product is **Repository Detective**. Use `REPOSITORY_DETECTIVE_*` env vars and `X-Repository-Detective-API-Key`. See [docs/NAMING.md](docs/NAMING.md).

Repo (canonical Gitea): https://git.commsnet.org/commstech/Repository-Detective.git  
GitHub mirror (public release only): https://github.com/CommsTech/Repository-Detective.git — keep Gitea current; push GitHub when public: [docs/GITHUB_MIRROR.md](docs/GITHUB_MIRROR.md)

**Beta feedback:** use [Gitea issue templates](https://git.commsnet.org/commstech/Repository-Detective/issues/new) (`.gitea/ISSUE_TEMPLATE/`) — include scan ID and finding fingerprint; never paste secrets.

## Setup

**Start here:** [docs/SETUP.md](docs/SETUP.md) — step-by-step from clone to working webhooks.

**Operator docs:** [docs/README.md](docs/README.md) · [Dashboard](docs/DASHBOARD_GUIDE.md) · [Auth (local)](docs/AUTH_LOCAL.md) · [Privacy](docs/PRIVACY_AND_DATA_PROTECTION.md)

The Gitea tree is a sanitized install base. Operator secrets (`.env`), local config (`config/config.yaml`), and the SQLite database under `data/` are gitignored and must stay private on your host.

Quick local trial (minimal compose uses port **8080**):

```bash
git clone https://git.commsnet.org/commstech/Repository-Detective.git && cd Repository-Detective
docker compose -f docker-compose.minimal.yml up -d --build
curl http://localhost:8080/health
```

Then open http://localhost:8080/onboard

Default `docker-compose.yml` / homelab installs use port **8081** (`http://127.0.0.1:8081`).

**AI agents (OpenClaw, Cursor, etc.):** [docs/AGENT_QUICKSTART.md](docs/AGENT_QUICKSTART.md) · [docs/MCP.md](docs/MCP.md) · [docs/OPENCLAW_INTEGRATION.md](docs/OPENCLAW_INTEGRATION.md) · [docs/openapi.yaml](docs/openapi.yaml)
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

Environment variables use the `REPOSITORY_DETECTIVE_` prefix only.

| Setting | Variable |
|---------|----------|
| HTTP port | `REPOSITORY_DETECTIVE_PORT` |
| API key | `REPOSITORY_DETECTIVE_API_KEY` |
| Public URL for webhooks | `REPOSITORY_DETECTIVE_PUBLIC_URL` |
| Gitea | `REPOSITORY_DETECTIVE_GITEA_URL`, `REPOSITORY_DETECTIVE_GITEA_TOKEN` |
| Webhook secret | `REPOSITORY_DETECTIVE_WEBHOOK_SECRET` |
| Local auth | `REPOSITORY_DETECTIVE_AUTH_MODE`, `REPOSITORY_DETECTIVE_SESSION_SECRET` |
| Database | `REPOSITORY_DETECTIVE_DATABASE_PATH` (default `./data/repository-detective.db`) |

Full reference: [docs/CONFIGURATION.md](docs/CONFIGURATION.md)

## HTTP endpoints

| Path | Auth | Notes |
|------|------|-------|
| `GET /health` | none | Orchestrator probe |
| `GET /onboard` | none | Setup wizard |
| `POST /webhook` | HMAC (`X-Gitea-Signature`) | Gitea calls this |
| `/api/v1/*` | API key header | Automation |
| `/ui/*` | API key (default) or session (`auth_mode=local`) | Operator UI |

**API key header:**

```http
X-Repository-Detective-API-Key: your-key
```

See [docs/API_ROUTES.md](docs/API_ROUTES.md). Machine-readable: [docs/openapi.yaml](docs/openapi.yaml) (`GET /api/v1/openapi.yaml`). MCP stdio bridge: `go build -o repository-detective-mcp ./cmd/repository-detective-mcp` — see [docs/MCP.md](docs/MCP.md).

## Documentation

See [docs/README.md](docs/README.md) for the full index. Agent entry points: [AGENT_QUICKSTART](docs/AGENT_QUICKSTART.md), [MCP](docs/MCP.md), [OpenClaw](docs/OPENCLAW_INTEGRATION.md).

## License

**Community (proposed):** AGPL-3.0-or-later — see [docs/LICENSING_STRATEGY.md](docs/LICENSING_STRATEGY.md).  
**Commercial / Enterprise:** paid terms — see [docs/EDITIONS.md](docs/EDITIONS.md).

License text is not yet published as a root `LICENSE` file; treat the strategy doc as planning guidance until formal SPDX text is added.
