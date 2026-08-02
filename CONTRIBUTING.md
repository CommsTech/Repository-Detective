# Contributing to Repository Detective

Thanks for helping. Canonical development is on **Gitea**; **GitHub** is the public mirror for discovery and community feedback.

| Host | URL |
|------|-----|
| Canonical (CI, wiki, primary issues) | https://git.commsnet.org/commstech/repository-detective |
| Public mirror | https://github.com/CommsTech/Repository-Detective |

## Ways to help

- File bugs, false positives, missed detections, and docs gaps
- Improve operator docs under `docs/` and `docs/wiki/`
- Share repro steps from self-hosted installs (redact secrets and private paths)

## Before you open an issue

1. Confirm you are on a recent `main` build (or note your image/tag).
2. Never paste API keys, forge tokens, `.env` contents, or real secrets.
3. Prefer: scan ID, finding fingerprint, rule ID, severity, and a minimal repro.
4. Run a quick health check: `curl -s http://127.0.0.1:8081/health`

Issue templates live under `.gitea/ISSUE_TEMPLATE/` (Gitea). On GitHub, use the same structure in free-form issues until `.github` templates are mirrored.

## Local development

```bash
git clone https://github.com/CommsTech/Repository-Detective.git
cd Repository-Detective
cp .env.example .env
cp config/config.yaml.example config/config.yaml
# Edit .env — set API key + at least one forge token
docker compose -f docker-compose.minimal.yml up -d --build
curl -s http://localhost:8080/health
```

Homelab / all-in-one default port is **8081**. See [docs/QUICKSTART.md](docs/QUICKSTART.md) and [docs/SETUP.md](docs/SETUP.md).

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
