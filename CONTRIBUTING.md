# Contributing to Repository Detective

Thanks for helping. Canonical development is on **Gitea**; **GitHub** is the public mirror for discovery and **public feedback**.

| Host | Role | URL |
|------|------|-----|
| Gitea | Canonical CI, wiki, maintainer development | https://git.commsnet.org/commstech/Repository-Detective |
| GitHub | Public mirror + **public issue reports** | https://github.com/CommsTech/Repository-Detective |

## Where to report

| Kind | Where |
|------|--------|
| Bug, install problem, scanner problem, feature request | **[GitHub Issues](https://github.com/CommsTech/Repository-Detective/issues/new/choose)** (templates provided) |
| Security vulnerability in Repository Detective | **[SECURITY.md](SECURITY.md)** — private advisory preferred; never paste exploit details in public issues |
| Maintainer / internal ops templates | Gitea `.gitea/ISSUE_TEMPLATE/` |

## Before you open an issue

1. Confirm you are on a recent `main` build (or note your image/tag).
2. Never paste API keys, forge tokens, `.env` contents, or real secrets.
3. Prefer: scan ID, finding fingerprint, rule ID, severity, and a minimal repro.
4. Run a quick health check: `curl -s http://127.0.0.1:8081/health`

## Local development

```bash
git clone https://github.com/CommsTech/Repository-Detective.git
cd Repository-Detective
cp .env.example .env
cp config/config.yaml.example config/config.yaml
# Edit .env — set API key; add Gitea URL/token/webhook when connecting a forge.
# AI is optional — leave REPOSITORY_DETECTIVE_ENABLE_LLM_AUDITORS=false
docker compose pull && docker compose up -d
curl -s http://127.0.0.1:8081/health
```

Recommended install uses port **8081**. Minimal compose (`docker-compose.minimal.yml`, port 8080) is an advanced/local-build option. See [docs/QUICKSTART.md](docs/QUICKSTART.md) and [docs/SETUP.md](docs/SETUP.md).

Tests (match `go.mod`):

```bash
docker run --rm -v "$PWD":/src -w /src golang:1.25-bookworm go test ./... -count=1
```

## Pull requests

- Prefer small, focused changes with a clear “why”
- Do not commit `.env`, `config/config.yaml`, `data/*.db`, or credentials
- Update docs when behavior or operator steps change
- By contributing, you agree your changes are licensed under **AGPL-3.0-or-later** (see [LICENSE](LICENSE))

## Security reports

See [SECURITY.md](SECURITY.md). Do not open public issues for unfixed vulnerabilities with exploit detail.

## Code of conduct

Be respectful. This is a security-oriented tool — treat findings and customer code with care.
