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
  <a href="https://git.commsnet.org/commstech/-/packages/container/repository-detective/v0.1.0-beta.3">
    <img src="https://img.shields.io/badge/release-v0.1.0--beta.3-brightgreen?style=flat" alt="Current beta release">
  </a>
  <a href="https://git.commsnet.org/commstech/-/packages">
    <img src="https://img.shields.io/badge/container-gitea%20packages-609926?style=flat&logo=gitea&logoColor=white" alt="Gitea packages">
  </a>
  <a href="https://github.com/CommsTech/Repository-Detective/pkgs/container/repository-detective">
    <img src="https://img.shields.io/badge/ghcr-mirror-2496ED?style=flat&logo=docker&logoColor=white" alt="GHCR mirror">
  </a>
  <a href="LICENSE">
    <img src="https://img.shields.io/badge/license-AGPL--3.0--or--later-blue?style=flat" alt="License: AGPL-3.0-or-later">
  </a>
  <a href="docs/E2E_GITEA_ACCEPTANCE.md">
    <img src="https://img.shields.io/badge/E2E%20tested-Gitea%201.22.3-0E7C86?style=flat&logo=gitea&logoColor=white" alt="E2E tested: Gitea 1.22.3">
  </a>
  <a href="https://pkg.go.dev/git.commsnet.org/commstech/repository-detective">
    <img src="https://img.shields.io/badge/go-1.25-00ADD8?style=flat&logo=go&logoColor=white" alt="Go 1.25">
  </a>
  <a href="docs/DOCKER.md">
    <img src="https://img.shields.io/badge/platforms-linux%2Famd64-lightgrey?style=flat&logo=linux&logoColor=white" alt="Platforms: linux/amd64">
  </a>
  <a href="docs/PUBLIC_BETA.md">
    <img src="https://img.shields.io/badge/edition-public%20community%20beta-brightgreen?style=flat" alt="Public community beta">
  </a>
</p>

---

## Community public beta

| | |
|---|---|
| **Edition** | Community public beta — single-operator self-host |
| **Forge** | Gitea-first (GitHub manual/bulk scans optional; issue filing less proven) |
| **UI auth** | API-key mode by default; optional local login (`auth_mode=local`) |
| **Not yet** | SaaS, multi-tenant, billing, auto-merge, third-party auto-submit |
| **Try it** | [Public beta guide](docs/PUBLIC_BETA.md) · [Quickstart](docs/QUICKSTART.md) · [Contributing](CONTRIBUTING.md) |
| **Report a bug** | [GitHub Issues](https://github.com/CommsTech/Repository-Detective/issues/new/choose) · Security: [SECURITY.md](SECURITY.md) |
| **Editions docs** | [Community](docs/COMMUNITY_EDITION.md) · [Commercial](docs/COMMERCIAL_ENTERPRISE.md) · [Editions overview](docs/EDITIONS.md) |

> **Naming:** The product is **Repository Detective**. Use `REPOSITORY_DETECTIVE_*` env vars and `X-Repository-Detective-API-Key`. See [docs/NAMING.md](docs/NAMING.md).

| Host | Role |
|------|------|
| [Gitea](https://git.commsnet.org/commstech/Repository-Detective.git) | Canonical — CI, wiki, day-to-day development |
| [GitHub](https://github.com/CommsTech/Repository-Detective.git) | Public mirror for discovery and **public feedback** |

Sync policy: [docs/GITHUB_MIRROR.md](docs/GITHUB_MIRROR.md).

**Docs:** [Public beta guide](docs/PUBLIC_BETA.md) · [Quickstart](docs/QUICKSTART.md) · [Wiki source (`docs/wiki/`)](docs/wiki/Home.md) · [GitHub Wiki](https://github.com/CommsTech/Repository-Detective/wiki) (after first publish) · [Gitea wiki](https://git.commsnet.org/commstech/repository-detective/wiki)

## Recommended Installation

**Start here:** [docs/QUICKSTART.md](docs/QUICKSTART.md) — one path from clone → configure Gitea → first scan.  
Full walkthrough: [docs/SETUP.md](docs/SETUP.md). Public beta notes: [docs/PUBLIC_BETA.md](docs/PUBLIC_BETA.md).

**Operator docs:** [docs/README.md](docs/README.md) · [Dashboard](docs/DASHBOARD_GUIDE.md) · [Auth (local)](docs/AUTH_LOCAL.md) · [Privacy](docs/PRIVACY_AND_DATA_PROTECTION.md)

The published tree is a sanitized install base. **Only examples ship** (`.env.example`, `config/*.example.yaml`). Operator secrets (`.env`), local config (`config/config.yaml`), and the SQLite database under `data/` are gitignored and must stay private on your host. Gate: `./scripts/check-public-release-secrets.sh`.

Canonical defaults: **Docker Compose** (`docker-compose.yml`), published **all-in-one** image, port **8081**, **AI optional** (LLM auditors off by default).

```bash
git clone https://github.com/CommsTech/Repository-Detective.git && cd Repository-Detective
cp .env.example .env   # set REPOSITORY_DETECTIVE_API_KEY; add Gitea URL/token/webhook for forge use
docker compose pull && docker compose up -d
curl http://127.0.0.1:8081/health
```

Then open http://127.0.0.1:8081/onboard — image: `git.commsnet.org/commstech/repository-detective:all-in-one` ([DOCKER.md](docs/DOCKER.md); GHCR is a public mirror).

### Advanced Installation Options

| Option | When to use |
|--------|-------------|
| `docker compose up -d --build` | Develop against local source (~long build) |
| `docker-compose.minimal.yml` (port **8080**) | Lightweight local build without published image |
| `docker-compose.offline.yml` | Air-gapped / preloaded tar |
| Host-network / Traefik overlays | Special networking — [docs/NETWORKING.md](docs/NETWORKING.md) |
| External DB / runner topology | [docs/DATABASE.md](docs/DATABASE.md) · [docs/RUNNER_DELEGATION.md](docs/RUNNER_DELEGATION.md) |

**AI agents (OpenClaw, Cursor, etc.):** [docs/AGENT_QUICKSTART.md](docs/AGENT_QUICKSTART.md) · [docs/MCP.md](docs/MCP.md) · [docs/OPENCLAW_INTEGRATION.md](docs/OPENCLAW_INTEGRATION.md) · [docs/openapi.yaml](docs/openapi.yaml)

## What it does

- Scans changed files on push; scans PR diff files on pull requests
- Runs **deterministic checks first**: static rules, [Trivy](https://github.com/aquasecurity/trivy), [Grype](https://github.com/anchore/grype), golangci-lint, ruff, shellcheck
- **AI is optional** — LLM auditors off by default (`REPOSITORY_DETECTIVE_ENABLE_LLM_AUDITORS=false`); local LLM (Ollama / OpenAI-compatible) is the recommended privacy-preserving option when enabled
- Creates Gitea issues with severity, file, line, code snippet, and PoC when available
- Remediation planner yes; **remediation PRs off by default**. Issue dedup is fingerprint + forge mapping (SQLite)
- **No auto-merge** and **no automatic third-party issue submission**
- Policy / commit-status outcomes describe compliance with an **owner-defined policy**, not that a repository is “safe” or “secure”
- **AI recommendations** (optional, off by default) — provider-neutral advisory layer with CAH gating; see [docs/AI_RECOMMENDATIONS.md](docs/AI_RECOMMENDATIONS.md)
- **Issue providers:** Gitea supported; GitHub code path exists but RC-unproven; GitLab not implemented — [docs/ISSUE_PROVIDERS.md](docs/ISSUE_PROVIDERS.md)
- Release maturity: [docs/release/RC_ACCEPTANCE_BASELINE.md](docs/release/RC_ACCEPTANCE_BASELINE.md)

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

**Community:** [AGPL-3.0-or-later](LICENSE) — see also [NOTICE](NOTICE) and [docs/LICENSING_STRATEGY.md](docs/LICENSING_STRATEGY.md).  
**Commercial / Enterprise:** paid terms — see [docs/EDITIONS.md](docs/EDITIONS.md).
